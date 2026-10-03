package stats

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/database/entity"
	watchedFeature "github.com/sbondCo/Watcharr/feature/watched"
	"github.com/sbondCo/Watcharr/internal/testutil"
	"github.com/sbondCo/Watcharr/media/tmdb"
	appRouter "github.com/sbondCo/Watcharr/router"
)

func TestActivityAveragesDistinctTitlesAndMilestonesIncludeAllRepeats(t *testing.T) {
	day := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	records := make([]*watchedRecord, 12)
	for i := range records {
		r := &watchedRecord{
			content: &entity.Content{Title: fmt.Sprintf("Title %02d", i), Type: entity.MOVIE},
			plays:   []time.Time{day, day.Add(time.Hour)},
		}
		r.watched.ID = uint(i + 1)
		records[i] = r
	}
	records[0].watched.Rating = 10
	records[1].watched.Rating = 2
	records[0].plays = append(records[0].plays, day.Add(2*time.Hour))
	activity := buildActivity(records)
	if activity.Weeks[0].Plays != 25 || activity.Weeks[0].UniqueTitles != 12 || activity.Weeks[0].AverageRating != 6 || activity.Months[0].AverageRating != 6 {
		t.Fatalf("repeat watches must not weight rating averages: %#v", activity)
	}
	milestones := buildMilestones(records, nil)
	if len(milestones.MostWatched) != 12 || milestones.MostWatched[0].Title != "Title 00" || milestones.MostWatched[0].Plays != 3 || milestones.MostWatched[1].Title != "Title 01" {
		t.Fatalf("repeat-watch milestones should include every qualifying title with stable ties: %#v", milestones.MostWatched)
	}
}

func TestReleaseBreakdownExcludesLaterAndUnknownReleaseYears(t *testing.T) {
	records := make([]*watchedRecord, 0, 4)
	for _, year := range []int{2024, 2025, 2026, 0} {
		content := &entity.Content{Type: entity.MOVIE}
		if year != 0 {
			release := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
			content.ReleaseDate = &release
		}
		records = append(records, &watchedRecord{content: content})
	}
	breakdown := buildBreakdown(records, records, Query{Scope: ScopeYear, Year: 2025}, nil)
	if breakdown.Release[0].Count != 1 || breakdown.Release[1].Count != 1 {
		t.Fatalf("only selected-year and earlier releases belong in the pie: %#v", breakdown.Release)
	}
}

