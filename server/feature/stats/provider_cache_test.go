package stats

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/internal/testutil"
	"github.com/sbondCo/Watcharr/media/tmdb"
)

type persistedTMDB struct {
	episodeTMDB
	calls atomic.Int32
	fetch func() (tmdb.MovieDetails, error)
}

func (p *persistedTMDB) MovieDetails(tmdb.MovieDetailsOptions) (tmdb.MovieDetails, error) {
	p.calls.Add(1)
	if p.fetch != nil {
		return p.fetch()
	}
	var details tmdb.MovieDetails
	err := json.Unmarshal([]byte(`{"runtime":90,"genres":[{"name":"Drama"}],"credits":{"cast":[{"id":1,"name":"Actor"}]}}`), &details)
	return details, err
}

func TestProviderCacheSurvivesRestartForAllStatsLookups(t *testing.T) {
	dir := t.TempDir()
	first := NewCachedTMDBProvider(&persistedTMDB{}, dir)
	lookup := func(p *cachedTMDBProvider) {
		t.Helper()
		movie, err := p.MovieDetails(tmdb.MovieDetailsOptions{ID: "10"})
		if err != nil || movie.Runtime != 90 {
			t.Fatalf("movie metadata: %#v %v", movie, err)
		}
		show, err := p.ShowDetails(tmdb.ShowDetailsOptions{ID: "10"})
		if err != nil || show.FirstAirDate != "2020-01-01" {
			t.Fatalf("show metadata: %#v %v", show, err)
		}
		season, err := p.SeasonDetails("10", "1")
		if err != nil || len(season.Episodes) != 2 {
			t.Fatalf("season metadata: %#v %v", season, err)
		}
		credits, err := p.EpisodeCredits("10", "1", "1")
		if err != nil || len(credits.Cast) != 2 {
			t.Fatalf("episode metadata: %#v %v", credits, err)
		}
	}
	lookup(first)
	// A nil upstream panics on any lookup, proving this new instance uses disk.
	lookup(NewCachedTMDBProvider(nil, dir))
	files, _ := os.ReadDir(dir)
	if len(files) != 4 {
		t.Fatalf("expected four provider responses, got %d", len(files))
	}
}

func TestProviderCacheServesStaleWhileRefreshIsBlockedAndCoalesces(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		options := tmdb.MovieDetailsOptions{ID: "10"}
		first := NewCachedTMDBProvider(&persistedTMDB{}, dir)
		if _, err := first.MovieDetails(options); err != nil {
			t.Fatal(err)
		}
		time.Sleep(25 * time.Hour)
		release := make(chan struct{})
		provider := &persistedTMDB{fetch: func() (tmdb.MovieDetails, error) {
			<-release
			var result tmdb.MovieDetails
			result.Runtime = 120
			return result, nil
		}}
		p := NewCachedTMDBProvider(provider, dir)
		started := time.Now()
		for range 20 {
			result, err := p.MovieDetails(options)
			if err != nil || result.Runtime != 90 || time.Since(started) != 0 {
				t.Fatalf("stale response blocked or changed: %d %v", result.Runtime, err)
			}
		}
		synctest.Wait()
		if provider.calls.Load() != 1 {
			t.Fatalf("duplicate refreshes: %d", provider.calls.Load())
		}
		close(release)
		p.refresh.Wait()
		result, err := NewCachedTMDBProvider(nil, dir).MovieDetails(options)
		if err != nil || result.Runtime != 120 {
			t.Fatalf("refreshed metadata wasn't persisted: %d %v", result.Runtime, err)
		}
	})
}

func TestProviderCacheFailureRetainsStaleAndDoesNotExtendRetention(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		opts := tmdb.MovieDetailsOptions{ID: "10"}
		if _, err := NewCachedTMDBProvider(&persistedTMDB{}, dir).MovieDetails(opts); err != nil {
			t.Fatal(err)
		}
		time.Sleep(25 * time.Hour)
		provider := &persistedTMDB{fetch: func() (tmdb.MovieDetails, error) { return tmdb.MovieDetails{}, errors.New("offline") }}
		p := NewCachedTMDBProvider(provider, dir)
		for range 3 {
			result, err := p.MovieDetails(opts)
			if err != nil || result.Runtime != 90 {
				t.Fatal("refresh failure discarded last successful metadata")
			}
			p.refresh.Wait()
		}
		if provider.calls.Load() != 1 {
			t.Fatal("failed background refresh wasn't cooled down")
		}
		time.Sleep(5*time.Minute + time.Nanosecond)
		p.MovieDetails(opts)
		p.refresh.Wait()
		if provider.calls.Load() != 2 {
			t.Fatal("refresh couldn't retry after cooldown")
		}
		time.Sleep(7 * 24 * time.Hour)
		if _, err := p.MovieDetails(opts); err == nil {
			t.Fatal("metadata older than retention must be fetched or report failure")
		}
		p.MovieDetails(opts)
		if provider.calls.Load() != 4 {
			t.Fatal("cold failures must not be cached")
		}
	})
}

