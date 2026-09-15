package domain

import (
	"testing"
	"time"

	"github.com/sbondCo/Watcharr/database/dbmodel"
	"github.com/sbondCo/Watcharr/database/entity"
)

func TestGetLastSeenFromActivityIncludesShowSeasonAndEpisodeFinishes(t *testing.T) {
	showFinished := time.Date(2025, time.January, 2, 12, 0, 0, 0, time.UTC)
	seasonFinished := time.Date(2025, time.February, 3, 12, 0, 0, 0, time.UTC)
	episodeCreated := time.Date(2025, time.March, 4, 12, 0, 0, 0, time.UTC)
	episodeCustom := time.Date(2025, time.April, 5, 12, 0, 0, 0, time.UTC)
	activities := []entity.Activity{
		{
			GormModel:   dbmodel.GormModel{CreatedAt: showFinished},
			Type:        entity.STATUS_CHANGED,
			Data:        "FINISHED",
			CountAsPlay: true,
		},
		{
			GormModel: dbmodel.GormModel{CreatedAt: seasonFinished},
			Type:      entity.SEASON_STATUS_CHANGED,
			Data:      `{"season":2,"status":"FINISHED"}`,
		},
		{
			GormModel:  dbmodel.GormModel{CreatedAt: episodeCreated},
			Type:       entity.EPISODE_ADDED,
			Data:       `{"season":2,"episode":3,"status":"FINISHED"}`,
			CustomDate: &episodeCustom,
		},
	}

	lastSeen := getLastSeenFromActivity(activities)
	if lastSeen == nil || !lastSeen.Equal(episodeCustom) {
		t.Fatalf("last seen = %v, want %v", lastSeen, episodeCustom)
	}
}

func TestGetLastSeenFromActivityIgnoresNonFinishedStatusActivity(t *testing.T) {
	created := time.Date(2025, time.May, 6, 12, 0, 0, 0, time.UTC)
	activities := []entity.Activity{
		{
			GormModel: dbmodel.GormModel{CreatedAt: created},
			Type:      entity.EPISODE_STATUS_CHANGED,
			Data:      `{"season":1,"episode":1,"status":"WATCHING"}`,
		},
		{
			GormModel: dbmodel.GormModel{CreatedAt: created.Add(time.Hour)},
			Type:      entity.RATING_CHANGED,
			Data:      `{"status":"FINISHED"}`,
		},
	}

	if lastSeen := getLastSeenFromActivity(activities); lastSeen != nil {
		t.Fatalf("last seen = %v, want nil", lastSeen)
	}
}
