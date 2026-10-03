package stats

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/internal/testutil"
	"github.com/sbondCo/Watcharr/media/tmdb"
)

type episodeTMDB struct{ fail bool }

func (episodeTMDB) MovieDetails(tmdb.MovieDetailsOptions) (tmdb.MovieDetails, error) {
	return tmdb.MovieDetails{}, errors.New("unused")
}
func (episodeTMDB) ShowDetails(tmdb.ShowDetailsOptions) (tmdb.ShowDetails, error) {
	var d tmdb.ShowDetails
	err := json.Unmarshal([]byte(`{"first_air_date":"2020-01-01","aggregate_credits":{"cast":[{"id":1,"name":"Regular"}]}}`), &d)
	return d, err
}
func (p episodeTMDB) SeasonDetails(string, string) (tmdb.SeasonDetails, error) {
	if p.fail {
		return tmdb.SeasonDetails{}, errors.New("unavailable")
	}
	var d tmdb.SeasonDetails
	err := json.Unmarshal([]byte(`{"episodes":[{"episode_number":1,"name":"First","still_path":"/first.jpg","air_date":"2025-01-01","guest_stars":[{"id":2,"name":"Guest"}]},{"episode_number":2,"name":"Second","air_date":"2020-01-01","guest_stars":[{"id":2,"name":"Guest"}]}]}`), &d)
	return d, err
}
func (episodeTMDB) EpisodeCredits(string, string, string) (tmdb.ContentCredits, error) {
	var d tmdb.ContentCredits
	err := json.Unmarshal([]byte(`{"cast":[{"id":1,"name":"Regular"},{"id":1,"name":"Regular"}],"guest_stars":[{"id":2,"name":"Guest"}]}`), &d)
	return d, err
}

func TestEpisodeActivityRatingsAndCast(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "tv-episodes", Password: "password"}
	other := entity.User{Username: "other-episodes", Password: "password"}
	for _, u := range []*entity.User{&owner, &other} {
		if err := db.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	show := entity.Content{TmdbID: 606, Title: "Show", Type: entity.SHOW}
	if err := db.Create(&show).Error; err != nil {
		t.Fatal(err)
	}
	w := entity.Watched{UserID: owner.ID, ContentID: &show.ID, Status: entity.WATCHING}
	if err := db.Create(&w).Error; err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 2; i++ {
		ep := entity.WatchedEpisode{UserID: owner.ID, WatchedID: w.ID, SeasonNumber: 1, EpisodeNumber: i, Status: entity.FINISHED, Rating: int8(10 - i)}
		ep.CreatedAt = date("2024-01-01")
		if err := db.Create(&ep).Error; err != nil {
			t.Fatal(err)
		}
		payload, _ := json.Marshal(map[string]any{"season": 1, "episode": i, "status": "FINISHED"})
		a := entity.Activity{UserID: owner.ID, WatchedID: w.ID, Type: entity.EPISODE_ADDED_JF, Data: string(payload), CustomDate: dateTime("2025-02-01")}
		if err := db.Create(&a).Error; err != nil {
			t.Fatal(err)
		}
	}
	// Rewatch counts as activity, but does not multiply cast credits or rating weight.
	for _, a := range []entity.Activity{
		{UserID: owner.ID, WatchedID: w.ID, Type: entity.EPISODE_STATUS_CHANGED, Data: `{"season":1,"episode":1,"status":"FINISHED"}`, CustomDate: dateTime("2025-02-02")},
		{UserID: other.ID, WatchedID: w.ID, Type: entity.EPISODE_STATUS_CHANGED, Data: `{"season":1,"episode":1,"status":"FINISHED"}`, CustomDate: dateTime("2025-03-01")},
	} {
		if err := db.Create(&a).Error; err != nil {
			t.Fatal(err)
		}
	}
	foreign := entity.WatchedEpisode{UserID: other.ID, WatchedID: w.ID, SeasonNumber: 1, EpisodeNumber: 3, Status: entity.FINISHED}
	if err := db.Create(&foreign).Error; err != nil {
		t.Fatal(err)
	}
	data, err := NewService(db, episodeTMDB{}).GetStats(owner.ID, Query{Scope: ScopeYear, Year: 2025, Media: "tv"})
	if err != nil {
		t.Fatal(err)
	}
	if data.Summary.Titles != 0 || data.Activity.Total != 3 || data.Activity.Months[1].Plays != 3 || len(data.Activity.Months[1].Items) != 2 || data.Activity.Months[1].AverageRating != 8.5 {
		t.Fatalf("episode-only activity: %#v", data.Activity)
	}
	if len(data.HighestRatedEpisodes.Current) != 1 || data.HighestRatedEpisodes.Current[0].EpisodeNumber != 1 || len(data.HighestRatedEpisodes.Older) != 0 {
		t.Fatalf("episode rankings: %#v", data.HighestRatedEpisodes)
	}
	if first := data.HighestRatedEpisodes.Current[0]; first.EpisodeName != "First" || first.StillPath != "/first.jpg" {
		t.Fatalf("episode names and thumbnails must survive stats aggregation: %#v", first)
	}
	if first := data.Activity.Months[1].Items[0]; first.EpisodeName != "First" || first.StillPath != "/first.jpg" {
		t.Fatalf("activity popups must retain episode metadata: %#v", first)
	}
	if len(data.People.Cast) != 2 {
		t.Fatalf("cast: %#v", data.People.Cast)
	}
	for _, p := range data.People.Cast {
		if p.Titles != 2 || len(p.TitleKeys) != 2 || p.AverageRating != 8.5 {
			t.Fatalf("episode credits should be distinct and use episode ratings: %#v", p)
		}
	}
	failed, err := NewService(db, episodeTMDB{fail: true}).GetStats(owner.ID, Query{Scope: ScopeYear, Year: 2025, Media: "tv"})
	if err != nil || failed.Activity.Total != 3 || !failed.Metadata.Partial || len(failed.Episodes) != 2 || failed.Episodes[0].Rating == 0 {
		t.Fatalf("metadata failure must retain episodes: %#v %v", failed, err)
	}
	lifetime, err := NewService(db, episodeTMDB{}).GetStats(owner.ID, Query{Scope: ScopeLifetime, Media: "tv"})
	if err != nil || len(lifetime.HighestRatedEpisodes.Current) != 1 {
		t.Fatalf("lifetime rating threshold: %#v %v", lifetime.HighestRatedEpisodes, err)
	}
}

