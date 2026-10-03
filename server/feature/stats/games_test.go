package stats

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/database/entity"
	watchedFeature "github.com/sbondCo/Watcharr/feature/watched"
	"github.com/sbondCo/Watcharr/internal/testutil"
	"github.com/sbondCo/Watcharr/media/igdb"
	appRouter "github.com/sbondCo/Watcharr/router"
	"gorm.io/gorm"
)

type gameStatsProvider struct {
	calls int
	ids   []int
	data  map[int]igdb.GameStatsDetails
	err   error
}

func (p *gameStatsProvider) GameStatsDetails(ids []int) (map[int]igdb.GameStatsDetails, error) {
	p.calls++
	p.ids = append([]int{}, ids...)
	return p.data, p.err
}
func createStatsGame(t *testing.T, db *gorm.DB, owner uint, id int, status entity.WatchedStatus, rating float64, hours *uint) entity.Watched {
	t.Helper()
	g := entity.Game{IgdbID: id, Name: fmt.Sprintf("Game %03d", id), Genres: "Adventure|Adventure||", Platforms: "PC|Switch|", GameModes: "Single player|", Rating: 80, RatingCount: 20}
	if err := db.Create(&g).Error; err != nil {
		t.Fatal(err)
	}
	w := entity.Watched{UserID: owner, GameID: &g.ID, Status: status, Rating: rating, PlaytimeHours: hours}
	w.CreatedAt = date("2024-01-01")
	if err := db.Create(&w).Error; err != nil {
		t.Fatal(err)
	}
	return w
}
func addGameStatsActivity(t *testing.T, db *gorm.DB, w entity.Watched, kind entity.ActivityType, payload, at string, completion bool) {
	t.Helper()
	a := entity.Activity{UserID: w.UserID, WatchedID: w.ID, Type: kind, Data: payload, CountAsPlay: completion, CustomDate: dateTime(at)}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
}
func getGameStatsForTest(t *testing.T, s *Service, owner uint, q Query) StatsResponse {
	t.Helper()
	q.Media = "game"
	data, err := s.GetStats(owner, q)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestYearGameHoursOnlyIncludeFirstCompletions(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "first-completion-owner", Password: "password"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	newHours, replayHours, playingHours := uint(20), uint(30), uint(10)
	newGame := createStatsGame(t, db, owner.ID, 201, entity.FINISHED, 8, &newHours)
	replay := createStatsGame(t, db, owner.ID, 202, entity.FINISHED, 8, &replayHours)
	playing := createStatsGame(t, db, owner.ID, 203, entity.WATCHING, 8, &playingHours)
	addGameStatsActivity(t, db, newGame, entity.STATUS_CHANGED, `"FINISHED"`, "2025-01-01", true)
	addGameStatsActivity(t, db, newGame, entity.STATUS_CHANGED, `"FINISHED"`, "2025-06-01", true)
	// Insert the replay before the first completion to check chronological ordering.
	addGameStatsActivity(t, db, replay, entity.STATUS_CHANGED, `"FINISHED"`, "2025-02-01", true)
	addGameStatsActivity(t, db, replay, entity.IMPORTED_WATCHED, `{"status":"FINISHED"}`, "2024-01-01", true)
	addGameStatsActivity(t, db, playing, entity.STATUS_CHANGED, `"WATCHING"`, "2025-01-01", false)
	service := NewService(db, nil)
	for _, expected := range []struct {
		query Query
		hours float64
	}{
		{Query{Scope: ScopeYear, Year: 2025}, 20},
		{Query{Scope: ScopeYear, Year: 2024}, 30},
		{Query{Scope: ScopeLifetime}, 60},
	} {
		data := getGameStatsForTest(t, service, owner.ID, expected.query)
		if data.Summary.Hours == nil || *data.Summary.Hours != expected.hours {
			t.Fatalf("query %#v hours: got %v, want %v", expected.query, data.Summary.Hours, expected.hours)
		}
	}
}

func TestGameStatsProgressCompletionsReplayAndHours(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "game-owner", Password: "password"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	ten, zero := uint(10), uint(0)
	playing := createStatsGame(t, db, owner.ID, 101, entity.WATCHING, 8, &ten)
	finished := createStatsGame(t, db, owner.ID, 102, entity.FINISHED, 10, &zero)
	planned := createStatsGame(t, db, owner.ID, 103, entity.PLANNED, 7, &ten)
	createStatsGame(t, db, owner.ID, 104, entity.FINISHED, 0, nil)
	addGameStatsActivity(t, db, playing, entity.ADDED_WATCHED, `{"status":"PLANNED"}`, "2024-12-31", false)
	addGameStatsActivity(t, db, playing, entity.STATUS_CHANGED, `"WATCHING"`, "2025-01-01", false)
	addGameStatsActivity(t, db, playing, entity.STATUS_CHANGED, `"HOLD"`, "2025-01-01", false)
	addGameStatsActivity(t, db, playing, entity.STATUS_CHANGED_AUTO, `{"status":"WATCHING"}`, "2025-01-02", false)
	addGameStatsActivity(t, db, playing, entity.RATING_CHANGED, `8`, "2026-01-01", false)
	addGameStatsActivity(t, db, playing, entity.PLAYTIME_CHANGED, `{"hours":10}`, "2026-01-01", false)
	addGameStatsActivity(t, db, playing, entity.THOUGHTS_CHANGED, `review`, "2026-01-01", false)
	addGameStatsActivity(t, db, finished, entity.IMPORTED_WATCHED, `{"status":"FINISHED"}`, "2024-12-31", true)
	addGameStatsActivity(t, db, finished, entity.STATUS_CHANGED, `"FINISHED"`, "2025-01-01", true)
	addGameStatsActivity(t, db, finished, entity.STATUS_CHANGED, `"FINISHED"`, "2025-01-01", true)
	addGameStatsActivity(t, db, planned, entity.ADDED_WATCHED, `{"status":"PLANNED"}`, "2025-01-02", false)
	service := NewService(db, nil)
	year := getGameStatsForTest(t, service, owner.ID, Query{Scope: ScopeYear, Year: 2025})
	if year.Summary.Titles != 2 || year.Summary.Games != 2 || year.Summary.Completed != 1 || year.Summary.Plays != 2 || year.Summary.AverageRating != 9 {
		t.Fatalf("year summary: %#v", year.Summary)
	}
	if year.Summary.Hours == nil || *year.Summary.Hours != 0 {
		t.Fatalf("year hours must exclude unfinished games and games first completed in another year: %#v", year.Summary)
	}
	if year.Activity.Total != 3 || year.Games.Completions.Total != 2 || year.Games.CompletionPercentage != 50 || len(year.Activity.Months) != 12 {
		t.Fatalf("progress/completion activity: %#v", year.Games)
	}
	if year.Activity.Months[0].AverageRating != 9 || len(year.Activity.Months[0].Items) != 2 {
		t.Fatal("progress must not weight ratings or duplicate cards")
	}
	if year.Breakdown.Plays[0].Count != 0 || year.Breakdown.Plays[1].Count != 2 || len(year.Breakdown.Plays[1].TitleKeys) != 1 || len(year.Milestones.MostWatched) != 1 {
		t.Fatalf("cross-year replay: %#v", year.Breakdown.Plays)
	}
	if len(year.Watchlist) != 1 || year.Watchlist[0].ID != 103 || year.Breakdown.WatchlistAdditions != 1 {
		t.Fatal("unstarted backlog must be separate")
	}
	if year.Genres[0].Count != 2 || year.Genres[0].AverageRating != 9 || len(year.Games.Platforms) != 2 {
		t.Fatal("saved categories must be deduplicated")
	}
	body, _ := json.Marshal(year)
	for _, forbidden := range []string{"playtimeHours", "totalHours", "averageHours", "medianHours", "mostPlaytime", "leastPlaytime", "\"playtime\""} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("year response leaked hours: %s", forbidden)
		}
	}
	if year.HighsLows.HighestCommunityRated.CommunityRating != 8 || len(year.RatingDifferences.Higher) != 1 {
		t.Fatal("IGDB /100 scores must compare on /10")
	}
	expectedYears := []int{}
	for y := range map[int]bool{time.Now().UTC().Year(): true, 2025: true, 2024: true} {
		expectedYears = append(expectedYears, y)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(expectedYears)))
	if !reflect.DeepEqual(year.AvailableYears, expectedYears) {
		t.Fatalf("available years: %v", year.AvailableYears)
	}
	lifetime := getGameStatsForTest(t, service, owner.ID, Query{Scope: ScopeLifetime})
	p := lifetime.Games.Playtime
	if lifetime.Summary.Titles != 3 || lifetime.Summary.Completed != 2 || p.TotalHours != 10 || p.RecordedGames != 2 || p.AverageHours != 5 || p.MedianHours != 5 {
		t.Fatalf("lifetime totals: %#v %#v", lifetime.Summary, p)
	}
	if lifetime.HighsLows.LeastPlaytime.ID != 102 || *lifetime.HighsLows.LeastPlaytime.PlaytimeHours != 0 {
		t.Fatal("explicit zero must be retained")
	}
	if p.Distribution[0].Label != "Unrecorded" || p.Distribution[0].Count != 1 ||
		p.Distribution[1].Label != "1–9 hours" || p.Distribution[1].Count != 0 {
		t.Fatal("unrecorded hours must stay separate and zero hours must be omitted")
	}
	legacy := getGameStatsForTest(t, service, owner.ID, Query{Scope: ScopeYear, Year: 2024})
	if legacy.Summary.Titles != 2 || legacy.Summary.Completed != 1 {
		t.Fatalf("creation fallback must not invent dated completions: %#v", legacy.Summary)
	}
	if data := getGameStatsForTest(t, service, owner.ID, Query{Scope: ScopeYear, Year: 2026}); data.Summary.Titles != 0 {
		t.Fatal("bookkeeping must not establish year membership")
	}
}