func TestGetStatsUsesEffectiveDatesAndSeparatesRewatches(t *testing.T) {
	db := testutil.SetupDB(t)
	private := false
	owner := entity.User{
		Username:     "owner",
		Password:     "password",
		UserSettings: entity.UserSettings{Private: &private},
	}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}

	currentRelease := date("2025-06-01")
	oldRelease := date("2020-06-01")
	current := entity.Content{TmdbID: 101, Title: "Current", Type: entity.MOVIE, ReleaseDate: &currentRelease, VoteAverage: 8.1, VoteCount: 100}
	old := entity.Content{TmdbID: 202, Title: "Older", Type: entity.MOVIE, ReleaseDate: &oldRelease, VoteAverage: 7.2, VoteCount: 200}
	planned := entity.Content{TmdbID: 303, Title: "Planned", Type: entity.MOVIE, ReleaseDate: &currentRelease, VoteAverage: 9.1, VoteCount: 300}
	for _, content := range []*entity.Content{&current, &old, &planned} {
		if err := db.Create(content).Error; err != nil {
			t.Fatal(err)
		}
	}

	currentWatched := entity.Watched{UserID: owner.ID, ContentID: &current.ID, Status: entity.FINISHED, Rating: 9, Thoughts: "review"}
	oldWatched := entity.Watched{UserID: owner.ID, ContentID: &old.ID, Status: entity.FINISHED, Rating: 5}
	plannedWatched := entity.Watched{UserID: owner.ID, ContentID: &planned.ID, Status: entity.PLANNED}
	for _, watched := range []*entity.Watched{&currentWatched, &oldWatched, &plannedWatched} {
		if err := db.Create(watched).Error; err != nil {
			t.Fatal(err)
		}
	}

	legacyActivity := entity.Activity{UserID: owner.ID, WatchedID: oldWatched.ID, Type: entity.STATUS_CHANGED, Data: `"FINISHED"`, CountAsPlay: true}
	legacyActivity.CreatedAt = date("2024-12-31")
	activities := []entity.Activity{
		{UserID: owner.ID, WatchedID: currentWatched.ID, Type: entity.ADDED_WATCHED, Data: `{"status":"FINISHED"}`, CountAsPlay: true, CustomDate: dateTime("2025-01-03")},
		{UserID: owner.ID, WatchedID: currentWatched.ID, Type: entity.STATUS_CHANGED, Data: `"FINISHED"`, CountAsPlay: true, CustomDate: dateTime("2025-02-03")},
		{UserID: owner.ID, WatchedID: oldWatched.ID, Type: entity.ADDED_WATCHED, Data: `{"status":"FINISHED"}`, CountAsPlay: true, CustomDate: dateTime("2025-03-03")},
		{UserID: owner.ID, WatchedID: plannedWatched.ID, Type: entity.ADDED_WATCHED, Data: `{"status":"PLANNED"}`, CustomDate: dateTime("2025-04-03")},
		legacyActivity,
	}
	for _, activity := range activities {
		if err := db.Create(&activity).Error; err != nil {
			t.Fatal(err)
		}
	}

	response, err := NewService(db, nil).GetStats(owner.ID, Query{Scope: ScopeYear, Year: 2025})
	if err != nil {
		t.Fatal(err)
	}
	if response.Summary.Titles != 2 || response.Summary.Plays != 3 {
		t.Fatalf("summary = %#v, want two titles and three plays", response.Summary)
	}
	if response.Breakdown.Plays[1].Count != 2 {
		t.Fatalf("rewatches = %#v, want two including a prior-year watch", response.Breakdown.Plays)
	}
	if response.Breakdown.Release[0].Count != 1 || response.Breakdown.Release[1].Count != 1 {
		t.Fatalf("release breakdown = %#v, want one current and one older", response.Breakdown.Release)
	}
	if response.Breakdown.WatchlistAdditions != 1 {
		t.Fatalf("watchlist additions = %d, want one", response.Breakdown.WatchlistAdditions)
	}
	if len(response.Activity.Weeks) != 53 {
		t.Fatalf("weeks = %#v, want all calendar weeks including empty weeks", response.Activity.Weeks)
	}
	if len(response.History) != 2 {
		t.Fatalf("history = %#v, want 2024 and 2025", response.History)
	}
	if len(response.Decades) != 0 {
		t.Fatalf("yearly decades = %#v, want none", response.Decades)
	}

	lifetime, err := NewService(db, nil).GetStats(owner.ID, Query{Scope: ScopeLifetime})
	if err != nil {
		t.Fatal(err)
	}
	if len(lifetime.HighestRated.Current) != 1 {
		t.Fatalf("lifetime highest rated = %#v, want one title rated above 8/10", lifetime.HighestRated.Current)
	}
	if len(lifetime.Decades) != 1 {
		t.Fatalf("lifetime decades = %#v, want one release decade", lifetime.Decades)
	}
}

