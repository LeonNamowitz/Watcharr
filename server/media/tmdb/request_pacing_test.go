package tmdb

import (
	"errors"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	gocache "github.com/robfig/go-cache"
)

type tmdbRoundTripper func(*http.Request) (*http.Response, error)

func (f tmdbRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func mockTMDBRequests(t *testing.T, transport tmdbRoundTripper) {
	t.Helper()
	originalClient, originalPacer, originalStore := http.DefaultClient, tmdbRequests, ContentStore
	http.DefaultClient = &http.Client{Transport: transport}
	tmdbRequests = &requestPacer{interval: 50 * time.Millisecond}
	// Disable the real-time cleanup goroutine while using synctest's fake clock.
	ContentStore = gocache.New(24*time.Hour, 0)
	t.Cleanup(func() {
		http.DefaultClient, tmdbRequests, ContentStore = originalClient, originalPacer, originalStore
	})
}

func tmdbResponse(status int, retryAfter, body string) *http.Response {
	headers := make(http.Header)
	if retryAfter != "" {
		headers.Set("Retry-After", retryAfter)
	}
	return &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader(body))}
}

func TestTMDBRequestsTimeOutStalledConnections(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		mockTMDBRequests(t, func(r *http.Request) (*http.Response, error) {
			<-r.Context().Done()
			return nil, r.Context().Err()
		})
		started := time.Now()
		_, err := NewTMDB("test").apiRequest("/movie/10", nil)
		if err == nil || time.Since(started) != 15*time.Second {
			t.Fatalf("stalled request: elapsed=%s err=%v", time.Since(started), err)
		}
	})
}

func TestTMDBRequestsSharePacingAcrossClientsAndEndpoints(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mu sync.Mutex
		var starts []time.Time
		paths := map[string]bool{}
		mockTMDBRequests(t, func(r *http.Request) (*http.Response, error) {
			mu.Lock()
			defer mu.Unlock()
			starts = append(starts, time.Now())
			paths[r.URL.Path] = true
			if r.URL.Query().Get("language") != "en-US" || r.URL.Query().Get("query") != "example" {
				t.Error("pacing changed request parameters")
			}
			return tmdbResponse(http.StatusOK, "", `{}`), nil
		})
		clients := []*TMDB{NewTMDB("test-a"), NewTMDB("test-b")}
		endpoints := []string{"/search/movie", "/search/tv", "/movie/1", "/tv/2", "/tv/2/season/1", "/tv/2/season/1/episode/1/credits"}
		var wg sync.WaitGroup
		for i, endpoint := range endpoints {
			wg.Go(func() {
				if _, err := clients[i%len(clients)].apiRequest(endpoint, map[string]string{"query": "example"}); err != nil {
					t.Error(err)
				}
			})
		}
		wg.Wait()
		if len(starts) != len(endpoints) || len(paths) != len(endpoints) {
			t.Fatalf("got %d calls to %d endpoints", len(starts), len(paths))
		}
		sort.Slice(starts, func(i, j int) bool { return starts[i].Before(starts[j]) })
		for i := 1; i < len(starts); i++ {
			if starts[i].Sub(starts[i-1]) < 50*time.Millisecond {
				t.Fatal("concurrent clients exceeded the shared 20 requests/second pace")
			}
		}
	})
}

func TestTMDBRateLimitPausesAlreadyQueuedCalls(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		queued := make(chan struct{})
		firstRequest := make(chan struct{})
		var mu sync.Mutex
		calls := map[string][]time.Time{}
		mockTMDBRequests(t, func(r *http.Request) (*http.Response, error) {
			mu.Lock()
			calls[r.URL.Path] = append(calls[r.URL.Path], time.Now())
			first := r.URL.Path == "/3/movie/1" && len(calls[r.URL.Path]) == 1
			mu.Unlock()
			if first {
				close(firstRequest)
				<-queued
				return tmdbResponse(http.StatusTooManyRequests, "1", `{"error":"rate limited"}`), nil
			}
			return tmdbResponse(http.StatusOK, "", `{}`), nil
		})
		var wg sync.WaitGroup
		wg.Go(func() {
			if _, err := NewTMDB("test").apiRequest("/movie/1", nil); err != nil {
				t.Error(err)
			}
		})
		<-firstRequest
		wg.Go(func() {
			if _, err := NewTMDB("test").apiRequest("/search/movie", nil); err != nil {
				t.Error(err)
			}
		})
		// The search is sleeping for its normal slot before the 429 arrives.
		synctest.Wait()
		close(queued)
		wg.Wait()
		if len(calls["/3/movie/1"]) != 2 || len(calls["/3/search/movie"]) != 1 {
			t.Fatalf("unexpected retries: %v", calls)
		}
		for path, times := range calls {
			for i, at := range times {
				if path == "/3/movie/1" && i == 0 {
					continue
				}
				if at.Sub(start) < time.Second {
					t.Fatalf("%s escaped the shared Retry-After cooldown", path)
				}
			}
		}
	})
}