func TestGameProgressStatusesEffectiveDatesAndFallbacks(t *testing.T) {
	for _, status := range []entity.WatchedStatus{entity.FINISHED, entity.WATCHING, entity.HOLD, entity.DROPPED, entity.PLANNED} {
		t.Run(string(status), func(t *testing.T) {
			w := entity.Watched{Status: entity.PLANNED, Game: &entity.Game{IgdbID: 1, Name: "Test"}}
			w.CreatedAt = date("2020-01-01")
			custom, err := time.Parse(time.RFC3339, "2025-01-01T00:30:00+02:00")
			if err != nil {
				t.Fatal(err)
			}
			a := entity.Activity{Type: entity.STATUS_CHANGED, Data: fmt.Sprintf(`{"status":%q}`, status), CustomDate: &custom}
			a.CreatedAt = date("2025-05-01")
			w.Activity = []entity.Activity{a}
			r := makeGameRecord(w, false)
			if status == entity.PLANNED {
				if len(r.progress) != 0 || len(r.backlogDates) != 1 {
					t.Fatal("planned status counted as play")
				}
			} else if len(r.progress) != 1 || r.progress[0].Year() != 2024 {
				t.Fatalf("effective UTC date: %v", r.progress)
			}
		})
	}
	w := entity.Watched{Status: entity.DROPPED, Game: &entity.Game{IgdbID: 1, Name: "Legacy"}}
	w.CreatedAt = date("2020-01-01")
	if r := makeGameRecord(w, false); len(r.progress) != 1 || r.progress[0].Year() != 2020 {
		t.Fatal("legacy creation fallback missing")
	}
	for _, kind := range []entity.ActivityType{entity.ADDED_WATCHED, entity.IMPORTED_WATCHED, entity.IMPORTED_ADDED_WATCHED, entity.STATUS_CHANGED_AUTO} {
		a := entity.Activity{Type: kind, Data: `"WATCHING"`}
		a.CreatedAt = date("2024-01-01")
		w.Activity = []entity.Activity{a}
		if r := makeGameRecord(w, false); len(r.progress) != 1 || r.progress[0].Year() != 2024 {
			t.Fatalf("non-planned %s not counted", kind)
		}
	}
}

