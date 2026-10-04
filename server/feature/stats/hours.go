package stats

import (
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
)

// Legacy finishes have no recorded status history to supply or suppress a play.
func hasLegacyFinish(r *watchedRecord) bool {
	if len(r.plays) > 0 || r.watched.Status != entity.FINISHED || r.watched.CreatedAt.IsZero() {
		return false
	}
	for _, activity := range r.watched.Activity {
		if topLevelStatus(activity) != "" {
			return false
		}
	}
	return true
}

// Finished legacy rows without status history use their original recorded date.
func recordsForHours(records []*watchedRecord, q Query) []*watchedRecord {
	result := []*watchedRecord{}
	for _, r := range records {
		clone := *r
		clone.plays = gameDatesInScope(r.plays, q)
		if hasLegacyFinish(r) {
			clone.plays = gameDatesInScope([]time.Time{r.watched.CreatedAt.UTC()}, q)
		}
		if len(clone.plays) > 0 {
			result = append(result, &clone)
		}
	}
	return result
}

// Whole-show finishes cover all episodes. Explicit episode finishes only add
// time beyond that coverage, so marking both does not double the estimate.
func estimatedHours(records, episodes []*watchedRecord) *float64 {
	whole, individual := map[uint]float64{}, map[uint]float64{}
	for _, r := range records {
		minutes := float64(r.content.Runtime) * float64(len(r.plays))
		if r.content.Type == entity.SHOW {
			minutes *= float64(r.content.NumberOfEpisodes)
		}
		whole[r.watched.ID] += minutes
	}
	for _, ep := range episodes {
		runtime := ep.content.Runtime
		if runtime == 0 {
			runtime = ep.parent.content.Runtime
		}
		individual[ep.parent.watched.ID] += float64(runtime) * float64(len(ep.plays))
	}
	for id, minutes := range individual {
		if minutes > whole[id] {
			whole[id] = minutes
		}
	}
	hours := float64(0)
	for _, minutes := range whole {
		hours += minutes / 60
	}
	return &hours
}
