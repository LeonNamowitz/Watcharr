package igdb

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Stats only needs classifications and community scores, not full game pages.
type StatsCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GameStatsDetails struct {
	ID           int             `json:"id"`
	Rating       float64         `json:"rating"`
	RatingCount  int             `json:"rating_count"`
	Genres       []StatsCategory `json:"genres"`
	Platforms    []StatsCategory `json:"platforms"`
	GameModes    []StatsCategory `json:"game_modes"`
	Themes       []StatsCategory `json:"themes"`
	Perspectives []StatsCategory `json:"player_perspectives"`
	Companies    []struct {
		Company   StatsCategory `json:"company"`
		Developer bool          `json:"developer"`
		Publisher bool          `json:"publisher"`
	} `json:"involved_companies"`
}

// Serializes cache fills across concurrent stats requests, including the cache
// recheck, to avoid duplicate fetches. GameStatsDetails keeps successes if a
// later batch fails so callers can fall back only for the missing games.
var statsFetchMu sync.Mutex

func (i *IGDB) GameStatsDetails(ids []int) (map[int]GameStatsDetails, error) {
	result := make(map[int]GameStatsDetails)
	if len(ids) == 0 {
		return result, nil
	}
	if i == nil || i.ClientID == nil || i.AccessToken == "" {
		return result, errors.New("IGDB is not configured")
	}
	statsFetchMu.Lock()
	defer statsFetchMu.Unlock()
	missing := make([]int, 0)
	seen := make(map[int]bool)
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		if cached, ok := GameStore.Get("stats:" + strconv.Itoa(id)); ok {
			result[id] = cached.(GameStatsDetails)
		} else {
			missing = append(missing, id)
		}
	}
	var failures []error
	for start := 0; start < len(missing); start += 100 {
		end := start + 100
		if end > len(missing) {
			end = len(missing)
		}
		values := make([]string, 0, end-start)
		for _, id := range missing[start:end] {
			values = append(values, strconv.Itoa(id))
		}
		var response []GameStatsDetails
		err := i.req(igdbHost, "/games", nil,
			`fields rating,rating_count,genres.name,platforms.name,game_modes.name,themes.name,player_perspectives.name,involved_companies.company.name,involved_companies.developer,involved_companies.publisher; where id = (`+strings.Join(values, ",")+`); limit 100;`, &response)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		for _, game := range response {
			if !seen[game.ID] {
				continue
			}
			result[game.ID] = game
			GameStore.Set("stats:"+strconv.Itoa(game.ID), game, time.Hour*24)
		}
	}
	return result, errors.Join(failures...)
}