func TestGamePlaytimeBucketsAndEmptyLibrary(t *testing.T) {
	hours := []uint{0, 1, 9, 10, 24, 25, 49, 50, 99, 100, 500}
	records := []*gameRecord{{card: MediaCard{ID: 1, Type: "game"}}}
	for i := range hours {
		records = append(records, &gameRecord{card: MediaCard{ID: i + 2, Type: "game", Title: fmt.Sprint(i), PlaytimeHours: &hours[i], Rating: float64(i)}})
	}
	p := gamePlaytime(records)
	counts := []int{}
	for _, b := range p.Distribution {
		counts = append(counts, b.Count)
	}
	labels := []string{}
	for _, b := range p.Distribution {
		labels = append(labels, b.Label)
	}
	wantLabels := []string{
		"Unrecorded",
		"1–9 hours",
		"10–24 hours",
		"25–49 hours",
		"50–99 hours",
		"100–499 hours",
		"500+ hours",
	}
	if !reflect.DeepEqual(labels, wantLabels) ||
		!reflect.DeepEqual(counts, []int{1, 2, 2, 2, 2, 1, 1}) ||
		p.RecordedGames != 11 || p.TotalHours != 867 || p.MedianHours != 25 {
		t.Fatalf("playtime buckets/totals: %#v", p)
	}
	if p.ByRating[0].Rating != 0 || len(p.ByRating[0].Items) != 1 {
		t.Fatal("unrated hours omitted")
	}
	if p := gamePlaytime(nil); p.RecordedGames != 0 || p.AverageHours != 0 || p.MedianHours != 0 || len(p.MostPlayed) != 0 {
		t.Fatal("empty hours must be finite")
	}
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "empty-games", Password: "password"}
	db.Create(&owner)
	data := getGameStatsForTest(t, NewService(db, nil), owner.ID, Query{Scope: ScopeLifetime})
	if data.Metadata.Partial || data.Summary.Titles != 0 || data.Games.Playtime == nil {
		t.Fatal("empty library must work without IGDB")
	}
}