func TestTVStatsCountOnlyWholeShowPlays(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "episode-owner", Password: "password"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	release := date("2019-01-01")
	show := entity.Content{TmdbID: 505, Title: "Episode Show", Type: entity.SHOW, ReleaseDate: &release, NumberOfEpisodes: 8, Runtime: 42}
	if err := db.Create(&show).Error; err != nil {
		t.Fatal(err)
	}
	watched := entity.Watched{UserID: owner.ID, ContentID: &show.ID, Status: entity.FINISHED}
	if err := db.Create(&watched).Error; err != nil {
		t.Fatal(err)
	}
	episode := entity.WatchedEpisode{
		UserID:        owner.ID,
		WatchedID:     watched.ID,
		SeasonNumber:  1,
		EpisodeNumber: 2,
		Rating:        9,
	}
	episode.CreatedAt = date("2024-12-01")
	if err := db.Create(&episode).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&entity.Activity{
		UserID:      owner.ID,
		WatchedID:   watched.ID,
		Type:        entity.ADDED_WATCHED,
		Data:        `{"status":"FINISHED"}`,
		CountAsPlay: true,
		CustomDate:  dateTime("2025-01-01"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&entity.Activity{
		UserID:     owner.ID,
		WatchedID:  watched.ID,
		Type:       entity.EPISODE_ADDED_JF,
		Data:       `{"season":1,"episode":2,"rating":9}`,
		CustomDate: dateTime("2025-02-01"),
	}).Error; err != nil {
		t.Fatal(err)
	}

	response, err := NewService(db, nil).GetStats(owner.ID, Query{Scope: ScopeYear, Year: 2025, Media: "tv"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Summary.Shows != 1 || response.Summary.Plays != 1 {
		t.Fatalf("TV must count whole-show plays, not episode events: %#v", response.Summary)
	}
	if response.HighsLows.Longest.Runtime != 42 {
		t.Fatalf("TV runtime must be episode runtime")
	}
	if err := db.Model(&entity.Activity{}).Where("watched_id = ? AND count_as_play = ?", watched.ID, true).Update("count_as_play", false).Error; err != nil {
		t.Fatal(err)
	}
	response, err = NewService(db, nil).GetStats(owner.ID, Query{Scope: ScopeYear, Year: 2025, Media: "tv"})
	if err != nil || response.Summary.Titles != 0 {
		t.Fatalf("episode viewing cannot count as a whole-show watch: %#v %v", response.Summary, err)
	}

}

func TestStatsMetadataFailureKeepsLocalResults(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "partial-owner", Password: "password"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	release := date("2025-01-01")
	content := entity.Content{TmdbID: 606, Title: "Unavailable metadata", Type: entity.MOVIE, ReleaseDate: &release}
	if err := db.Create(&content).Error; err != nil {
		t.Fatal(err)
	}
	watched := entity.Watched{UserID: owner.ID, ContentID: &content.ID, Status: entity.FINISHED}
	if err := db.Create(&watched).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&entity.Activity{
		UserID:      owner.ID,
		WatchedID:   watched.ID,
		Type:        entity.ADDED_WATCHED,
		Data:        `{"status":"FINISHED"}`,
		CountAsPlay: true,
		CustomDate:  dateTime("2025-03-01"),
	}).Error; err != nil {
		t.Fatal(err)
	}

	response, err := NewService(db, failingTMDB{}).GetStats(owner.ID, Query{Scope: ScopeYear, Year: 2025})
	if err != nil {
		t.Fatal(err)
	}
	if response.Summary.Titles != 1 || !response.Metadata.Partial || len(response.Metadata.FailedTitles) != 1 || response.Metadata.FailedTitles[0] != content.Title {
		t.Fatalf("partial response = %#v, want local title and affected title warning", response)
	}
}

type failingTMDB struct{}

func (failingTMDB) MovieDetails(tmdb.MovieDetailsOptions) (tmdb.MovieDetails, error) {
	return tmdb.MovieDetails{}, errors.New("metadata unavailable")
}

func (failingTMDB) ShowDetails(tmdb.ShowDetailsOptions) (tmdb.ShowDetails, error) {
	return tmdb.ShowDetails{}, errors.New("metadata unavailable")
}

func (failingTMDB) ShowCredits(string) (tmdb.ContentCredits, error) {
	return tmdb.ContentCredits{}, errors.New("metadata unavailable")
}

func (failingTMDB) SeasonDetails(string, string) (tmdb.SeasonDetails, error) {
	return tmdb.SeasonDetails{}, errors.New("metadata unavailable")
}

func TestPublicStatsIsOwnerScopedAndAnonymous(t *testing.T) {
	db := testutil.SetupDB(t)
	private := false
	owner := entity.User{
		Username:     "owner",
		Password:     "password",
		UserSettings: entity.UserSettings{Private: &private},
	}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	release := date("2025-01-01")
	content := entity.Content{TmdbID: 404, Title: "Public Movie", Type: entity.MOVIE, ReleaseDate: &release}
	if err := db.Create(&content).Error; err != nil {
		t.Fatal(err)
	}
	watched := entity.Watched{UserID: owner.ID, ContentID: &content.ID, Status: entity.FINISHED, Rating: 8, Thoughts: "must not be returned"}
	if err := db.Create(&watched).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&entity.Activity{UserID: owner.ID, WatchedID: watched.ID, Type: entity.ADDED_WATCHED, Data: `{"status":"FINISHED"}`, CountAsPlay: true, CustomDate: dateTime("2025-01-02")}).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	br := appRouter.NewBaseRouter(db, engine.Group("/api"), &config.ServerConfig{JWT_SECRET: "test"})
	NewRouter(br, NewService(db, nil), watchedFeature.NewService(db, nil, nil, nil, nil)).AddRoutes()
	path := "/api/public/users/" + strconv.FormatUint(uint64(owner.ID), 10) + "/owner/stats?year=2025"
	recorder := request(t, engine, path)
	if recorder.Code != http.StatusOK {
		t.Fatalf("public stats status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if len(recorder.Body.String()) == 0 || strings.Contains(recorder.Body.String(), "must not be returned") {
		t.Fatal("public stats response was empty or leaked review text")
	}
	var response StatsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Summary.Titles != 1 || response.Summary.Plays != 1 {
		t.Fatalf("public summary = %#v", response.Summary)
	}

	private = true
	if err := db.Model(&entity.User{}).Where("id = ?", owner.ID).Update("private", true).Error; err != nil {
		t.Fatal(err)
	}
	recorder = request(t, engine, path)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("private public stats status = %d, want 403", recorder.Code)
	}
}

func request(t *testing.T, engine *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	engine.ServeHTTP(recorder, request)
	return recorder
}

func date(value string) time.Time {
	parsed, _ := time.Parse("2006-01-02", value)
	return parsed
}

func dateTime(value string) *time.Time {
	parsed := date(value)
	return &parsed
}

func TestWatchlistTransitionsAndHistoryUseDistinctTitles(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "history", Password: "password"}
	db.Create(&owner)
	c := entity.Content{TmdbID: 701, Title: "History Film", Type: entity.MOVIE}
	db.Create(&c)
	w := entity.Watched{UserID: owner.ID, ContentID: &c.ID, Status: entity.PLANNED, Rating: 8, Thoughts: "review"}
	db.Create(&w)
	for _, d := range []string{"2024-12-31", "2025-01-01", "2025-04-01"} {
		a := entity.Activity{UserID: owner.ID, WatchedID: w.ID, CountAsPlay: true, CustomDate: dateTime(d)}
		db.Create(&a)
	}
	for _, d := range []string{"2024-01-01", "2025-03-01", "2025-06-01"} {
		a := entity.Activity{UserID: owner.ID, WatchedID: w.ID, Type: entity.STATUS_CHANGED, Data: `"PLANNED"`, CustomDate: dateTime(d)}
		db.Create(&a)
	}
	// Planned without a dated activity must not produce an inferred historical addition.
	unknown := entity.Content{TmdbID: 702, Title: "No activity", Type: entity.MOVIE}
	db.Create(&unknown)
	db.Create(&entity.Watched{UserID: owner.ID, ContentID: &unknown.ID, Status: entity.PLANNED})
	tv := entity.Content{TmdbID: 703, Title: "History Show", Type: entity.SHOW}
	db.Create(&tv)
	show := entity.Watched{UserID: owner.ID, ContentID: &tv.ID, Rating: 4}
	db.Create(&show)
	db.Create(&entity.Activity{UserID: owner.ID, WatchedID: show.ID, CountAsPlay: true, CustomDate: dateTime("2025-02-01")})
	s := NewService(db, nil)
	data, err := s.GetStats(owner.ID, Query{Scope: ScopeYear, Year: 2025})
	if err != nil {
		t.Fatal(err)
	}
	if data.Summary.Titles != 1 || data.Breakdown.Plays[0].Count != 0 || data.Breakdown.Plays[1].Count != 2 {
		t.Fatalf("expected one distinct movie and two rewatches: %#v", data)
	}
	if data.Breakdown.WatchlistAdditions != 1 {
		t.Fatalf("repeated Planned transitions must deduplicate: %d", data.Breakdown.WatchlistAdditions)
	}
	h := data.History[1]
	if h.Movies != 1 || h.Shows != 1 || h.AverageRating != 6 || h.Reviewed == nil || *h.Reviewed != 1 {
		t.Fatalf("history must include both media and weight titles once: %#v", h)
	}
	if len(h.Items) != 2 || len(data.History[0].Items) != 1 || !reflect.DeepEqual(h.ReviewedTitleKeys, []string{"movie:701"}) {
		t.Fatalf("history popup membership must be year-specific and distinct: %#v", data.History)
	}
	if h.Items[0].Type != "movie" || h.Items[1].Type != "tv" || h.Items[0].Rating != 8 || h.Items[1].Rating != 4 {
		t.Fatalf("history cards must retain media and ratings: %#v", h.Items)
	}
	if len(data.Watchlist) != 0 {
		t.Fatalf("watched planned titles and unknown ratings must not be recommendations")
	}
	hidden, err := s.GetStats(owner.ID, Query{Scope: ScopeLifetime, HideReviews: true})
	if err != nil || hidden.Breakdown.Reviews != nil || hidden.History[1].Reviewed != nil || len(hidden.History[1].ReviewedTitleKeys) != 0 {
		t.Fatalf("private review presence leaked: %#v %v", hidden, err)
	}
}

