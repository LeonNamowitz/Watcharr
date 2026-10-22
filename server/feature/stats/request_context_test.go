package stats

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/internal/testutil"
	"github.com/sbondCo/Watcharr/media/tmdb"
	appRouter "github.com/sbondCo/Watcharr/router"
	"gorm.io/gorm"
)

func TestStatsInternalFailuresAre500AndInvalidQueriesAre400(t *testing.T) {
	db := testutil.SetupDB(t)
	if err := db.Callback().Query().Before("gorm:query").Register("fail-stats-query", func(tx *gorm.DB) { tx.AddError(errors.New("private database failure")) }); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router := &Router{service: NewService(db, nil), br: &appRouter.BaseRouter{DB: db}, validator: validStatsOwner{}}
	engine.GET("/public/:id/:username", router.GetPublicStats)
	engine.GET("/stats", func(c *gin.Context) { c.Set("userId", uint(1)); router.GetPrivateStats(c) })
	for _, query := range []string{"year=2025", "year=all&media=tv", "media=game"} {
		response := request(t, engine, "/stats?"+query)
		if response.Code != 500 || strings.Contains(response.Body.String(), "private database failure") {
			t.Fatalf("internal error should be safe 500: %d %s", response.Code, response.Body.String())
		}
	}
	if response := request(t, engine, "/public/1/owner"); response.Code != 500 || strings.Contains(response.Body.String(), "private database failure") {
		t.Fatalf("public owner database failure: %d %s", response.Code, response.Body.String())
	}
	for _, query := range []string{"year=oops", "year=0", "media=invalid"} {
		if response := request(t, engine, "/stats?"+query); response.Code != 400 {
			t.Fatalf("invalid query should be 400: %d", response.Code)
		}
	}
}

func TestCancelledStatsRequestSkipsDatabaseAndProviders(t *testing.T) {
	db := testutil.SetupDB(t)
	queries := 0
	if err := db.Callback().Query().Before("gorm:query").Register("count-stats-query", func(tx *gorm.DB) { queries++ }); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, media := range []string{"movie", "tv", "game"} {
		_, err := NewService(db, nil).GetStatsContext(ctx, 1, Query{Scope: ScopeLifetime, Media: media})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled %s request: %v", media, err)
		}
	}
	if queries != 0 {
		t.Fatalf("cancelled requests issued %d queries", queries)
	}
}

func TestPeopleRatedCountsIgnoreUnratedTitlesAndDuplicateCredits(t *testing.T) {
	records := []*watchedRecord{}
	metadata := map[string]contentMetadata{}
	credit := personCredit{id: 42, name: "Contributor", job: "Director", department: "Directing"}
	for index, rating := range []float64{9, 0, 7} {
		r := &watchedRecord{content: &entity.Content{TmdbID: index + 1, Type: entity.MOVIE}, watched: entity.Watched{Rating: rating}}
		records = append(records, r)
		metadata[contentKey(r.content)] = contentMetadata{cast: []personCredit{credit, credit}, crew: []personCredit{credit, credit}, studios: []personCredit{credit, credit}}
	}
	for _, size := range []int{2, 3} {
		people := buildPeople(records[:size], metadata)
		for _, list := range [][]PersonStat{people.Cast, people.Directors, buildStudios(records[:size], metadata), buildCrew(records[:size], metadata)[0].Jobs[0].People} {
			if len(list) != 1 || list[0].Titles != size || list[0].RatedCount != size-1 {
				t.Fatalf("incorrect contributor rated count: %#v", list)
			}
		}
	}
}

