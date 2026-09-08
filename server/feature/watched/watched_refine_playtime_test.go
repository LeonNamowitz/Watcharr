package watched

import (
	"testing"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/internal/testutil"
	"github.com/sbondCo/Watcharr/util"
)

type playtimeSortUserProvider struct{}

func (playtimeSortUserProvider) UserGetSettings(uint) (entity.UserSettings, error) {
	return entity.UserSettings{}, nil
}

func TestGetWatchedPageSortsOnlyRecordedGamePlaytime(t *testing.T) {
	db := testutil.SetupDB(t)
	user := entity.User{Username: "playtime-sort-owner", Password: "password"}
	mustCreate(t, db.Create(&user).Error)

	movieA := entity.Content{TmdbID: 401, Title: "Movie A", Type: entity.MOVIE}
	show := entity.Content{TmdbID: 402, Title: "Show", Type: entity.SHOW}
	movieB := entity.Content{TmdbID: 403, Title: "Movie B", Type: entity.MOVIE}
	lowGame := entity.Game{IgdbID: 401, Name: "Low Game"}
	unsetGame := entity.Game{IgdbID: 402, Name: "Unset Game"}
	highGame := entity.Game{IgdbID: 403, Name: "High Game"}
	for _, value := range []any{
		&movieA,
		&show,
		&movieB,
		&lowGame,
		&unsetGame,
		&highGame,
	} {
		mustCreate(t, db.Create(value).Error)
	}

	lowHours := uint(12)
	highHours := uint(90)
	entries := []entity.Watched{
		{
			UserID:    user.ID,
			ContentID: &movieA.ID,
			Status:    entity.FINISHED,
			Pinned:    true,
		},
		{UserID: user.ID, ContentID: &movieB.ID, Status: entity.FINISHED},
		{
			UserID:        user.ID,
			GameID:        &lowGame.ID,
			Status:        entity.FINISHED,
			PlaytimeHours: &lowHours,
		},
		{UserID: user.ID, ContentID: &show.ID, Status: entity.WATCHING},
		{UserID: user.ID, GameID: &unsetGame.ID, Status: entity.PLANNED},
		{
			UserID:        user.ID,
			GameID:        &highGame.ID,
			Status:        entity.FINISHED,
			PlaytimeHours: &highHours,
		},
	}
	for i := range entries {
		mustCreate(t, db.Create(&entries[i]).Error)
	}

	service := NewService(db, nil, nil, nil, playtimeSortUserProvider{})
	assertPlaytimeSort(t, service, user.ID, domain.WatchedSortDirDesc, []string{
		"Movie A",
		"High Game",
		"Low Game",
		"Movie B",
		"Show",
		"Unset Game",
	})
	assertPlaytimeSort(t, service, user.ID, domain.WatchedSortDirAsc, []string{
		"Movie A",
		"Low Game",
		"High Game",
		"Movie B",
		"Show",
		"Unset Game",
	})

	dto := domain.NewWatchedDtoForLists(&entries[2])
	if dto.PlaytimeHours == nil || *dto.PlaytimeHours != lowHours {
		t.Fatalf("list dto playtime = %v, want %d", dto.PlaytimeHours, lowHours)
	}

	publicDTO := domain.NewWatchedDtoForPublicLists(&entries[2])
	if publicDTO.PlaytimeHours == nil || *publicDTO.PlaytimeHours != lowHours {
		t.Fatalf(
			"public list dto playtime = %v, want %d",
			publicDTO.PlaytimeHours,
			lowHours,
		)
	}
}

func assertPlaytimeSort(
	t *testing.T,
	service *Service,
	userID uint,
	direction domain.SortDirection,
	want []string,
) {
	t.Helper()
	page, err := service.GetWatchedPage(
		userID,
		util.PaginationParams{Page: 1, Limit: 20},
		domain.WatchedGetPageRequest{
			Sort:    domain.WatchedSortPlaytime,
			SortDir: direction,
		},
		nil,
	)
	if err != nil {
		t.Fatalf("playtime page failed: %v", err)
	}

	got := make([]string, 0, len(page.Results))
	for i := range page.Results {
		switch {
		case page.Results[i].Content != nil:
			got = append(got, page.Results[i].Content.Title)
		case page.Results[i].Game != nil:
			got = append(got, page.Results[i].Game.Name)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("sorted names = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sorted names = %v, want %v", got, want)
		}
	}
}