func TestEffectiveDateUsesUTCAndFullCalendar(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2025-01-01T00:30:00+02:00")
	if effectiveDate(created, nil).Year() != 2024 {
		t.Fatal("UTC boundary must place the event in 2024")
	}
	custom, _ := time.Parse(time.RFC3339, "2024-12-31T23:30:00-02:00")
	if effectiveDate(created, &custom).Year() != 2025 {
		t.Fatal("custom date takes precedence and is converted to UTC")
	}
	data := fillActivity(ActivityStats{}, nil, 2024, date("2025-01-01"))
	if len(data.Months) != 12 || len(data.Weeks) != 53 || data.AveragePerWeek != 0 {
		t.Fatalf("empty leap year calendar: %#v", data)
	}
}

type countingTMDB struct{ calls int }

func (p *countingTMDB) MovieDetails(tmdb.MovieDetailsOptions) (tmdb.MovieDetails, error) {
	p.calls++
	return tmdb.MovieDetails{}, errors.New("not available")
}
func (p *countingTMDB) ShowDetails(tmdb.ShowDetailsOptions) (tmdb.ShowDetails, error) {
	p.calls++
	return tmdb.ShowDetails{}, errors.New("not available")
}

func TestPublicPrivacyValidationBeforeMetadata(t *testing.T) {
	db := testutil.SetupDB(t)
	private := true
	hidden := true
	owner := entity.User{Username: "private-owner", Password: "password", UserSettings: entity.UserSettings{Private: &private, PrivateThoughts: &hidden}}
	db.Create(&owner)
	c := entity.Content{TmdbID: 800, Title: "Owner only", Type: entity.MOVIE}
	db.Create(&c)
	w := entity.Watched{UserID: owner.ID, ContentID: &c.ID, Rating: 8, Thoughts: "PRIVATE REVIEW"}
	db.Create(&w)
	db.Create(&entity.Activity{UserID: owner.ID, WatchedID: w.ID, CountAsPlay: true, CustomDate: dateTime("2025-01-01")})
	provider := &countingTMDB{}
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	br := appRouter.NewBaseRouter(db, engine.Group("/api"), &config.ServerConfig{JWT_SECRET: "test"})
	NewRouter(br, NewService(db, provider), watchedFeature.NewService(db, nil, nil, nil, nil)).AddRoutes()
	path := "/api/public/users/" + strconv.Itoa(int(owner.ID)) + "/private-owner/stats?year=2025"
	for _, p := range []string{path, strings.Replace(path, "private-owner", "wrong-name", 1)} {
		if r := request(t, engine, p); r.Code != 403 {
			t.Fatalf("expected privacy denial: %d", r.Code)
		}
	}
	if provider.calls != 0 {
		t.Fatal("metadata called before privacy validation")
	}
	db.Model(&owner).Update("private", false)
	r := request(t, engine, path)
	var response StatsResponse
	json.Unmarshal(r.Body.Bytes(), &response)
	if r.Code != 200 || response.ReviewsVisible || response.Breakdown.Reviews != nil || response.History[0].Reviewed != nil || strings.Contains(r.Body.String(), "PRIVATE REVIEW") {
		t.Fatalf("private thoughts leak: %s", r.Body.String())
	}
	if provider.calls != 1 {
		t.Fatalf("public owner data must be enriched once: %d", provider.calls)
	}
	for _, query := range []string{"year=oops", "media=game", "year=0"} {
		r := request(t, engine, strings.Split(path, "?")[0]+"?"+query)
		if r.Code != 400 {
			t.Fatalf("invalid query accepted: %s", query)
		}
	}
}