func TestPieMembershipIncludesOverlappingFirstWatchesAndRewatches(t *testing.T) {
	first := date("2025-01-01")
	release := date("2025-01-01")
	r := &watchedRecord{content: &entity.Content{TmdbID: 7, Type: entity.MOVIE, ReleaseDate: &release}, plays: []time.Time{first, first.AddDate(0, 1, 0)}, firstPlay: &first}
	data := buildBreakdown([]*watchedRecord{r}, nil, Query{Scope: ScopeYear, Year: 2025}, nil)
	for _, pie := range append(data.Release[:1], data.Plays...) {
		if len(pie.TitleKeys) != 1 || pie.TitleKeys[0] != "movie:7" {
			t.Fatalf("missing membership: %#v", pie)
		}
	}
}

func TestHighestRatedEpisodesRetainOlderAndCurrentBeyondFive(t *testing.T) {
	records := []*watchedRecord{}
	for _, year := range []int{2020, 2026} {
		for i := 1; i <= 9; i++ {
			release := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
			records = append(records, &watchedRecord{
				content: &entity.Content{TmdbID: 606, Type: entity.SHOW, Title: fmt.Sprintf("Pantheon · S1E%d", i), ReleaseDate: &release},
				watched: entity.Watched{Rating: float64(9 + i%2)},
				episode: &entity.WatchedEpisode{SeasonNumber: 1, EpisodeNumber: i},
				plays:   []time.Time{date("2026-01-01")},
			})
		}
	}
	ranked := buildHighestRated(records, 2026, nil)
	if len(ranked.Older) != 9 || len(ranked.Current) != 9 {
		t.Fatalf("UI expansion needs all ranked episodes: %#v", ranked)
	}
	for _, items := range [][]MediaCard{ranked.Current, ranked.Older} {
		seen := map[int]bool{}
		for i, card := range items {
			seen[card.EpisodeNumber] = true
			if i > 0 && card.Rating > items[i-1].Rating {
				t.Fatal("ratings must remain sorted")
			}
		}
		for _, n := range []int{7, 8, 9} {
			if !seen[n] {
				t.Fatalf("episode %d missing", n)
			}
		}
	}
}

func TestHighestRatedEpisodeMinimumAppliesToEveryPeriod(t *testing.T) {
	records := []*watchedRecord{}
	for _, year := range []int{2020, 2026} {
		release := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		for _, rating := range []float64{0, 8, 8.9, 9, 10} {
			records = append(records, &watchedRecord{content: &entity.Content{Type: entity.SHOW, ReleaseDate: &release}, watched: entity.Watched{Rating: rating}, episode: &entity.WatchedEpisode{EpisodeNumber: 1}})
		}
	}
	for _, year := range []int{2026, 0} {
		ranked := buildHighestRated(records, year, nil)
		if (year == 2026 && (len(ranked.Current) != 2 || len(ranked.Older) != 2)) || (year == 0 && len(ranked.Current) != 4) {
			t.Fatalf("90/100 cutoff for year %d: %#v", year, ranked)
		}
		for _, items := range [][]MediaCard{ranked.Current, ranked.Older} {
			for _, card := range items {
				if card.Rating < 9 {
					t.Fatalf("episode below cutoff: %#v", card)
				}
			}
		}
	}
}
