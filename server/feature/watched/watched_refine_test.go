package watched

import (
	"testing"
	"time"

	"github.com/sbondCo/Watcharr/database/dbmodel"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/internal/testutil"
	"github.com/sbondCo/Watcharr/util"
)

type watchedSortUserProvider struct{}

func (watchedSortUserProvider) UserGetSettings(uint) (entity.UserSettings, error) {
	return entity.UserSettings{}, nil
}

func TestGetWatchedPageDateAddedUsesActivityCustomDate(t *testing.T) {
	db := testutil.SetupDB(t)
	user := entity.User{Username: "owner", Password: "password"}
	mustCreate(t, db.Create(&user).Error)

	contents := []entity.Content{
		{TmdbID: 1, Title: "Pinned", Type: entity.MOVIE},
		{TmdbID: 2, Title: "Edited activity", Type: entity.MOVIE},
		{TmdbID: 3, Title: "Fallback", Type: entity.MOVIE},
		{TmdbID: 4, Title: "Recent activity", Type: entity.MOVIE},
	}
	for i := range contents {
		mustCreate(t, db.Create(&contents[i]).Error)
	}

	date := func(value string) time.Time {
		t.Helper()
		parsed, err := time.Parse(time.DateOnly, value)
		if err != nil {
			t.Fatalf("parsing test date %q failed: %v", value, err)
		}
		return parsed
	}

	createdDates := []time.Time{
		date("2010-01-01"),
		date("2026-01-01"),
		date("2020-01-01"),
		date("2019-01-01"),
	}
	watched := make([]entity.Watched, len(contents))
	for i := range watched {
		watched[i] = entity.Watched{
			GormModel: dbmodel.GormModel{
				CreatedAt: createdDates[i],
				UpdatedAt: createdDates[i],
			},
			UserID:    user.ID,
			ContentID: &contents[i].ID,
			Status:    entity.FINISHED,
			Pinned:    i == 0,
		}
		mustCreate(t, db.Create(&watched[i]).Error)
	}

	activityDates := map[int]time.Time{
		0: date("2000-01-01"),
		1: date("2017-12-01"),
		3: date("2024-01-01"),
	}
	for watchedIndex, customDate := range activityDates {
		mustCreate(t, db.Create(&entity.Activity{
			UserID:     user.ID,
			WatchedID:  watched[watchedIndex].ID,
			Type:       entity.ADDED_WATCHED,
			CustomDate: &customDate,
		}).Error)
	}
	// Other activity types must not redefine when the item was added.
	otherActivityDate := date("2030-01-01")
	mustCreate(t, db.Create(&entity.Activity{
		UserID:     user.ID,
		WatchedID:  watched[2].ID,
		Type:       entity.STATUS_CHANGED,
		CustomDate: &otherActivityDate,
	}).Error)

	service := NewService(db, nil, nil, nil, watchedSortUserProvider{})
	page, err := service.GetWatchedPage(
		user.ID,
		util.PaginationParams{Page: 1, Limit: 10},
		domain.WatchedGetPageRequest{
			Sort:    domain.WatchedSortDateAdded,
			SortDir: domain.WatchedSortDirDesc,
		},
		nil,
	)
	if err != nil {
		t.Fatalf("date-added page failed: %v", err)
	}

	assertWatchedNames(t, page.Results, []string{
		"Pinned",
		"Recent activity",
		"Fallback",
		"Edited activity",
	})
}
