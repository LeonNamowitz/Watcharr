package stats

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	gocache "github.com/robfig/go-cache"
	"github.com/sbondCo/Watcharr/media/tmdb"
	"golang.org/x/sync/singleflight"
)

const (
	providerFreshFor     = 24 * time.Hour
	providerKeepFor      = 7 * 24 * time.Hour
	providerCleanupEvery = time.Hour
)

type statsMetadataProvider interface {
	TMDBProvider
	seasonProvider
	episodeCreditsProvider
}

type providerCacheEntry struct {
	Version int             `json:"version"`
	SavedAt time.Time       `json:"savedAt"`
	Data    json.RawMessage `json:"data"`
}

// Persist only public TMDB responses, never watched records or a stats response.
// A daily restart retains enrichment while current owner data is rebuilt as usual.
type cachedTMDBProvider struct {
	provider    statsMetadataProvider
	dir         string
	memory      *gocache.Cache
	retries     *gocache.Cache
	requests    singleflight.Group
	now         func() time.Time
	mu          sync.Mutex
	pending     map[string]func()
	workers     int
	refresh     sync.WaitGroup
	diskMu      sync.Mutex
	cleanupStop chan struct{}
	cleanupDone chan struct{}
	closeOnce   sync.Once
}

func NewCachedTMDBProvider(provider statsMetadataProvider, dir string) *cachedTMDBProvider {
	p := &cachedTMDBProvider{
		provider: provider, dir: dir,
		memory:  gocache.New(providerKeepFor, 0),
		retries: gocache.New(5*time.Minute, 0),
		now:     time.Now, pending: make(map[string]func()),
		cleanupStop: make(chan struct{}), cleanupDone: make(chan struct{}),
	}
	p.cleanup()
	go func() {
		defer close(p.cleanupDone)
		ticker := time.NewTicker(providerCleanupEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				p.cleanup()
			case <-p.cleanupStop:
				return
			}
		}
	}()
	return p
}

// Close stops the cache cleanup worker.
func (p *cachedTMDBProvider) Close() {
	p.closeOnce.Do(func() { close(p.cleanupStop) })
	<-p.cleanupDone
}

func (p *cachedTMDBProvider) cleanup() {
	p.memory.DeleteExpired()
	p.retries.DeleteExpired()
	// Serialize pruning with replacement so a refresh cannot be removed after
	// the cleanup worker has read its previous, expired response.
	p.diskMu.Lock()
	defer p.diskMu.Unlock()
	files, err := os.ReadDir(p.dir)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Debug("Stats: failed reading TMDB metadata cache for cleanup", "error", err)
		}
		return
	}
	for _, file := range files {
		name := file.Name()
		if !file.Type().IsRegular() || len(name) != sha256.Size*2+len(".json") || filepath.Ext(name) != ".json" {
			continue
		}
		if _, err := hex.DecodeString(name[:sha256.Size*2]); err != nil {
			continue
		}
		path := filepath.Join(p.dir, name)
		data, err := os.ReadFile(path)
		var entry providerCacheEntry
		if err != nil || json.Unmarshal(data, &entry) != nil || entry.Version != 1 || p.now().Sub(entry.SavedAt) < providerKeepFor {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			slog.Debug("Stats: failed removing expired TMDB metadata cache", "error", err)
		}
	}
}

// Parameters are sorted so equivalent requests share a stable disk cache key.
func providerCacheKey(kind, id, country string, params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(kind+":"+id+":"+country+":"+values.Encode())))
}

func (p *cachedTMDBProvider) load(key string) (providerCacheEntry, bool) {
	if value, ok := p.memory.Get(key); ok {
		entry := value.(providerCacheEntry)
		if age := p.now().Sub(entry.SavedAt); age >= 0 && age < providerKeepFor {
			return entry, true
		}
		p.memory.Delete(key)
	}
	data, err := os.ReadFile(filepath.Join(p.dir, key+".json"))
	var entry providerCacheEntry
	if err != nil || json.Unmarshal(data, &entry) != nil || entry.Version != 1 || len(entry.Data) == 0 {
		return providerCacheEntry{}, false
	}
	age := p.now().Sub(entry.SavedAt)
	if age < 0 || age >= providerKeepFor {
		return providerCacheEntry{}, false
	}
	p.memory.Set(key, entry, providerKeepFor-age)
	return entry, true
}