func TestGameMetadataFallbackDeduplicationAndReadOnly(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "game-metadata", Password: "password"}
	db.Create(&owner)
	one := createStatsGame(t, db, owner.ID, 1, entity.WATCHING, 9, nil)
	createStatsGame(t, db, owner.ID, 2, entity.HOLD, 0, nil)
	provider := &gameStatsProvider{data: map[int]igdb.GameStatsDetails{1: {ID: 1, Rating: 75, RatingCount: 100, Genres: []igdb.StatsCategory{{Name: "RPG"}, {Name: "RPG"}}, Themes: []igdb.StatsCategory{{Name: "Fantasy"}}, Perspectives: []igdb.StatsCategory{{Name: "First person"}}}}, err: errors.New("one batch failed")}
	details := provider.data[1]
	if err := json.Unmarshal([]byte(`{"involved_companies":[{"company":{"id":7,"name":"Studio"},"developer":true,"publisher":true},{"company":{"id":7,"name":"Studio"},"developer":true}]}`), &details); err != nil {
		t.Fatal(err)
	}
	provider.data[1] = details
	data := getGameStatsForTest(t, NewService(db, nil, provider), owner.ID, Query{Scope: ScopeYear, Year: 2024})
	if provider.calls != 1 || !reflect.DeepEqual(provider.ids, []int{1, 2}) || !data.Metadata.Partial || !reflect.DeepEqual(data.Metadata.FailedTitles, []string{"Game 002"}) {
		t.Fatalf("partial metadata: %#v", data.Metadata)
	}
	if data.Games.Developers[0].Count != 1 || data.Games.Publishers[0].Count != 1 || data.Games.Themes[0].Label != "Fantasy" || data.Games.Perspectives[0].Label != "First person" {
		t.Fatal("rich classifications omitted/duplicated")
	}
	if data.HighsLows.HighestCommunityRated.ID != 2 || data.Posters[0].CommunityRating != 7.5 {
		t.Fatal("local scores must survive enrichment failure")
	}
	var persisted entity.Game
	db.First(&persisted, *one.GameID)
	if persisted.Rating != 80 || persisted.Genres != "Adventure|Adventure||" {
		t.Fatal("stats metadata must remain read-only")
	}
}

