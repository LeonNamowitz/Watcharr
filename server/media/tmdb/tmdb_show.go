package tmdb

import (
	"errors"
	"log/slog"
	"time"

	"github.com/sbondCo/Watcharr/cache"
)

type ShowDetailsOptions struct {
	// TMDB ID
	ID string
	// Country (currently used for watch providers)
	Country string
	// Request params map.
	Params map[string]string

	// If CacheContentShow should be ran or not.
	// If the caller wants to do its own caching to the db, it can use this
	// to avoid multiple calls to CacheContentShow.
	DontRunDBCache bool
}

func (t *TMDB) ShowDetails(o ShowDetailsOptions) (ShowDetails, error) {
	cacheKey := cache.CreateCacheKey(
		"ShowDetails",
		o.ID,
		o.Country,
		o.Params)
	resp := new(ShowDetails)
	if cache.GetCache(ContentStore, cacheKey, &resp) {
		slog.Debug("ShowDetails: Returning cache.")
		return *resp, nil
	}
	err := t.req("/tv/"+o.ID, o.Params, &resp)
	if err != nil {
		slog.Error("ShowDetails: Request failed!", "error", err)
		return ShowDetails{}, errors.New("request failed")
	}
	resp.WatchProvidersTransformed = transformProviders(
		&resp.WatchProviders,
		o.Country)
	// We don't want this to linger around (in cache) since we have the
	// transformed version now..
	resp.WatchProviders = nil
	if !o.DontRunDBCache {
		go t.contentProvider.CacheContentShow(*resp, true)
	}
	ContentStore.Set(cacheKey, resp, time.Hour*24)
	return *resp, nil
}

func (t *TMDB) ShowCredits(id string) (ContentCredits, error) {
	resp := new(ContentCredits)
	err := t.req("/tv/"+id+"/credits", map[string]string{}, &resp)
	if err != nil {
		slog.Error("ShowCredits: Request failed!", "error", err)
		return ContentCredits{}, errors.New("request failed")
	}
	return *resp, nil
}

func (t *TMDB) SeasonDetails(
	showId string,
	seasonNumber string,
) (SeasonDetails, error) {
	cacheKey := cache.CreateCacheKey("SeasonDetails", showId, seasonNumber)
	resp := new(SeasonDetails)
	if cache.GetCache(ContentStore, cacheKey, &resp) {
		slog.Debug("SeasonDetails: Returning cache.")
		return *resp, nil
	}
	err := t.req(
		"/tv/"+showId+"/season/"+seasonNumber,
		map[string]string{},
		&resp)
	if err != nil {
		slog.Error("SeasonDetails: Request failed!", "error", err)
		return SeasonDetails{}, errors.New("request failed")
	}
	ContentStore.Set(cacheKey, resp, time.Hour*24)
	return *resp, nil
}

// EpisodeCredits includes regular cast and guest stars for a specific episode.
func (t *TMDB) EpisodeCredits(showID, seasonNumber, episodeNumber string) (ContentCredits, error) {
	key := cache.CreateCacheKey("EpisodeCredits", showID, seasonNumber, episodeNumber)
	resp := new(ContentCredits)
	if cache.GetCache(ContentStore, key, &resp) {
		return *resp, nil
	}
	if err := t.req("/tv/"+showID+"/season/"+seasonNumber+"/episode/"+episodeNumber+"/credits", map[string]string{}, &resp); err != nil {
		return ContentCredits{}, errors.New("episode credits request failed")
	}
	ContentStore.Set(key, resp, time.Hour*24)
	return *resp, nil
}
