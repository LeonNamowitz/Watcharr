package stats

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/internal/testutil"
)

func libraryRecord(id uint, status entity.WatchedStatus, activities ...entity.Activity) *watchedRecord {
	r := &watchedRecord{watched: entity.Watched{Status: status, Activity: activities}, content: &entity.Content{TmdbID: int(id), Type: entity.MOVIE, Title: "Title"}}
	r.watched.ID = id
	for _, a := range activities {
		if a.CountAsPlay {
			r.plays = append(r.plays, effectiveDate(a.CreatedAt, a.CustomDate))
		}
	}
	return r
}
func libraryActivity(id uint, kind entity.ActivityType, data, day string, play bool) entity.Activity {
	a := entity.Activity{Type: kind, Data: data, CustomDate: dateTime(day), CountAsPlay: play}
	a.ID = id
	a.CreatedAt = date("2026-01-01")
	return a
}
func statusCounts(stats LibraryStats) map[entity.WatchedStatus]int {
	counts := map[entity.WatchedStatus]int{}
	for _, group := range stats.Statuses {
		counts[group.Status] = group.Count
	}
	return counts
}

func TestLibraryStatusActivityIsYearlyDistinctAndIndependentOfPlays(t *testing.T) {
	r := libraryRecord(1, entity.WATCHING,
		libraryActivity(5, entity.STATUS_CHANGED, `"WATCHING"`, "2025-06-01", false),
		libraryActivity(1, entity.IMPORTED_ADDED_WATCHED, `{"status":"PLANNED"}`, "2025-01-01", false),
		libraryActivity(4, entity.STATUS_CHANGED_AUTO, `DROPPED`, "2025-04-01", false),
		libraryActivity(3, entity.STATUS_CHANGED, `"DROPPED"`, "2025-03-01", false),
		libraryActivity(2, entity.STATUS_CHANGED, `FINISHED`, "2025-02-01", false),
		libraryActivity(6, entity.SEASON_STATUS_CHANGED, `{"status":"HOLD"}`, "2025-06-02", false),
		libraryActivity(7, entity.EPISODE_STATUS_CHANGED, `{"status":"HOLD"}`, "2025-06-03", false),
		libraryActivity(8, entity.STATUS_CHANGED, `HOLD`, "2024-12-01", false),
		libraryActivity(9, entity.STATUS_CHANGED, `not a status`, "2025-08-01", false),
	)
	counts := statusCounts(buildLibrary([]*watchedRecord{r}, nil, Query{Scope: ScopeYear, Year: 2025}))
	want := map[entity.WatchedStatus]int{entity.FINISHED: 1, entity.WATCHING: 1, entity.PLANNED: 1, entity.HOLD: 0, entity.DROPPED: 1}
	if !reflect.DeepEqual(counts, want) {
		t.Fatalf("status entry counts = %v, want %v", counts, want)
	}
	lifetime := statusCounts(buildLibrary([]*watchedRecord{r}, nil, Query{Scope: ScopeLifetime}))
	if lifetime[entity.WATCHING] != 1 || lifetime[entity.DROPPED] != 0 || lifetime[entity.PLANNED] != 0 {
		t.Fatalf("lifetime must use current saved status: %v", lifetime)
	}
	activities := sortedActivities(r.watched.Activity)
	if activities[0].ID != 8 || activities[1].ID != 1 {
		t.Fatalf("custom dates must determine event ordering: %#v", activities)
	}
}

func TestLibraryMomentumCrossYearAndWaitingBuckets(t *testing.T) {
	start := date("2024-12-31")
	records := []*watchedRecord{}
	for i, days := range []int{0, 1, 7, 8, 30, 31, 90, 91, 365, 366} {
		planned := start.AddDate(0, 0, -days)
		watch := start
		if days > 0 {
			planned = start
			watch = start.AddDate(0, 0, days)
		}
		r := libraryRecord(uint(i+1), entity.FINISHED,
			libraryActivity(3, entity.STATUS_CHANGED, `PLANNED`, watch.AddDate(0, 1, 0).Format("2006-01-02"), false), // replanning
			libraryActivity(2, entity.STATUS_CHANGED, `FINISHED`, watch.Format("2006-01-02"), true),
			libraryActivity(1, entity.ADDED_WATCHED, `{"status":"PLANNED"}`, planned.Format("2006-01-02"), false),
			libraryActivity(4, entity.STATUS_CHANGED, `FINISHED`, watch.AddDate(0, 2, 0).Format("2006-01-02"), true), // rewatch
		)
		records = append(records, r)
	}
	result := buildLibrary(records, nil, Query{Scope: ScopeLifetime})
	if result.Planned != 10 || result.Watched != 10 {
		t.Fatalf("first entries and watches only: %#v", result)
	}
	for i, want := range []int{1, 2, 2, 2, 2, 1} {
		if len(result.Waiting.Buckets[i].Items) != want {
			t.Fatalf("bucket %d: %#v", i, result.Waiting.Buckets)
		}
	}
	if *result.Waiting.MedianDays != 30.5 || result.Waiting.Longest[0].Days != 366 {
		t.Fatalf("median and longest waits: %#v", result.Waiting)
	}
	yearly := buildLibrary(records, nil, Query{Scope: ScopeYear, Year: 2025})
	if yearly.Planned != 0 || yearly.Watched != 8 || len(yearly.Momentum) != 12 || *yearly.Waiting.MedianDays != 30.5 {
		t.Fatalf("2025 must include conversions from prior plans only: %#v", yearly)
	}
	if len(yearly.Momentum[0].Watched) != 5 {
		t.Fatalf("January conversions: %#v", yearly.Momentum[0])
	}
}