func TestProviderCacheCoalescesColdRequestsAndBoundsBackgroundWorkers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		provider := &persistedTMDB{fetch: func() (tmdb.MovieDetails, error) {
			<-release
			return tmdb.MovieDetails{}, nil
		}}
		p := NewCachedTMDBProvider(provider, t.TempDir())
		var wg sync.WaitGroup
		for range 20 {
			wg.Go(func() { p.MovieDetails(tmdb.MovieDetailsOptions{ID: "10"}) })
		}
		synctest.Wait()
		if provider.calls.Load() != 1 {
			t.Fatalf("cold cache didn't coalesce requests: %d", provider.calls.Load())
		}
		close(release)
		wg.Wait()
		blocked := make(chan struct{})
		var active, peak atomic.Int32
		for i := range 30 {
			p.enqueue(string(rune('a'+i)), func() {
				a := active.Add(1)
				for old := peak.Load(); a > old; old = peak.Load() {
					if peak.CompareAndSwap(old, a) {
						break
					}
				}
				<-blocked
				active.Add(-1)
			})
		}
		synctest.Wait()
		if peak.Load() != metadataWorkers {
			t.Fatalf("expected six refresh workers, got %d", peak.Load())
		}
		close(blocked)
		p.refresh.Wait()
	})
}

func TestProviderCacheInvalidFilesAndStableKeys(t *testing.T) {
	if providerCacheKey("movie", "1", "", map[string]string{"a": "1", "b": "2"}) != providerCacheKey("movie", "1", "", map[string]string{"b": "2", "a": "1"}) {
		t.Fatal("map order changed cache key")
	}
	for _, data := range []string{"broken", `{"version":2}`, `{"version":1,"savedAt":"2099-01-01T00:00:00Z","data":{}}`, `{"version":1,"savedAt":"2000-01-01T00:00:00Z","data":{}}`} {
		p := NewCachedTMDBProvider(&persistedTMDB{}, t.TempDir())
		key := providerCacheKey("movie", "1", "", nil)
		if err := os.WriteFile(filepath.Join(p.dir, key+".json"), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		value, err := p.MovieDetails(tmdb.MovieDetailsOptions{ID: "1"})
		if err != nil || value.Runtime != 90 {
			t.Fatalf("invalid file must be a cache miss: %v", err)
		}
	}
	// Disk caching is best effort: read-only/unusable storage must not break stats.
	p := NewCachedTMDBProvider(&persistedTMDB{}, filepath.Join(t.TempDir(), "file"))
	if err := os.WriteFile(p.dir, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if value, err := p.MovieDetails(tmdb.MovieDetailsOptions{ID: "1"}); err != nil || value.Runtime != 90 {
		t.Fatal("disk failure broke provider lookup")
	}
}

func TestPersistedMetadataStillUsesCurrentOwnerStatsAndReviewPrivacy(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "cache-owner", Password: "password"}
	db.Create(&owner)
	c := entity.Content{TmdbID: 10, Title: "Film", Type: entity.MOVIE}
	db.Create(&c)
	w := entity.Watched{UserID: owner.ID, ContentID: &c.ID, Rating: 9, Thoughts: "private review words"}
	db.Create(&w)
	db.Create(&entity.Activity{UserID: owner.ID, WatchedID: w.ID, CountAsPlay: true, CustomDate: dateTime("2025-01-01")})
	dir := t.TempDir()
	q := Query{Scope: ScopeYear, Year: 2025, Media: "movie"}
	if _, err := NewService(db, NewCachedTMDBProvider(&persistedTMDB{}, dir)).GetStats(owner.ID, q); err != nil {
		t.Fatal(err)
	}
	db.Model(&w).Update("rating", 3)
	q.HideReviews = true
	data, err := NewService(db, NewCachedTMDBProvider(nil, dir)).GetStats(owner.ID, q)
	if err != nil || data.Summary.AverageRating != 3 || data.ReviewLengths != nil || data.Breakdown.Reviews != nil || data.Metadata.Partial {
		t.Fatalf("cache bypassed current rating/privacy: %#v %v", data, err)
	}
	files, _ := os.ReadDir(dir)
	for _, file := range files {
		contents, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil || strings.Contains(string(contents), owner.Username) || strings.Contains(string(contents), w.Thoughts) {
			t.Fatal("owner data leaked into metadata cache")
		}
	}
}