func TestMetadataRankingsDeduplicateCredits(t *testing.T) {
	release := date("1999-01-01")
	a := &watchedRecord{content: &entity.Content{TmdbID: 1, Type: entity.MOVIE, ReleaseDate: &release, VoteAverage: 9}, watched: entity.Watched{Rating: 5}, plays: []time.Time{date("2025-01-01")}}
	b := &watchedRecord{content: &entity.Content{TmdbID: 2, Type: entity.MOVIE, ReleaseDate: &release, VoteAverage: 4}, watched: entity.Watched{Rating: 9}, plays: []time.Time{date("2025-02-01")}}
	credit := personCredit{id: 1, name: "Recurring", job: "Director", department: "Directing"}
	metadata := map[string]contentMetadata{contentKey(a.content): {cast: []personCredit{credit, credit}, crew: []personCredit{credit, credit}}, contentKey(b.content): {cast: []personCredit{credit}, crew: []personCredit{credit}}}
	people := buildPeople([]*watchedRecord{a, b}, metadata)
	if len(people.Cast) != 1 || people.Cast[0].Titles != 2 || people.Cast[0].AverageRating != 7 {
		t.Fatalf("duplicate credits affect rankings: %#v", people)
	}
	highs := buildHighsLows([]*watchedRecord{a, b}, metadata)
	if highs.HighestTMDBRated.ID != 1 || highs.LowestRated.ID != 2 {
		t.Fatal("both rating extremes must use TMDB ratings")
	}
	records := make([]*watchedRecord, 12)
	for i := range records {
		clone := *a
		c := *a.content
		c.TmdbID = i + 10
		clone.content = &c
		records[i] = &clone
	}
	if len(buildRatingDifferences(records, nil).Lower) != 12 {
		t.Fatal("rating differences must not be truncated")
	}
	if len(buildDecades([]*watchedRecord{a}, nil)) != 0 {
		t.Fatal("a decade needs two rated titles")
	}
}