func TestLibraryWaitingExcludesMissingAndReversedHistory(t *testing.T) {
	records := []*watchedRecord{
		libraryRecord(1, entity.FINISHED, libraryActivity(1, entity.ADDED_WATCHED, `FINISHED`, "2025-01-01", true)),
		libraryRecord(2, entity.PLANNED, libraryActivity(1, entity.ADDED_WATCHED, `FINISHED`, "2025-01-01", true), libraryActivity(2, entity.STATUS_CHANGED, `PLANNED`, "2025-02-01", false)),
		libraryRecord(3, entity.FINISHED, libraryActivity(2, entity.STATUS_CHANGED, `PLANNED`, "2025-01-01", false), libraryActivity(1, entity.STATUS_CHANGED, `FINISHED`, "2025-01-01", true)),
		libraryRecord(4, entity.FINISHED, libraryActivity(1, entity.ADDED_WATCHED, `PLANNED`, "2025-01-01", false), libraryActivity(2, entity.STATUS_CHANGED, `FINISHED`, "2025-01-01", true)),
		libraryRecord(5, entity.PLANNED),
	}
	result := buildLibrary(records, nil, Query{Scope: ScopeYear, Year: 2025})
	if result.Waiting.Excluded != 3 || result.Planned != 1 || result.Watched != 1 || *result.Waiting.MedianDays != 0 {
		t.Fatalf("missing/reversed history and same-date activity ID order: %#v", result)
	}
	empty := buildLibrary(nil, nil, Query{Scope: ScopeYear, Year: 2024})
	if empty.Waiting.MedianDays != nil || len(empty.Momentum) != 12 || empty.Waiting.Excluded != 0 {
		t.Fatalf("empty library: %#v", empty)
	}
}

