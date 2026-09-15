package domain

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/sbondCo/Watcharr/database/entity"
)

var statusActivityTypes = map[entity.ActivityType]bool{
	entity.STATUS_CHANGED:             true,
	entity.STATUS_CHANGED_AUTO:        true,
	entity.SEASON_ADDED:               true,
	entity.SEASON_ADDED_AUTO:          true,
	entity.SEASON_ADDED_JF:            true,
	entity.SEASON_ADDED_PLEX:          true,
	entity.SEASON_STATUS_CHANGED:      true,
	entity.SEASON_STATUS_CHANGED_AUTO: true,
	entity.EPISODE_ADDED:              true,
	entity.EPISODE_ADDED_JF:           true,
	entity.EPISODE_ADDED_PLEX:         true,
	entity.EPISODE_STATUS_CHANGED:     true,
}

// LastSeenStatusActivityTypes returns the non-play activity types whose data
// can identify a TV show, season, or episode finish.
func LastSeenStatusActivityTypes() []entity.ActivityType {
	types := make([]entity.ActivityType, 0, len(statusActivityTypes))
	for activityType := range statusActivityTypes {
		types = append(types, activityType)
	}
	return types
}

type (
	// Internal struct accepted by AddActivity function.
	ActivityAddProps struct {
		WatchedID  uint                `json:"watchedId" binding:"required"`
		Type       entity.ActivityType `json:"type" binding:"required"`
		Data       string              `json:"data" binding:"required"`
		CustomDate *time.Time          `json:"customDate,omitempty"`
	}

	ActivityUpdateRequest struct {
		CustomDate time.Time `json:"customDate" binding:"required"`
	}

	ActivityAddProvider interface {
		AddActivity(
			userId uint,
			ar ActivityAddProps,
			countAsPlay bool,
		) (entity.Activity, error)
	}
)

// Looks through Activity for Watched entry and calculates the amount
// that count as plays.
func getPlaysFromActivity(a []entity.Activity) int {
	plays := 0
	for i := range a {
		if a[i].CountAsPlay {
			plays++
		}
	}
	return plays
}

// getLastSeenFromActivity returns the effective date of the latest finished
// activity. CountAsPlay covers ordinary/imported finishes. TV season and
// episode finishes do not count as whole-show plays, so their status-bearing
// activity data is checked as well.
func getLastSeenFromActivity(activities []entity.Activity) *time.Time {
	var latest time.Time
	for i := range activities {
		activity := &activities[i]
		if !activity.CountAsPlay && !activityHasFinishedStatus(*activity) {
			continue
		}
		activityDate := activity.CreatedAt
		if activity.CustomDate != nil {
			activityDate = *activity.CustomDate
		}
		if activityDate.After(latest) {
			latest = activityDate
		}
	}
	if latest.IsZero() {
		return nil
	}
	return &latest
}

func activityHasFinishedStatus(activity entity.Activity) bool {
	if !statusActivityTypes[activity.Type] {
		return false
	}
	if strings.EqualFold(strings.Trim(strings.TrimSpace(activity.Data), `"`), string(entity.FINISHED)) {
		return true
	}
	var data struct {
		Status entity.WatchedStatus `json:"status"`
	}
	return json.Unmarshal([]byte(activity.Data), &data) == nil && data.Status == entity.FINISHED
}
