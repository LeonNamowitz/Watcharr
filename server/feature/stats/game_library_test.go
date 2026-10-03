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

func TestGameLibraryStatusAndFirstBacklogCompletion(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "game-library", Password: "password"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	hours := uint(20)
	crossYear := createStatsGame(t, db, owner.ID, 501, entity.FINISHED, 8, &hours)
	// Insert out of chronological order; custom dates must determine conversion.
	addGameStatsActivity(t, db, crossYear, entity.STATUS_CHANGED, `FINISHED`, "2024-03-01", true)
	addGameStatsActivity(t, db, crossYear, entity.IMPORTED_ADDED_WATCHED, `{"status":"PLANNED"}`, "2023-12-31", false)
	addGameStatsActivity(t, db, crossYear, entity.STATUS_CHANGED, `WATCHING`, "2024-02-28", false)
	addGameStatsActivity(t, db, crossYear, entity.STATUS_CHANGED, `HOLD`, "2024-02-29", false)
	addGameStatsActivity(t, db, crossYear, entity.STATUS_CHANGED, `DROPPED`, "2024-02-29", false)
	addGameStatsActivity(t, db, crossYear, entity.STATUS_CHANGED_AUTO, `{"status":"WATCHING"}`, "2024-03-01", false)
	addGameStatsActivity(t, db, crossYear, entity.STATUS_CHANGED, `FINISHED`, "2024-03-01", true)
	addGameStatsActivity(t, db, crossYear, entity.STATUS_CHANGED, `PLANNED`, "2025-01-01", false)
	addGameStatsActivity(t, db, crossYear, entity.STATUS_CHANGED, `FINISHED`, "2025-01-02", true)
	planningOnly := createStatsGame(t, db, owner.ID, 502, entity.PLANNED, 0, nil)
	addGameStatsActivity(t, db, planningOnly, entity.IMPORTED_WATCHED, `{"status":"PLANNED"}`, "2019-01-01", false)
	leap := createStatsGame(t, db, owner.ID, 503, entity.FINISHED, 9, nil)
	addGameStatsActivity(t, db, leap, entity.ADDED_WATCHED, `PLANNED`, "2024-02-28", false)
	addGameStatsActivity(t, db, leap, entity.STATUS_CHANGED, `FINISHED`, "2024-02-29", true)
	missing := createStatsGame(t, db, owner.ID, 504, entity.FINISHED, 0, nil)
	addGameStatsActivity(t, db, missing, entity.IMPORTED_WATCHED, `{"status":"FINISHED"}`, "2024-01-01", true)
	alreadyCompleted := createStatsGame(t, db, owner.ID, 505, entity.PLANNED, 0, nil)
	addGameStatsActivity(t, db, alreadyCompleted, entity.STATUS_CHANGED, `FINISHED`, "2024-01-01", true)
	addGameStatsActivity(t, db, alreadyCompleted, entity.STATUS_CHANGED, `PLANNED`, "2024-01-02", false)
	addGameStatsActivity(t, db, alreadyCompleted, entity.STATUS_CHANGED, `FINISHED`, "2024-01-03", true)
	reversed := createStatsGame(t, db, owner.ID, 506, entity.PLANNED, 0, nil)
	addGameStatsActivity(t, db, reversed, entity.STATUS_CHANGED, `FINISHED`, "2024-04-01", true)
	addGameStatsActivity(t, db, reversed, entity.STATUS_CHANGED, `PLANNED`, "2024-04-01", false)
	sameDay := createStatsGame(t, db, owner.ID, 507, entity.FINISHED, 0, nil)
	addGameStatsActivity(t, db, sameDay, entity.ADDED_WATCHED, `PLANNED`, "2024-04-01", false)
	addGameStatsActivity(t, db, sameDay, entity.STATUS_CHANGED, `FINISHED`, "2024-04-01", true)
	createStatsGame(t, db, owner.ID, 508, entity.HOLD, 0, nil) // legacy progress only
	service := NewService(db, nil)
	year := getGameStatsForTest(t, service, owner.ID, Query{Scope: ScopeYear, Year: 2024})
	want := map[entity.WatchedStatus]int{entity.FINISHED: 6, entity.WATCHING: 1, entity.PLANNED: 4, entity.HOLD: 1, entity.DROPPED: 1}
	if !reflect.DeepEqual(statusCounts(*year.Library), want) || year.Library.Statuses[1].Label != "Playing" {
		t.Fatalf("distinct recorded statuses: %#v", year.Library.Statuses)
	}
	library := year.Library
	if library.Planned != 2 || library.Watched != 3 || library.Waiting.Excluded != 3 || *library.Waiting.MedianDays != 1 || library.Waiting.Longest[0].Days != 61 || len(library.Momentum) != 12 {
		t.Fatalf("first backlog completions: %#v", library)
	}
	for i, want := range []int{1, 1, 0, 1, 0, 0} {
		if len(library.Waiting.Buckets[i].Items) != want {
			t.Fatalf("waiting bucket %d: %#v", i, library.Waiting.Buckets)
		}
	}
	if len(library.Momentum[2].Watched) != 1 || library.Momentum[2].Watched[0].ID != 501 {
		t.Fatal("completion should convert the prior year's planning entry")
	}
	for _, q := range []Query{{Scope: ScopeLifetime}, {Scope: ScopeYear, Year: 2024}, {Scope: ScopeYear, Year: 2025}, {Scope: ScopeYear, Year: 2019}, {Scope: ScopeYear, Year: 2018}} {
		data := getGameStatsForTest(t, service, owner.ID, q)
		progress, completions := 0, 0
		for _, day := range data.Calendar {
			progress += day.Plays
			for _, item := range day.Items {
				if item.Type != "game" || item.Plays != 1 || item.Date != day.Date {
					t.Fatalf("progress cards must be deduplicated per game/day: %#v", day)
				}
			}
		}
		for _, day := range data.Games.CalendarCompletions {
			completions += day.Plays
		}
		if progress != data.Activity.Total || completions != data.Games.Completions.Total {
			t.Fatalf("calendar/activity totals disagree for %#v: %d/%d, %d/%d", q, progress, data.Activity.Total, completions, data.Games.Completions.Total)
		}
		if q.Scope == ScopeYear {
			body, _ := json.Marshal(data)
			if strings.Contains(string(body), "playtimeHours") {
				t.Fatal("new year cards must preserve the playtime field policy")
			}
		}
	}
	for _, day := range year.Games.CalendarCompletions {
		if day.Date == "2024-03-01" && (day.Plays != 2 || len(day.Items) != 1 || day.Items[0].Plays != 2) {
			t.Fatalf("repeated completions need one card and both finishes: %#v", day)
		}
	}
	life := getGameStatsForTest(t, service, owner.ID, Query{Scope: ScopeLifetime})
	if statusCounts(*life.Library)[entity.PLANNED] != 3 || statusCounts(*life.Library)[entity.HOLD] != 1 || life.Library.Planned != 4 || life.Library.Watched != 3 || len(life.Library.Momentum) != 3 {
		t.Fatalf("current statuses and annual first entries: %#v", life.Library)
	}
	planYear := getGameStatsForTest(t, service, owner.ID, Query{Scope: ScopeYear, Year: 2019})
	if planYear.Summary.Titles != 0 || planYear.Library.Planned != 1 || statusCounts(*planYear.Library)[entity.PLANNED] != 1 || !containsYear(planYear.AvailableYears, 2019) {
		t.Fatalf("planning-only years must be selectable: %#v", planYear)
	}
	replayYear := getGameStatsForTest(t, service, owner.ID, Query{Scope: ScopeYear, Year: 2025})
	if replayYear.Library.Planned != 0 || replayYear.Library.Watched != 0 || replayYear.Library.Waiting.Excluded != 0 {
		t.Fatalf("replanning/replays must not become first conversions: %#v", replayYear.Library)
	}
}

