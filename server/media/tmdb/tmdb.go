package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	gocache "github.com/robfig/go-cache"
	"github.com/sbondCo/Watcharr/database/entity"
)

var ContentStore = gocache.New(time.Hour*24, time.Minute)

type ContentProvider interface {
	CacheContentShow(content ShowDetails, onlyUpdate bool) (entity.Content, error)
	CacheContentMovie(content MovieDetails, onlyUpdate bool) (entity.Content, error)
}

type TMDB struct {
	Key             string
	contentProvider ContentProvider
}

func NewTMDB(key string) *TMDB {
	return &TMDB{
		Key: key,
	}
}

func (t *TMDB) AddContentProvider(contentProvider ContentProvider) {
	t.contentProvider = contentProvider
}

func (t *TMDB) GetKey() string {
	if t.Key != "" {
		return t.Key //Config.TMDB_KEY
	}
	return "d047fa61d926371f277e7a83c9c4ff2c"
}

func (t *TMDB) apiRequest(ep string, p map[string]string) ([]byte, error) {
	slog.Debug("tmdbAPIRequest", "endpoint", ep, "params", p)
	base, err := url.Parse("https://api.themoviedb.org/3")
	if err != nil {
		return nil, errors.New("failed to parse api uri")
	}

	// Path params
	base.Path += ep

	// Query params
	params := url.Values{}
	params.Add("api_key", t.GetKey())
	params.Add("language", "en-US")
	for k, v := range p {
		params.Add(k, v)
	}

	// Add params to url
	base.RawQuery = params.Encode()

	for attempt := 0; ; attempt++ {
		tmdbRequests.wait()
		// Bound connection and body reads so one stalled title cannot hold up
		// a library scan indefinitely. Keep the shared pacing and 429 retries.
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
		if err != nil {
			cancel()
			return nil, err
		}
		res, err := http.DefaultClient.Do(request)
		if err != nil {
			cancel()
			return nil, err
		}
		if res.StatusCode == http.StatusTooManyRequests {
			// Pause all queued requests, even if this caller has exhausted retries.
			tmdbRequests.backoff(rateLimitDelay(res.Header.Get("Retry-After"), attempt, time.Now()))
		}
		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		cancel()
		if err != nil {
			return nil, err
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < maxRateLimitRetries {
			continue
		}
		if res.StatusCode != http.StatusOK {
			slog.Error("TMDB non 200 status code:", "status_code", res.StatusCode)
			return nil, errors.New(string(body))
		}
		return body, nil
	}
}

func (t *TMDB) req(ep string, p map[string]string, resp interface{}) error {
	body, err := t.apiRequest(ep, p)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(body), &resp)
	if err != nil {
		return err
	}
	return nil
}