func TestTMDBRateLimitRetriesAreBoundedAndOtherErrorsAreNotRetried(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusNotFound, http.StatusInternalServerError, 0} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var starts []time.Time
				mockTMDBRequests(t, func(*http.Request) (*http.Response, error) {
					starts = append(starts, time.Now())
					if status == 0 {
						return nil, errors.New("network unavailable")
					}
					return tmdbResponse(status, "", "provider error"), nil
				})
				if _, err := NewTMDB("test").apiRequest("/movie/1", nil); err == nil {
					t.Fatal("provider failure must remain an error")
				}
				want := 1
				if status == http.StatusTooManyRequests {
					want = 3
					if len(starts) == want && (starts[1].Sub(starts[0]) != time.Second || starts[2].Sub(starts[1]) != 2*time.Second) {
						t.Fatalf("incorrect fallback backoff: %v", starts)
					}
					// The final 429 still pauses the next independent caller.
					before := time.Now()
					tmdbRequests.wait()
					if time.Since(before) != 4*time.Second {
						t.Fatal("exhausted retries must retain the shared cooldown")
					}
				}
				if len(starts) != want {
					t.Fatalf("got %d calls, want %d", len(starts), want)
				}
			})
		})
	}
}

func TestRateLimitDelay(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		header  string
		attempt int
		want    time.Duration
	}{
		{" 3 ", 0, 3 * time.Second},
		{"0", 0, 0},
		{now.Add(5 * time.Second).Format(http.TimeFormat), 0, 5 * time.Second},
		{now.Add(-time.Second).Format(http.TimeFormat), 0, 0},
		{"", 0, time.Second},
		{"invalid", 1, 2 * time.Second},
		{"-1", 2, 4 * time.Second},
	} {
		if got := rateLimitDelay(tc.header, tc.attempt, now); got != tc.want {
			t.Errorf("Retry-After %q: got %v, want %v", tc.header, got, tc.want)
		}
	}
}

func TestEpisodeCreditsCoalescesCacheMissesAndRetainsCache(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		requests := make(chan string, 2)
		release := make(chan struct{})
		var mu sync.Mutex
		calls := map[string]int{}
		mockTMDBRequests(t, func(r *http.Request) (*http.Response, error) {
			mu.Lock()
			calls[r.URL.Path]++
			mu.Unlock()
			requests <- r.URL.Path
			<-release
			return tmdbResponse(http.StatusOK, "", `{"cast":[{"id":1,"name":"Regular"}],"guest_stars":[{"id":2,"name":"Guest"}]}`), nil
		})
		clients := []*TMDB{NewTMDB("test-a"), NewTMDB("test-b")}
		results := make(chan ContentCredits, 12)
		var wg sync.WaitGroup
		for i := 0; i < cap(results); i++ {
			wg.Go(func() {
				credits, err := clients[i%len(clients)].EpisodeCredits("10", "1", "1")
				if err != nil {
					t.Error(err)
				}
				results <- credits
			})
		}
		<-requests
		synctest.Wait()
		// An unrelated episode must start while the first episode is blocked.
		wg.Go(func() {
			if _, err := clients[0].EpisodeCredits("10", "1", "2"); err != nil {
				t.Error(err)
			}
		})
		if path := <-requests; !strings.HasSuffix(path, "/episode/2/credits") {
			t.Fatalf("duplicate fetch instead of an independent episode: %s", path)
		}
		close(release)
		wg.Wait()
		first := <-results
		if len(first.Cast) != 1 || len(first.GuestStars) != 1 {
			t.Fatal("coalescing lost regular cast or guest stars")
		}
		for i := 1; i < cap(results); i++ {
			if !reflect.DeepEqual(first, <-results) {
				t.Fatal("simultaneous callers received different credits")
			}
		}
		if _, err := clients[0].EpisodeCredits("10", "1", "1"); err != nil {
			t.Fatal(err)
		}
		if len(calls) != 2 || calls["/3/tv/10/season/1/episode/1/credits"] != 1 {
			t.Fatalf("cold callers or cached repeat made duplicate calls: %v", calls)
		}
		// Successes retain the existing 24-hour expiry.
		time.Sleep(24*time.Hour + time.Nanosecond)
		if _, err := clients[0].EpisodeCredits("10", "1", "1"); err != nil {
			t.Fatal(err)
		}
		if calls["/3/tv/10/season/1/episode/1/credits"] != 2 {
			t.Fatal("expired credits were not refreshed")
		}
	})
}

func TestEpisodeCreditsSharesFailureAndAllowsLaterRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started, release := make(chan struct{}), make(chan struct{})
		calls := 0
		mockTMDBRequests(t, func(*http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				close(started)
				<-release
				return tmdbResponse(http.StatusInternalServerError, "", "unavailable"), nil
			}
			return tmdbResponse(http.StatusOK, "", `{}`), nil
		})
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Go(func() {
				if _, err := NewTMDB("test").EpisodeCredits("20", "1", "1"); err == nil || err.Error() != "episode credits request failed" {
					t.Errorf("unexpected shared error: %v", err)
				}
			})
		}
		<-started
		synctest.Wait()
		close(release)
		wg.Wait()
		if calls != 1 {
			t.Fatalf("a failed in-flight request was duplicated %d times", calls)
		}
		if _, err := NewTMDB("test").EpisodeCredits("20", "1", "1"); err != nil || calls != 2 {
			t.Fatalf("later caller could not retry failed credits: %d calls, %v", calls, err)
		}
	})
}