func TestLibraryTVFirstEpisodeQualifiesAcrossYears(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "library-tv", Password: "password"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	show := entity.Content{TmdbID: 717, Type: entity.SHOW, Title: "Planned series"}
	if err := db.Create(&show).Error; err != nil {
		t.Fatal(err)
	}
	watched := entity.Watched{UserID: owner.ID, ContentID: &show.ID, Status: entity.WATCHING}
	if err := db.Create(&watched).Error; err != nil {
		t.Fatal(err)
	}
	for _, a := range []entity.Activity{
		libraryActivity(0, entity.ADDED_WATCHED, `{"status":"PLANNED"}`, "2024-12-31", false),
		libraryActivity(0, entity.EPISODE_ADDED_PLEX, `{"season":1,"episode":1,"status":"FINISHED"}`, "2025-01-02", false),
		libraryActivity(0, entity.EPISODE_STATUS_CHANGED, `{"season":1,"episode":1,"status":"FINISHED"}`, "2025-01-03", false),
		libraryActivity(0, entity.STATUS_CHANGED, `FINISHED`, "2026-01-01", true),
	} {
		a.UserID = owner.ID
		a.WatchedID = watched.ID
		if err := db.Create(&a).Error; err != nil {
			t.Fatal(err)
		}
	}
	ep := entity.WatchedEpisode{UserID: owner.ID, WatchedID: watched.ID, SeasonNumber: 1, EpisodeNumber: 1, Status: entity.FINISHED}
	ep.CreatedAt = date("2024-01-01")
	if err := db.Create(&ep).Error; err != nil {
		t.Fatal(err)
	}
	result, err := NewService(db, nil).GetStats(owner.ID, Query{Scope: ScopeYear, Year: 2025, Media: "tv"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Plays != 0 || result.Library.Planned != 0 || result.Library.Watched != 1 || *result.Library.Waiting.MedianDays != 2 || result.Activity.Total != 2 {
		t.Fatalf("first imported episode must qualify: %#v", result)
	}
	if len(result.Calendar) != 2 || result.Calendar[0].Items[0].EpisodeNumber != 1 {
		t.Fatalf("episode calendar: %#v", result.Calendar)
	}
	// The later whole-show play must not become a new first watch.
	later, err := NewService(db, nil).GetStats(owner.ID, Query{Scope: ScopeYear, Year: 2026, Media: "tv"})
	if err != nil || later.Library.Watched != 0 {
		t.Fatalf("prior episodes must prevent duplicate conversions: %#v, %v", later.Library, err)
	}
	// Equal custom dates still preserve the activity order: watching before
	// planning is an existing watch, rather than a same-day conversion.
	secondShow := entity.Content{TmdbID: 718, Type: entity.SHOW, Title: "Previously watched"}
	if err := db.Create(&secondShow).Error; err != nil {
		t.Fatal(err)
	}
	secondWatched := entity.Watched{UserID: owner.ID, ContentID: &secondShow.ID, Status: entity.PLANNED}
	if err := db.Create(&secondWatched).Error; err != nil {
		t.Fatal(err)
	}
	secondEpisode := entity.WatchedEpisode{UserID: owner.ID, WatchedID: secondWatched.ID, SeasonNumber: 1, EpisodeNumber: 1, Status: entity.FINISHED}
	if err := db.Create(&secondEpisode).Error; err != nil {
		t.Fatal(err)
	}
	for _, a := range []entity.Activity{
		libraryActivity(0, entity.EPISODE_ADDED, `{"season":1,"episode":1,"status":"FINISHED"}`, "2025-01-10", false),
		libraryActivity(0, entity.STATUS_CHANGED, `PLANNED`, "2025-01-10", false),
	} {
		a.UserID = owner.ID
		a.WatchedID = secondWatched.ID
		if err := db.Create(&a).Error; err != nil {
			t.Fatal(err)
		}
	}
	sameDay, err := NewService(db, nil).GetStats(owner.ID, Query{Scope: ScopeYear, Year: 2025, Media: "tv"})
	if err != nil {
		t.Fatal(err)
	}
	if sameDay.Library.Watched != 1 || sameDay.Library.Waiting.Excluded != 1 {
		t.Fatalf("same-date episode completions must respect activity ID order: %#v", sameDay.Library)
	}
	if !containsYear(result.AvailableYears, 2024) {
		t.Fatalf("planning-only year missing: %v", result.AvailableYears)
	}
}

func TestCalendarRepeatsAndYearBoundariesMatchActivity(t *testing.T) {
	r := libraryRecord(1, entity.FINISHED)
	r.plays = []time.Time{date("2023-12-31"), date("2024-02-28"), date("2024-02-29"), date("2024-02-29").Add(time.Hour), date("2024-03-01")}
	other := libraryRecord(2, entity.FINISHED)
	other.plays = []time.Time{date("2024-02-29")}
	records := []*watchedRecord{r, other}
	days := buildCalendar(records)
	total := 0
	for _, day := range days {
		total += day.Plays
	}
	if total != buildActivity(records).Total || len(days) != 4 || days[2].Plays != 3 || len(days[2].Items) != 2 || days[2].Items[0].Plays != 2 {
		t.Fatalf("calendar must preserve repeats without duplicate cards: %#v", days)
	}
	scoped := buildCalendar(filterRecordsForYear(records, 2024))
	if len(scoped) != 3 || scoped[0].Date != "2024-02-28" || scoped[2].Date != "2024-03-01" {
		t.Fatalf("year/calendar boundary: %#v", scoped)
	}
	if len(buildCalendar(nil)) != 0 {
		t.Fatal("empty calendar should be empty")
	}
}

func TestLibraryPlanningOnlyYearUsesLocalDataAndOwnerScope(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "library-owner", Password: "password"}
	other := entity.User{Username: "library-viewer", Password: "password"}
	for _, user := range []*entity.User{&owner, &other} {
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i, user := range []*entity.User{&owner, &other} {
		content := entity.Content{TmdbID: 800 + i, Type: entity.MOVIE, Title: []string{"Owner's dropped title", "Viewer's private title"}[i]}
		if err := db.Create(&content).Error; err != nil {
			t.Fatal(err)
		}
		w := entity.Watched{UserID: user.ID, ContentID: &content.ID, Status: entity.DROPPED, Thoughts: "never expose thoughts"}
		if err := db.Create(&w).Error; err != nil {
			t.Fatal(err)
		}
		for _, a := range []entity.Activity{
			libraryActivity(0, entity.IMPORTED_WATCHED, `{"status":"PLANNED"}`, "2020-01-01", false),
			libraryActivity(0, entity.STATUS_CHANGED, `DROPPED`, "2021-01-01", false),
		} {
			a.UserID = user.ID
			a.WatchedID = w.ID
			if err := db.Create(&a).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	response, err := NewService(db, failingTMDB{}).GetStats(owner.ID, Query{Scope: ScopeYear, Year: 2021, Media: "movie"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Summary.Titles != 0 || statusCounts(*response.Library)[entity.DROPPED] != 1 || response.Metadata.Partial || !containsYear(response.AvailableYears, 2020) || !containsYear(response.AvailableYears, 2021) {
		t.Fatalf("local status-only data should survive without enrichment: %#v", response)
	}
	bytes, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(bytes, &payload); err != nil {
		t.Fatal(err)
	}
	item := response.Library.Statuses[4].Items[0]
	if item.Title != "Owner's dropped title" || len(response.Library.Statuses[4].Items) != 1 {
		t.Fatalf("owner isolation: %#v", response.Library)
	}
	if strings.Contains(string(bytes), "never expose thoughts") || strings.Contains(string(bytes), "Viewer's private title") {
		t.Fatal("library payload leaked private thoughts or another owner's title")
	}
}