func TestAuthenticatedStatsAndPublicViewerKeepOwnerScope(t *testing.T) {
	db := testutil.SetupDB(t)
	owners := []entity.User{{Username: "owner", Password: "password"}, {Username: "viewer", Password: "password"}}
	for i := range owners {
		if err := db.Create(&owners[i]).Error; err != nil {
			t.Fatal(err)
		}
		c := entity.Content{TmdbID: 901 + i, Type: entity.MOVIE, Title: owners[i].Username + " movie"}
		db.Create(&c)
		w := entity.Watched{UserID: owners[i].ID, ContentID: &c.ID}
		db.Create(&w)
		db.Create(&entity.Activity{UserID: owners[i].ID, WatchedID: w.ID, CountAsPlay: true, CustomDate: dateTime("2025-01-01")})
	}
	engine := gin.New()
	br := appRouter.NewBaseRouter(db, engine.Group("/api"), &config.ServerConfig{JWT_SECRET: "test"})
	NewRouter(br, NewService(db, &successfulTMDB{}), watchedFeature.NewService(db, nil, nil, nil, nil)).AddRoutes()
	auth, err := jwt.NewWithClaims(jwt.SigningMethodHS256, entity.TokenClaims{UserID: owners[1].ID, RegisteredClaims: jwt.RegisteredClaims{IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}).SignedString([]byte("test"))
	if err != nil {
		t.Fatal(err)
	}
	if r := request(t, engine, "/api/stats?year=2025"); r.Code != 401 {
		t.Fatalf("private stats must require authentication: %d", r.Code)
	}
	for _, p := range []string{"/api/stats?year=2025", "/api/public/users/" + strconv.Itoa(int(owners[0].ID)) + "/owner/stats?year=2025"} {
		r := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, p, nil)
		req.Header.Set("Authorization", auth)
		engine.ServeHTTP(r, req)
		var data StatsResponse
		if err := json.Unmarshal(r.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		expected := "viewer movie"
		if strings.Contains(p, "public") {
			expected = "owner movie"
		}
		if r.Code != 200 || len(data.Posters) != 1 || data.Posters[0].Title != expected {
			t.Fatalf("wrong stats owner: %s", r.Body.String())
		}
		expectedKey := "movie:902"
		if strings.Contains(p, "public") {
			expectedKey = "movie:901"
		}
		if len(data.Genres) != 1 || !reflect.DeepEqual(data.Genres[0].TitleKeys, []string{expectedKey}) {
			t.Fatalf("membership must belong to requested owner, never viewer: %#v", data.Genres)
		}

	}
}

func TestRatingDifferenceAtOnePointHandlesFloat32Metadata(t *testing.T) {
	r := &watchedRecord{content: &entity.Content{TmdbID: 77, Type: entity.MOVIE, VoteAverage: 8.1}, watched: entity.Watched{Rating: 9.1}}
	result := buildRatingDifferences([]*watchedRecord{r}, nil)
	if len(result.Higher) != 1 {
		t.Fatal("an exact one-point difference must survive float32 metadata storage")
	}
	r.watched.Rating = 7.1
	if len(buildRatingDifferences([]*watchedRecord{r}, nil).Lower) != 1 {
		t.Fatal("an exact minus-one difference must be included")
	}
}

type successfulTMDB struct {
	active         atomic.Int32
	peak           atomic.Int32
	calls          atomic.Int32
	invalidOptions atomic.Bool
}

func (p *successfulTMDB) MovieDetails(o tmdb.MovieDetailsOptions) (tmdb.MovieDetails, error) {
	p.calls.Add(1)
	active := p.active.Add(1)
	defer p.active.Add(-1)
	for old := p.peak.Load(); active > old; old = p.peak.Load() {
		if p.peak.CompareAndSwap(old, active) {
			break
		}
	}
	if !o.DontRunDBCache || o.Params["append_to_response"] != "credits" {
		p.invalidOptions.Store(true)
	}
	time.Sleep(2 * time.Millisecond)
	var details tmdb.MovieDetails
	err := json.Unmarshal([]byte(`{"release_date":"2020-01-01","runtime":90,"vote_average":8.5,"vote_count":100,"genres":[{"id":1,"name":"Drama"}],"production_countries":[{"iso_3166_1":"FR","name":"France"}],"spoken_languages":[{"english_name":"French","iso_639_1":"fr"}],"production_companies":[{"id":2,"name":"Studio"}],"credits":{"cast":[{"id":3,"name":"Actor"},{"id":3,"name":"Actor"}],"crew":[{"id":4,"name":"Director","job":"Director","department":"Directing"}]}}`), &details)
	return details, err
}
func (p *successfulTMDB) ShowDetails(tmdb.ShowDetailsOptions) (tmdb.ShowDetails, error) {
	return tmdb.ShowDetails{}, errors.New("unused")
}

func TestMetadataEnrichmentIsBoundedAndReadOnly(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "enrichment", Password: "password"}
	db.Create(&owner)
	for i := 0; i < 20; i++ {
		c := entity.Content{TmdbID: 10000 + i, Title: strconv.Itoa(i), Type: entity.MOVIE}
		if err := db.Create(&c).Error; err != nil {
			t.Fatal(err)
		}
		w := entity.Watched{UserID: owner.ID, ContentID: &c.ID, Rating: 9}
		db.Create(&w)
		db.Create(&entity.Activity{UserID: owner.ID, WatchedID: w.ID, CountAsPlay: true, CustomDate: dateTime("2025-01-01")})
	}
	provider := &successfulTMDB{}
	data, err := NewService(db, provider).GetStats(owner.ID, Query{Scope: ScopeYear, Year: 2025})
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls.Load() != 20 || provider.peak.Load() > 6 || provider.invalidOptions.Load() {
		t.Fatalf("provider options/concurrency: %d calls, %d peak", provider.calls.Load(), provider.peak.Load())
	}
	if data.Genres[0].Count != 20 || data.Countries[0].Label != "France" || data.Languages[0].Label != "French" || data.Studios[0].Titles != 20 || data.People.Cast[0].Titles != 20 {
		t.Fatalf("metadata aggregates incorrect: %#v", data)
	}
	if data.Posters[0].Runtime != 90 || data.HighsLows.HighestTMDBRated.TMDBRating != 8.5 || data.Metadata.Partial {
		t.Fatal("enrichment must populate metadata comparisons")
	}
	var persisted entity.Content
	db.First(&persisted)
	if persisted.Runtime != 0 || persisted.VoteAverage != 0 {
		t.Fatal("stats enrichment must not mutate the database")
	}
}