func TestGameLibraryCalendarUTCAndCreatedDate(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "game-utc", Password: "password"}
	db.Create(&owner)
	w := createStatsGame(t, db, owner.ID, 601, entity.FINISHED, 0, nil)
	plan := entity.Activity{UserID: owner.ID, WatchedID: w.ID, Type: entity.ADDED_WATCHED, Data: `PLANNED`}
	plan.CreatedAt = date("2024-02-28").Add(23 * time.Hour)
	if err := db.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	// The local Feb 28 date falls on leap day in UTC.
	finishedAt, err := time.Parse(time.RFC3339, "2024-02-28T21:30:00-03:00")
	if err != nil {
		t.Fatal(err)
	}
	finish := entity.Activity{UserID: owner.ID, WatchedID: w.ID, Type: entity.STATUS_CHANGED, Data: `FINISHED`, CustomDate: &finishedAt, CountAsPlay: true}
	if err := db.Create(&finish).Error; err != nil {
		t.Fatal(err)
	}
	data := getGameStatsForTest(t, NewService(db, nil), owner.ID, Query{Scope: ScopeYear, Year: 2024})
	if data.Library.Watched != 1 || *data.Library.Waiting.MedianDays != 1 || len(data.Calendar) != 1 || data.Calendar[0].Date != "2024-02-29" || data.Games.CalendarCompletions[0].Date != "2024-02-29" {
		t.Fatalf("effective dates and waiting days must use UTC: %#v", data.Library)
	}
}
