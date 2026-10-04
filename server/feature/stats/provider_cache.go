package stats

import (
	"crypto/sha256"
	"encoding/json"
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
	providerFreshFor = 24 * time.Hour
	providerKeepFor  = 7 * 24 * time.Hour
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
	provider statsMetadataProvider
	dir      string
	memory   *gocache.Cache
	retries  *gocache.Cache
	requests singleflight.Group
	now      func() time.Time
	mu       sync.Mutex
	pending  map[string]func()
	workers  int
	refresh  sync.WaitGroup
}

func NewCachedTMDBProvider(provider statsMetadataProvider, dir string) *cachedTMDBProvider {
	return &cachedTMDBProvider{
		provider: provider, dir: dir,
		memory:  gocache.New(providerKeepFor, 0),
		retries: gocache.New(5*time.Minute, 0),
		now:     time.Now, pending: make(map[string]func()),
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
	fresh := func() (T, error) {
		result, err, _ := p.requests.Do(key, func() (any, error) {
			var value T
			if entry, ok := p.load(key); ok && p.now().Sub(entry.SavedAt) < providerFreshFor && json.Unmarshal(entry.Data, &value) == nil {
				return value, nil
			}
			value, err := fetch()
			if err != nil {
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
		if err != nil {
			var zero T
			return zero, err
		}
		return result.(T), nil
	}
	if entry, ok := p.load(key); ok {
		var value T
		if json.Unmarshal(entry.Data, &value) == nil {
			if p.now().Sub(entry.SavedAt) >= providerFreshFor {
				p.enqueue(key, func() {
					if _, err := fresh(); err != nil {
						slog.Debug("Stats: background metadata refresh failed", "error", err)
					}
				})
			}
			return value, nil
		}
	}
	return fresh()
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
