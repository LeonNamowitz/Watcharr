package watched

import (
	"testing"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/internal/testutil"
	"github.com/sbondCo/Watcharr/util"
)

type watchedSearchUserProvider struct{}

func (watchedSearchUserProvider) UserGetSettings(uint) (entity.UserSettings, error) {
	return entity.UserSettings{}, nil
}

func TestFuzzyTitleMatch(t *testing.T) {
	tests := []struct {
		name  string
		title string
		query string
		want  bool
	}{
		{
			name:  "case insensitive typo and punctuation",
			title: "Spider-Man: No Way Home",
			query: "SPIDE MAN",
			want:  true,
		},
		{
			name:  "joined words ignore punctuation",
			title: "Spider-Man: No Way Home",
			query: "spiderman",
			want:  true,
		},
		{
			name:  "tokens can be reordered",
			title: "Spider-Man: No Way Home",
			query: "man spider",
			want:  true,
		},
		{
			name:  "partial title",
			title: "The Witcher 3: Wild Hunt",
			query: "witch",
			want:  true,
		},
		{
			name:  "single token typo",
			title: "The Witcher 3: Wild Hunt",
			query: "witchr",
			want:  true,
		},
		{
			name:  "different first character is not a typo",
			title: "Jurassic Park",
			query: "dark",
			want:  false,
		},
		{
			name:  "unrelated title",
			title: "Superman",
			query: "spide man",
			want:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := fuzzyTitleMatch(test.title, test.query); got != test.want {
				t.Fatalf("fuzzyTitleMatch(%q, %q) = %v, want %v",
					test.title, test.query, got, test.want)
			}
		})
	}
}

func TestGetWatchedPageFuzzySearchIsOwnerScopedSortedAndPaginated(t *testing.T) {
	db := testutil.SetupDB(t)
	public := false
	owner := entity.User{
		Username:     "owner",
		Password:     "password",
		UserSettings: entity.UserSettings{Private: &public},
	}
	other := entity.User{
		Username:     "other",
		Password:     "password",
		UserSettings: entity.UserSettings{Private: &public},
	}
	mustCreate(t, db.Create(&owner).Error)
	mustCreate(t, db.Create(&other).Error)

	spiderMan := entity.Content{TmdbID: 11, Title: "Spider-Man", Type: entity.MOVIE}
	noWayHome := entity.Content{TmdbID: 12, Title: "Spider-Man: No Way Home", Type: entity.SHOW}
	superman := entity.Content{TmdbID: 13, Title: "Superman", Type: entity.MOVIE}
	otherSpiderMan := entity.Content{TmdbID: 14, Title: "Spider-Man 2099", Type: entity.MOVIE}
	spiderManGame := entity.Game{IgdbID: 21, Name: "Spider-Man 2"}
	for _, value := range []any{&spiderMan, &noWayHome, &superman, &otherSpiderMan, &spiderManGame} {
		mustCreate(t, db.Create(value).Error)
	}

	watched := []entity.Watched{
		{
			UserID:    owner.ID,
			ContentID: &spiderMan.ID,
			Status:    entity.FINISHED,
			Rating:    8,
		},
		{
			UserID:    owner.ID,
			ContentID: &noWayHome.ID,
			Status:    entity.PLANNED,
			Rating:    7,
		},
		{
			UserID: owner.ID,
			GameID: &spiderManGame.ID,
			Status: entity.DROPPED,
			Rating: 6,
		},
		{
			UserID:    owner.ID,
			ContentID: &superman.ID,
			Status:    entity.WATCHING,
		},
		{
			UserID:    other.ID,
			ContentID: &otherSpiderMan.ID,
			Status:    entity.FINISHED,
			Rating:    10,
		},
	}
	for i := range watched {
		mustCreate(t, db.Create(&watched[i]).Error)
	}

	service := NewService(db, nil, nil, nil, watchedSearchUserProvider{})
	request := domain.WatchedGetPageRequest{
		Sort:    domain.WatchedSortAlphabetical,
		SortDir: domain.WatchedSortDirAsc,
	}
	extra := &domain.WatchedGetPageExtraProps{Query: "SPIDE MAN"}

	firstPage, err := service.GetWatchedPage(
		owner.ID,
		util.PaginationParams{Page: 1, Limit: 2},
		request,
		extra,
	)
	if err != nil {
		t.Fatalf("first page search failed: %v", err)
	}
	if firstPage.TotalResults != 3 || firstPage.TotalPages != 2 {
		t.Fatalf("first page totals = (%d results, %d pages), want (3, 2)",
			firstPage.TotalResults, firstPage.TotalPages)
	}
	assertWatchedNames(t, firstPage.Results, []string{"Spider-Man", "Spider-Man 2"})
	assertStatuses(t, firstPage.Results, []entity.WatchedStatus{
		entity.FINISHED,
		entity.DROPPED,
	})

	secondPage, err := service.GetWatchedPage(
		owner.ID,
		util.PaginationParams{Page: 2, Limit: 2},
		request,
		extra,
	)
	if err != nil {
		t.Fatalf("second page search failed: %v", err)
	}
	if secondPage.TotalResults != 3 || secondPage.TotalPages != 2 {
		t.Fatalf("second page totals = (%d results, %d pages), want (3, 2)",
			secondPage.TotalResults, secondPage.TotalPages)
	}
	assertWatchedNames(t, secondPage.Results, []string{"Spider-Man: No Way Home"})
	assertStatuses(t, secondPage.Results, []entity.WatchedStatus{entity.PLANNED})

	unfiltered, err := service.GetWatchedPage(
		owner.ID,
		util.PaginationParams{Page: 1, Limit: 10},
		request,
		&domain.WatchedGetPageExtraProps{Query: ""},
	)
	if err != nil {
		t.Fatalf("empty-query list failed: %v", err)
	}
	if unfiltered.TotalResults != 4 {
		t.Fatalf("empty query returned %d owner items, want 4", unfiltered.TotalResults)
	}
}

func mustCreate(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("creating test record failed: %v", err)
	}
}

func assertWatchedNames(t *testing.T, watched []entity.Watched, want []string) {
	t.Helper()
	if len(watched) != len(want) {
		t.Fatalf("got %d watched items, want %d", len(watched), len(want))
	}
	for i := range watched {
		name := ""
		if watched[i].Content != nil {
			name = watched[i].Content.Title
		} else if watched[i].Game != nil {
			name = watched[i].Game.Name
		}
		if name != want[i] {
			t.Fatalf("item %d name = %q, want %q", i, name, want[i])
		}
	}
}

func assertStatuses(t *testing.T, watched []entity.Watched, want []entity.WatchedStatus) {
	t.Helper()
	if len(watched) != len(want) {
		t.Fatalf("got %d watched items, want %d", len(watched), len(want))
	}
	for i := range watched {
		if watched[i].Status != want[i] {
			t.Fatalf("item %d status = %q, want %q", i, watched[i].Status, want[i])
		}
	}
}
