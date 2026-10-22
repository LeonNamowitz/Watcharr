package igdb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

type statsRoundTripper func(*http.Request) (*http.Response, error)

func (f statsRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGameStatsDetailsBatchesCachesAndPacesRequests(t *testing.T) {
	original := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = original })
	calls := 0
	var requestTimes []time.Time
	http.DefaultClient = &http.Client{Transport: statsRoundTripper(func(r *http.Request) (*http.Response, error) {
		calls++
		requestTimes = append(requestTimes, time.Now())
		if r.Method != http.MethodPost || r.URL.String() != igdbHost+"/games" || r.Header.Get("Client-ID") != "test" || r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("incorrect IGDB request: %v", r)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		query := string(body)
		if !strings.Contains(query, "themes.name") || !strings.Contains(query, "player_perspectives.name") || !strings.Contains(query, "involved_companies.publisher") || !strings.Contains(query, "limit 100;") {
			t.Fatalf("incomplete stats query: %s", query)
		}
		ids := strings.Split(strings.Split(strings.Split(query, "where id = (")[1], ")")[0], ",")
		if len(ids) > 100 {
			t.Fatal("batch exceeded 100 games")
		}
		games := []GameStatsDetails{}
		for _, value := range ids {
			id, err := strconv.Atoi(value)
			if err != nil {
				t.Fatal(err)
			}
			games = append(games, GameStatsDetails{ID: id, Rating: 85, Themes: []StatsCategory{{Name: "Fantasy"}}})
		}
		payload, _ := json.Marshal(games)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(payload))), Header: make(http.Header)}, nil
	})}
	clientID := "test"
	provider := &IGDB{ClientID: &clientID, AccessToken: "token"}
	ids := []int{}
	for id := 31000; id < 31202; id++ {
		ids = append(ids, id)
		GameStore.Delete("stats:" + strconv.Itoa(id))
	}
	t.Cleanup(func() {
		for _, id := range ids {
			GameStore.Delete("stats:" + strconv.Itoa(id))
		}
	})
	ids = append(ids, 31000, -1, 0)
	games, err := provider.GameStatsDetails(ids)
	if err != nil || len(games) != 202 || calls != 3 {
		t.Fatalf("batch result: %d games, %d calls, %v", len(games), calls, err)
	}
	for i := 1; i < len(requestTimes); i++ {
		if requestTimes[i].Sub(requestTimes[i-1]) < 240*time.Millisecond {
			t.Fatal("IGDB cache misses exceeded four requests per second")
		}
	}
	cached, err := provider.GameStatsDetails(ids)
	if err != nil || calls != 3 || !reflect.DeepEqual(cached, games) {
		t.Fatal("repeat request must use metadata cache")
	}
}

func TestGameStatsDetailsKeepsSuccessfulBatchesAndHandlesDisabledCredentials(t *testing.T) {
	original := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = original })
	calls := 0
	http.DefaultClient = &http.Client{Transport: statsRoundTripper(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 2 {
			return nil, errors.New("provider unavailable")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[{"id":32000,"rating":80}]`))}, nil
	})}
	ids := []int{}
	for id := 32000; id < 32101; id++ {
		ids = append(ids, id)
		GameStore.Delete("stats:" + strconv.Itoa(id))
	}
	t.Cleanup(func() {
		for _, id := range ids {
			GameStore.Delete("stats:" + strconv.Itoa(id))
		}
	})
	clientID := "test"
	provider := &IGDB{ClientID: &clientID, AccessToken: "token"}
	games, err := provider.GameStatsDetails(ids)
	if err == nil || len(games) != 1 || games[32000].Rating != 80 {
		t.Fatalf("successes must survive a failed batch: %v %v", games, err)
	}
	before := calls
	for _, disabled := range []*IGDB{nil, {}, {ClientID: &clientID}} {
		if _, err := disabled.GameStatsDetails([]int{1}); err == nil {
			t.Fatal("disabled IGDB must report unavailable metadata")
		}
	}
	if calls != before {
		t.Fatal("disabled credentials must not make requests")
	}
	if empty, err := provider.GameStatsDetails(nil); err != nil || len(empty) != 0 {
		t.Fatal("empty library needs no metadata lookup")
	}
}

func TestGameStatsCancellationStopsFurtherBatches(t *testing.T) {
	original := http.DefaultClient
	defer func() { http.DefaultClient = original }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	http.DefaultClient = &http.Client{Transport: statsRoundTripper(func(r *http.Request) (*http.Response, error) {
		calls++
		cancel()
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}
	clientID := "test"
	provider := &IGDB{ClientID: &clientID, AccessToken: "token"}
	ids := []int{}
	for id := 90000; id < 90201; id++ {
		ids = append(ids, id)
	}
	_, err := provider.WithContext(ctx).GameStatsDetails(ids)
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("cancelled batches: calls=%d err=%v", calls, err)
	}
	if provider.ctx != nil {
		t.Fatal("request context leaked onto shared provider")
	}
}

func TestGameStatsCancellationWhileWaitingForSharedFetch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		previousSlot := statsFetchSlot
		statsFetchSlot = make(chan struct{}, 1)
		defer func() { statsFetchSlot = previousSlot }()
		statsFetchSlot <- struct{}{}
		defer func() { <-statsFetchSlot }()
		ctx, cancel := context.WithCancel(context.Background())
		clientID := "test"
		provider := (&IGDB{ClientID: &clientID, AccessToken: "token"}).WithContext(ctx)
		done := make(chan error, 1)
		go func() { _, err := provider.GameStatsDetails([]int{99999}); done <- err }()
		synctest.Wait()
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("shared-fetch wait not cancelled: %v", err)
		}
	})
}

func TestIGDBPacingWaitCanBeCancelled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		previous := nextRequest
		nextRequest = time.Now().Add(time.Hour)
		defer func() { nextRequest = previous }()
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { _, err := waitForRequestSlotContext(ctx); done <- err }()
		synctest.Wait()
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("pacing wait not cancelled: %v", err)
		}
		if len(requestSlots) != 0 {
			t.Fatal("cancelled wait leaked a concurrency slot")
		}
	})
}
