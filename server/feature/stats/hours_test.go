package stats

import (
	"testing"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/internal/testutil"
)

func TestHoursFinishedShowAndEpisodeCoverage(t *testing.T) {
	d := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	show := &watchedRecord{content: &entity.Content{Type: entity.SHOW, Runtime: 30, NumberOfEpisodes: 10}, plays: []time.Time{d}}
	show.watched.ID = 1
	episode := &watchedRecord{parent: show, content: &entity.Content{Runtime: 30}, plays: []time.Time{d}}
	if got := *estimatedHours([]*watchedRecord{show}, []*watchedRecord{episode}); got != 5 {
		t.Fatalf("show completion must cover all episodes without counting explicit finishes twice: %v", got)
	}
	if got := *estimatedHours(nil, []*watchedRecord{episode}); got != 0.5 {
		t.Fatalf("episode-only viewing must still contribute: %v", got)
	}
	show.plays = append(show.plays, d)
	if got := *estimatedHours([]*watchedRecord{show}, []*watchedRecord{episode}); got != 10 {
		t.Fatalf("whole-show rewatches must contribute another full runtime: %v", got)
	}
}

func TestHoursFinishedLegacyShowUsesRecordedYear(t *testing.T) {
	show := &watchedRecord{content: &entity.Content{Type: entity.SHOW, Runtime: 30, NumberOfEpisodes: 10}}
	show.watched.Status = entity.FINISHED
	show.watched.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, year := range []int{2024, 2025} {
		got := *estimatedHours(recordsForHours([]*watchedRecord{show}, Query{Scope: ScopeYear, Year: year}), nil)
		want := float64(0)
		if year == 2025 {
			want = 5
		}
		if got != want {
			t.Fatalf("year %d hours: got %v, want %v", year, got, want)
		}
	}
}

func TestHoursDoesNotInferFinishesFromStatusHistory(t *testing.T) {
	for _, status := range []entity.WatchedStatus{entity.PLANNED, entity.FINISHED} {
		t.Run(string(status), func(t *testing.T) {
			movie := libraryRecord(1, entity.FINISHED,
				libraryActivity(1, entity.ADDED_WATCHED, string(status), "2024-01-01", false),
				libraryActivity(2, entity.STATUS_CHANGED, "FINISHED", "2025-02-01", false),
			)
			movie.watched.CreatedAt = date("2024-01-01")
			movie.content.Runtime = 120
			for _, q := range []Query{{Scope: ScopeLifetime}, {Scope: ScopeYear, Year: 2024}, {Scope: ScopeYear, Year: 2025}} {
				if hours := *estimatedHours(recordsForHours([]*watchedRecord{movie}, q), nil); hours != 0 {
					t.Fatalf("uncounted finish must not infer hours from creation date for %#v: %v", q, hours)
				}
			}
		})
	}
}

func TestStatsLegacyFinishYearsAreAvailable(t *testing.T) {
	for _, media := range []entity.ContentType{entity.MOVIE, entity.SHOW} {
		t.Run(string(media), func(t *testing.T) {
			db := testutil.SetupDB(t)
			owner := entity.User{Username: "legacy-hours-owner", Password: "password"}
			if err := db.Create(&owner).Error; err != nil {
				t.Fatal(err)
			}
			content := entity.Content{TmdbID: 11231, Type: media, Title: "Legacy finish", Runtime: 120, NumberOfEpisodes: 1}
			if err := db.Create(&content).Error; err != nil {
				t.Fatal(err)
			}
			watched := entity.Watched{UserID: owner.ID, ContentID: &content.ID, Status: entity.FINISHED}
			watched.CreatedAt = date("2019-01-01")
			if err := db.Create(&watched).Error; err != nil {
				t.Fatal(err)
			}
			service := NewService(db, nil)
			for _, q := range []Query{{Scope: ScopeLifetime, Media: string(media)}, {Scope: ScopeYear, Year: 2019, Media: string(media)}} {
				data, err := service.GetStats(owner.ID, q)
				if err != nil {
					t.Fatal(err)
				}
				if !containsYear(data.AvailableYears, 2019) || data.Summary.Hours == nil || *data.Summary.Hours != 2 {
					t.Fatalf("legacy finish hours must have a selectable year: years=%v hours=%v", data.AvailableYears, data.Summary.Hours)
				}
			}
		})
	}
}