func TestStatsMembershipMatchesDistinctWatchedTitles(t *testing.T) {
	a := &watchedRecord{content: &entity.Content{TmdbID: 1, Type: entity.MOVIE}, watched: entity.Watched{Rating: 9}, plays: []time.Time{date("2025-01-01"), date("2025-01-02")}}
	b := &watchedRecord{content: &entity.Content{TmdbID: 2, Type: entity.MOVIE}, watched: entity.Watched{Rating: 7}, plays: []time.Time{date("2024-01-01")}}
	c := &watchedRecord{content: &entity.Content{TmdbID: 3, Type: entity.SHOW}, watched: entity.Watched{Rating: 5}, plays: []time.Time{date("2025-01-01")}}
	credit := personCredit{id: 42, name: "Actor", job: "Director", department: "Directing"}
	metadata := map[string]contentMetadata{}
	for _, r := range []*watchedRecord{a, b, c} {
		metadata[contentKey(r.content)] = contentMetadata{genres: []string{"Drama", "Drama"}, countries: []string{"France", "France"}, languages: []string{"French", "French"}, cast: []personCredit{credit, credit}, crew: []personCredit{credit, credit}, studios: []personCredit{credit, credit}}
	}
	records := []*watchedRecord{a, b, a, c}
	movieRecords := filterMedia(records, "movie")
	want := []string{"movie:1", "movie:2"}
	people := buildPeople(movieRecords, metadata)
	for _, list := range [][]PersonStat{people.Cast, people.Directors, buildStudios(movieRecords, metadata), buildCrew(movieRecords, metadata)[0].Jobs[0].People} {
		if len(list) != 1 || list[0].Titles != 2 || list[0].AverageRating != 8 || !reflect.DeepEqual(list[0].TitleKeys, want) {
			t.Fatalf("contributor membership duplicates: %#v", list)
		}
	}
	for _, list := range [][]BarStat{buildBars(movieRecords, metadata, true), buildBars(movieRecords, metadata, false), buildLanguageBars(movieRecords, metadata)} {
		if len(list) != 1 || list[0].Count != 2 || !reflect.DeepEqual(list[0].TitleKeys, want) {
			t.Fatalf("category membership duplicates: %#v", list)
		}
	}
	scoped := filterRecordsForYear(filterMedia([]*watchedRecord{a, b, c}, "movie"), 2025)
	genres := buildBars(scoped, metadata, true)
	if genres[0].Count != 1 || !reflect.DeepEqual(genres[0].TitleKeys, []string{"movie:1"}) {
		t.Fatalf("year/media membership isolation: %#v", genres)
	}
	if len(buildBars(scoped, map[string]contentMetadata{}, true)) != 0 {
		t.Fatal("failed metadata must not invent category memberships")
	}
}