func (p *cachedTMDBProvider) save(key string, entry providerCacheEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(p.dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(p.dir, ".metadata-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	p.diskMu.Lock()
	defer p.diskMu.Unlock()
	return os.Rename(file.Name(), filepath.Join(p.dir, key+".json"))
}

// Drain a deduplicated queue with at most six workers, including large episode
// libraries. Requests still pass through TMDB's shared pacer and Retry-After.
func (p *cachedTMDBProvider) enqueue(key string, fetch func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, queued := p.pending[key]; queued {
		return
	}
	if _, coolingDown := p.retries.Get(key); coolingDown {
		return
	}
	p.pending[key] = fetch
	p.refresh.Add(1)
	if p.workers >= metadataWorkers {
		return
	}
	p.workers++
	go func() {
		for {
			p.mu.Lock()
			var key string
			var fetch func()
			for k, f := range p.pending {
				if f != nil {
					key, fetch = k, f
					p.pending[k] = nil // retain membership while fetching
					break
				}
			}
			if fetch == nil {
				p.workers--
				p.mu.Unlock()
				return
			}
			p.mu.Unlock()
			fetch()
			p.mu.Lock()
			delete(p.pending, key)
			p.mu.Unlock()
			p.refresh.Done()
		}
	}()
}

func cachedProviderValue[T any](p *cachedTMDBProvider, key string, fetch func() (T, error)) (T, error) {
	return cachedProviderValueContext(context.Background(), p, key, func(context.Context) (T, error) { return fetch() })
}

var errMetadataFetchCancelled = errors.New("metadata fetch owner cancelled")

func cachedProviderValueContext[T any](ctx context.Context, p *cachedTMDBProvider, key string, fetch func(context.Context) (T, error)) (T, error) {
	fresh := func(ctx context.Context) (T, error) {
		for {
			if err := ctx.Err(); err != nil {
				var zero T
				return zero, err
			}
			pending := p.requests.DoChan(key, func() (any, error) {
				if err := ctx.Err(); err != nil {
					return nil, errors.Join(errMetadataFetchCancelled, err)
				}
				var value T
				if entry, ok := p.load(key); ok && p.now().Sub(entry.SavedAt) < providerFreshFor && json.Unmarshal(entry.Data, &value) == nil {
					return value, nil
				}
				value, err := fetch(ctx)
				if ctx.Err() != nil {
					return nil, errors.Join(errMetadataFetchCancelled, ctx.Err())
				}
				if err != nil {
					if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
						return nil, err
					}
					p.retries.Set(key, true, 5*time.Minute)
					return value, err
				}
				data, err := json.Marshal(value)
				if err != nil {
					return value, err
				}
				entry := providerCacheEntry{Version: 1, SavedAt: p.now(), Data: data}
				p.memory.Set(key, entry, providerKeepFor)
				p.retries.Delete(key)
				if err := p.save(key, entry); err != nil {
					slog.Warn("Stats: failed saving TMDB metadata cache", "error", err)
				}
				return value, nil
			})
			select {
			case <-ctx.Done():
				var zero T
				return zero, ctx.Err()
			case result := <-pending:
				// A different request may have started the shared fetch and cancelled it.
				// Live waiters retry, rather than inheriting that request's cancellation.
				if errors.Is(result.Err, errMetadataFetchCancelled) && ctx.Err() == nil {
					continue
				}
				if result.Err != nil {
					var zero T
					return zero, result.Err
				}
				return result.Val.(T), nil
			}
		}
	}
	if err := ctx.Err(); err != nil {
		var zero T
		return zero, err
	}
	if entry, ok := p.load(key); ok {
		var value T
		if json.Unmarshal(entry.Data, &value) == nil {
			if p.now().Sub(entry.SavedAt) >= providerFreshFor {
				p.enqueue(key, func() {
					if _, err := fresh(context.Background()); err != nil {
						slog.Debug("Stats: background metadata refresh failed", "error", err)
					}
				})
			}
			return value, nil
		}
	}
	return fresh(ctx)
}

func (p *cachedTMDBProvider) MovieDetails(o tmdb.MovieDetailsOptions) (tmdb.MovieDetails, error) {
	return cachedProviderValue(p, providerCacheKey("movie", o.ID, o.Country, o.Params), func() (tmdb.MovieDetails, error) {
		return p.provider.MovieDetails(o)
	})
}

func (p *cachedTMDBProvider) ShowDetails(o tmdb.ShowDetailsOptions) (tmdb.ShowDetails, error) {
	return cachedProviderValue(p, providerCacheKey("tv", o.ID, o.Country, o.Params), func() (tmdb.ShowDetails, error) {
		return p.provider.ShowDetails(o)
	})
}

func (p *cachedTMDBProvider) SeasonDetails(id, season string) (tmdb.SeasonDetails, error) {
	return cachedProviderValue(p, providerCacheKey("season", id, "", map[string]string{"season": season}), func() (tmdb.SeasonDetails, error) {
		return p.provider.SeasonDetails(id, season)
	})
}

func (p *cachedTMDBProvider) EpisodeCredits(id, season, episode string) (tmdb.ContentCredits, error) {
	return cachedProviderValue(p, providerCacheKey("episode", id, "", map[string]string{"season": season, "episode": episode}), func() (tmdb.ContentCredits, error) {
		return p.provider.EpisodeCredits(id, season, episode)
	})
}

// A request-scoped view keeps cache state shared without copying mutexes.
type contextualTMDBCache struct {
	cache *cachedTMDBProvider
	ctx   context.Context
}

func bindTMDBContext(ctx context.Context, provider TMDBProvider) TMDBProvider {
	switch p := provider.(type) {
	case *tmdb.TMDB:
		return p.WithContext(ctx)
	case *cachedTMDBProvider:
		return &contextualTMDBCache{cache: p, ctx: ctx}
	default:
		return provider
	}
}

func (p *contextualTMDBCache) provider(ctx context.Context) statsMetadataProvider {
	if provider, ok := p.cache.provider.(*tmdb.TMDB); ok {
		return provider.WithContext(ctx)
	}
	return p.cache.provider
}

func (p *contextualTMDBCache) MovieDetails(o tmdb.MovieDetailsOptions) (tmdb.MovieDetails, error) {
	return cachedProviderValueContext(p.ctx, p.cache, providerCacheKey("movie", o.ID, o.Country, o.Params), func(ctx context.Context) (tmdb.MovieDetails, error) { return p.provider(ctx).MovieDetails(o) })
}
func (p *contextualTMDBCache) ShowDetails(o tmdb.ShowDetailsOptions) (tmdb.ShowDetails, error) {
	return cachedProviderValueContext(p.ctx, p.cache, providerCacheKey("tv", o.ID, o.Country, o.Params), func(ctx context.Context) (tmdb.ShowDetails, error) { return p.provider(ctx).ShowDetails(o) })
}
func (p *contextualTMDBCache) SeasonDetails(id, season string) (tmdb.SeasonDetails, error) {
	return cachedProviderValueContext(p.ctx, p.cache, providerCacheKey("season", id, "", map[string]string{"season": season}), func(ctx context.Context) (tmdb.SeasonDetails, error) {
		return p.provider(ctx).SeasonDetails(id, season)
	})
}
func (p *contextualTMDBCache) EpisodeCredits(id, season, episode string) (tmdb.ContentCredits, error) {
	return cachedProviderValueContext(p.ctx, p.cache, providerCacheKey("episode", id, "", map[string]string{"season": season, "episode": episode}), func(ctx context.Context) (tmdb.ContentCredits, error) {
		return p.provider(ctx).EpisodeCredits(id, season, episode)
	})
}