func TestPublicGameStatsPrivacyOwnerIsolationAndHiddenReviews(t *testing.T) {
	db := testutil.SetupDB(t)
	private, thoughts := false, true
	owner := entity.User{Username: "shared-games", Password: "password", UserSettings: entity.UserSettings{Private: &private, PrivateThoughts: &thoughts}}
	viewer := entity.User{Username: "viewer-games", Password: "password"}
	db.Create(&owner)
	db.Create(&viewer)
	w := createStatsGame(t, db, owner.ID, 41, entity.WATCHING, 9, nil)
	db.Model(&w).Update("thoughts", "GAME PRIVATE REVIEW")
	createStatsGame(t, db, viewer.ID, 42, entity.FINISHED, 2, nil)
	db.Create(&entity.Activity{UserID: viewer.ID, WatchedID: w.ID, CountAsPlay: true, CustomDate: dateTime("2025-01-01")})
	provider := &gameStatsProvider{data: map[int]igdb.GameStatsDetails{41: {ID: 41, Rating: 80}}}
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	br := appRouter.NewBaseRouter(db, engine.Group("/api"), &config.ServerConfig{JWT_SECRET: "test"})
	NewRouter(br, NewService(db, nil, provider), watchedFeature.NewService(db, nil, nil, nil, nil)).AddRoutes()
	base := fmt.Sprintf("/api/public/users/%d/shared-games/stats?media=game", owner.ID)
	r := request(t, engine, base+"&year=all")
	if r.Code != http.StatusOK || r.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("anonymous games: %s", r.Body.String())
	}
	var data StatsResponse
	if err := json.Unmarshal(r.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if data.Owner.Username != owner.Username || data.Summary.Titles != 1 || data.Posters[0].ID != 41 || data.ReviewsVisible || data.Breakdown.Reviews != nil || data.History[0].Reviewed != nil || strings.Contains(r.Body.String(), "GAME PRIVATE REVIEW") {
		t.Fatalf("owner/review isolation: %s", r.Body.String())
	}
	r = request(t, engine, base+"&year=2025")
	json.Unmarshal(r.Body.Bytes(), &data)
	if data.Summary.Titles != 0 {
		t.Fatal("foreign owner activity leaked")
	}
	auth, err := jwt.NewWithClaims(jwt.SigningMethodHS256, entity.TokenClaims{UserID: viewer.ID, RegisteredClaims: jwt.RegisteredClaims{IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}).SignedString([]byte("test"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/stats?media=game&year=all", base + "&year=all"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", auth)
		engine.ServeHTTP(rec, req)
		var response StatsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		expectedID := 42
		if strings.Contains(path, "public") {
			expectedID = 41
		}
		if rec.Code != http.StatusOK || len(response.Posters) != 1 || response.Posters[0].ID != expectedID {
			t.Fatalf("authenticated owner isolation: %s", rec.Body.String())
		}
	}
	calls := provider.calls
	db.Model(&owner).Update("private", true)
	for _, path := range []string{base, strings.Replace(base, "shared-games", "wrong-name", 1)} {
		if r := request(t, engine, path); r.Code != http.StatusForbidden {
			t.Fatalf("privacy: %d", r.Code)
		}
	}
	if provider.calls != calls {
		t.Fatal("IGDB called before privacy validation")
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(method, base, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatal("public stats must expose no mutations")
		}
	}
	if r := request(t, engine, "/api/stats?media=game&year=all"); r.Code != http.StatusUnauthorized {
		t.Fatal("private stats must require auth")
	}
}

func TestGameBacklogPicksStableAndExcludeStartedPlannedGames(t *testing.T) {
	records := []*gameRecord{}
	for i := 1; i <= 40; i++ {
		records = append(records, &gameRecord{card: MediaCard{Type: "game", ID: i, CommunityRating: 8, VoteCount: 10}})
	}
	q := Query{Scope: ScopeYear, Year: 2025}
	first := gameBacklogPicks(records, 7, q)
	if len(first) != 5 || !reflect.DeepEqual(first, gameBacklogPicks(records, 7, q)) {
		t.Fatal("backlog picks must be stable and capped")
	}
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "planned-again", Password: "password"}
	db.Create(&owner)
	w := createStatsGame(t, db, owner.ID, 77, entity.PLANNED, 8, nil)
	addGameStatsActivity(t, db, w, entity.STATUS_CHANGED, `"WATCHING"`, "2024-01-01", false)
	data := getGameStatsForTest(t, NewService(db, nil), owner.ID, Query{Scope: ScopeLifetime})
	if data.Summary.Titles != 1 || len(data.Watchlist) != 0 || data.Games.Statuses[4].Count != 1 {
		t.Fatal("started planned game must retain history and leave backlog")
	}
}