func TestCacheCancellationAllowsLiveWaiterToRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := NewCachedTMDBProvider(&persistedTMDB{}, t.TempDir())
		defer cache.Close()
		ctx, cancel := context.WithCancel(context.Background())
		started := make(chan struct{})
		finished := make(chan error, 1)
		go func() {
			_, err := cachedProviderValueContext(ctx, cache, "shared", func(ctx context.Context) (int, error) { close(started); <-ctx.Done(); return 0, ctx.Err() })
			finished <- err
		}()
		<-started
		live := make(chan int, 1)
		go func() {
			value, err := cachedProviderValueContext(context.Background(), cache, "shared", func(context.Context) (int, error) { return 42, nil })
			if err != nil {
				live <- -1
			} else {
				live <- value
			}
		}()
		synctest.Wait()
		cancel()
		if err := <-finished; !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled caller: %v", err)
		}
		if value := <-live; value != 42 {
			t.Fatalf("live waiter inherited cancellation: %d", value)
		}
		if _, ok := cache.retries.Get("shared"); ok {
			t.Fatal("cancellation must not create a retry cooldown")
		}
	})
}

func TestCacheProviderTimeoutDoesNotRetryForever(t *testing.T) {
	cache := NewCachedTMDBProvider(&persistedTMDB{}, t.TempDir())
	defer cache.Close()
	calls := 0
	_, err := cachedProviderValueContext(context.Background(), cache, "timeout", func(context.Context) (int, error) { calls++; return 0, context.DeadlineExceeded })
	if !errors.Is(err, context.DeadlineExceeded) || calls != 1 {
		t.Fatalf("provider timeout: calls=%d err=%v", calls, err)
	}
}

func TestCancelledHandlerDoesNotWriteAnErrorResponse(t *testing.T) {
	db := testutil.SetupDB(t)
	router := &Router{service: NewService(db, nil)}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.Request = httptest.NewRequest(http.MethodGet, "/stats", nil).WithContext(ctx)
	c.Set("userId", uint(1))
	router.GetPrivateStats(c)
	if recorder.Body.Len() != 0 {
		t.Fatalf("wrote a response after cancellation: %s", recorder.Body.String())
	}
}

type validStatsOwner struct{}

func (validStatsOwner) ValidatePublicWatchedList(uint, string) error { return nil }

type cancelledStatsTransport func(*http.Request) (*http.Response, error)

func (f cancelledStatsTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestStatsCancellationStopsEnrichmentQueueAndHTTP(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "cancel-owner", Password: "password"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	for index := 0; index < metadataWorkers*3; index++ {
		content := entity.Content{TmdbID: 80000 + index, Title: "Cancellation fixture", Type: entity.MOVIE}
		if err := db.Create(&content).Error; err != nil {
			t.Fatal(err)
		}
		watched := entity.Watched{UserID: owner.ID, ContentID: &content.ID, Status: entity.FINISHED}
		if err := db.Create(&watched).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&entity.Activity{UserID: owner.ID, WatchedID: watched.ID, CountAsPlay: true}).Error; err != nil {
			t.Fatal(err)
		}
	}
	original := http.DefaultClient
	defer func() { http.DefaultClient = original }()
	started, stopped := make(chan struct{}, metadataWorkers), make(chan struct{}, metadataWorkers)
	var calls atomic.Int32
	http.DefaultClient = &http.Client{Transport: cancelledStatsTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		started <- struct{}{}
		<-r.Context().Done()
		stopped <- struct{}{}
		return nil, r.Context().Err()
	})}
	cache := NewCachedTMDBProvider(tmdb.NewTMDB("test"), t.TempDir())
	defer cache.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := NewService(db, cache).GetStatsContext(ctx, owner.ID, Query{Scope: ScopeLifetime, Media: "movie"})
		done <- err
	}()
	for index := 0; index < metadataWorkers; index++ {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("enrichment did not reach provider")
		}
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("stats request: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("stats request ignored cancellation")
	}
	for index := 0; index < metadataWorkers; index++ {
		select {
		case <-stopped:
		case <-time.After(5 * time.Second):
			t.Fatal("in-flight HTTP was not cancelled")
		}
	}
	if calls.Load() != int32(metadataWorkers) {
		t.Fatalf("queued enrichment continued: %d calls", calls.Load())
	}
}
