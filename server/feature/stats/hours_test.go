package stats

import (
	"testing"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
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