func TestLifetimeFavoritesAndDecadesAreUncappedAboveEight(t *testing.T) {
	records := make([]*watchedRecord, 16)
	total := 0.0
	for i := range records {
		rating := 8.1
		if i == 0 {
			rating = 8
		}
		if i == 1 {
			rating = 0
		}
		if i >= 10 {
			rating = 9
		}
		r := &watchedRecord{content: &entity.Content{TmdbID: i + 1, Type: entity.MOVIE, Title: fmt.Sprintf("Title %02d", i), ReleaseDate: dateTime("1999-01-01")}, watched: entity.Watched{Rating: rating}, plays: []time.Time{date("2025-01-01")}}
		records[i] = r
		total += rating
	}
	favorites := buildHighestRated(records, 0, nil).Current
	if len(favorites) != 14 {
		t.Fatalf("expected all 14 above 8/10: %#v", favorites)
	}
	for i, c := range favorites {
		if c.Rating <= 8 {
			t.Fatal("threshold must be strictly above 8")
		}
		if i > 0 && c.Rating == favorites[i-1].Rating && c.Title < favorites[i-1].Title {
			t.Fatal("unstable tie ordering")
		}
	}
	decades := buildDecades(records, nil)
	if len(decades) != 1 || len(decades[0].Items) != 14 || decades[0].Titles != 16 || decades[0].AverageRating != total/15 {
		t.Fatalf("decade ranking/counts must remain independent of poster threshold: %#v", decades)
	}
	yearly := buildHighestRated(records, 2025, nil)
	if len(yearly.Older) != 5 {
		t.Fatal("yearly highlights should contain five titles")
	}
	records[0].watched.Rating = 8
	records[1].watched.Rating = 7
	noFavorites := buildDecades(records[:2], nil)
	if len(noFavorites) != 1 || len(noFavorites[0].Items) != 0 {
		t.Fatal("eligible decades with no above-eight posters should remain present")
	}
}

func TestWatchlistPicksAreStableForOwnerYearAndMedia(t *testing.T) {
	records := make([]*watchedRecord, 40)
	for i := range records {
		records[i] = &watchedRecord{content: &entity.Content{TmdbID: i + 1, Type: entity.MOVIE, Title: fmt.Sprintf("Pick %02d", i), VoteAverage: float32(10 - float64(i)/10), VoteCount: 100}, watched: entity.Watched{Status: entity.PLANNED}}
		records[i].watched.ID = uint(i + 1)
	}
	q := Query{Scope: ScopeYear, Year: 2025, Media: "movie"}
	first := topWatchlist(records, 42, q)
	if len(first) != 5 {
		t.Fatal("expected five picks")
	}
	for _, c := range first {
		if c.ID > 30 {
			t.Fatal("picks should come from the best 30")
		}
	}
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}
	if !reflect.DeepEqual(first, topWatchlist(records, 42, q)) {
		t.Fatal("reload/order changed the picks")
	}
	q.Year = 2024
	if reflect.DeepEqual(first, topWatchlist(records, 42, q)) {
		t.Fatal("years should have different deterministic picks")
	}
	q.Year = 2025
	if reflect.DeepEqual(first, topWatchlist(records, 43, q)) {
		t.Fatal("owner seed should affect picks")
	}
	q.Media = "tv"
	if reflect.DeepEqual(first, topWatchlist(records, 42, q)) {
		t.Fatal("media seed should affect picks")
	}
	records[0].plays = []time.Time{date("2020-01-01")}
	records[1].content.VoteCount = 0
	records[2].watched.Status = entity.FINISHED
	small := topWatchlist(records[:5], 42, q)
	if len(small) != 2 {
		t.Fatalf("small pool must exclude seen, unknown rating/votes and non-Planned: %#v", small)
	}
	q.Scope = ScopeLifetime
	q.Year = 0
	lifetime := topWatchlist(records, 42, q)
	q.Year = 2025
	if !reflect.DeepEqual(lifetime, topWatchlist(records, 42, q)) {
		t.Fatal("lifetime key must use all rather than current year")
	}
}
