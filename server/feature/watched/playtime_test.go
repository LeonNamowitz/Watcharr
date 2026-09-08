package watched

import (
	"encoding/json"
	"testing"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/internal/testutil"
)

type playtimeActivityProvider struct {
	requests []domain.ActivityAddProps
}

func (p *playtimeActivityProvider) AddActivity(
	_ uint,
	ar domain.ActivityAddProps,
	_ bool,
) (entity.Activity, error) {
	p.requests = append(p.requests, ar)
	return entity.Activity{Type: ar.Type, Data: ar.Data}, nil
}

func TestUpdateWatchedGamePlaytime(t *testing.T) {
	db := testutil.SetupDB(t)
	user := entity.User{Username: "playtime-owner", Password: "password"}
	game := entity.Game{IgdbID: 101, Name: "Test Game"}
	mustCreate(t, db.Create(&user).Error)
	mustCreate(t, db.Create(&game).Error)

	watched := entity.Watched{
		UserID: user.ID,
		GameID: &game.ID,
		Status: entity.PLANNED,
	}
	mustCreate(t, db.Create(&watched).Error)
	activityProvider := &playtimeActivityProvider{}
	service := NewService(db, nil, nil, activityProvider, nil)

	hours := uint(42)
	if _, err := service.updateWatched(user.ID, watched.ID, domain.WatchedUpdateRequest{
		PlaytimeHours: &hours,
	}); err != nil {
		t.Fatalf("setting playtime failed: %v", err)
	}

	var updated entity.Watched
	mustCreate(t, db.First(&updated, watched.ID).Error)
	if updated.PlaytimeHours == nil || *updated.PlaytimeHours != hours {
		t.Fatalf("playtime = %v, want %d", updated.PlaytimeHours, hours)
	}
	assertPlaytimeActivity(t, activityProvider, &hours)

	if _, err := service.updateWatched(user.ID, watched.ID, domain.WatchedUpdateRequest{
		RemovePlaytime: true,
	}); err != nil {
		t.Fatalf("clearing playtime failed: %v", err)
	}
	mustCreate(t, db.First(&updated, watched.ID).Error)
	if updated.PlaytimeHours != nil {
		t.Fatalf("playtime = %v after clearing, want nil", updated.PlaytimeHours)
	}
	assertPlaytimeActivity(t, activityProvider, nil)

	zero := uint(0)
	if _, err := service.updateWatched(user.ID, watched.ID, domain.WatchedUpdateRequest{
		PlaytimeHours: &zero,
	}); err != nil {
		t.Fatalf("setting zero playtime failed: %v", err)
	}
	mustCreate(t, db.First(&updated, watched.ID).Error)
	if updated.PlaytimeHours == nil || *updated.PlaytimeHours != 0 {
		t.Fatalf("playtime = %v, want explicit zero", updated.PlaytimeHours)
	}
	assertPlaytimeActivity(t, activityProvider, &zero)
}

func assertPlaytimeActivity(
	t *testing.T,
	provider *playtimeActivityProvider,
	wantHours *uint,
) {
	t.Helper()
	if len(provider.requests) == 0 {
		t.Fatal("playtime update did not add activity")
	}
	activity := provider.requests[len(provider.requests)-1]
	if activity.Type != entity.PLAYTIME_CHANGED {
		t.Fatalf("activity type = %q, want %q", activity.Type, entity.PLAYTIME_CHANGED)
	}
	var data struct {
		Hours *uint `json:"hours"`
	}
	if err := json.Unmarshal([]byte(activity.Data), &data); err != nil {
		t.Fatalf("activity data %q is invalid: %v", activity.Data, err)
	}
	if wantHours == nil {
		if data.Hours != nil {
			t.Fatalf("activity hours = %v, want nil", *data.Hours)
		}
		return
	}
	if data.Hours == nil || *data.Hours != *wantHours {
		t.Fatalf("activity hours = %v, want %d", data.Hours, *wantHours)
	}
}

func TestUpdateWatchedRejectsPlaytimeForNonGameAndOtherUser(t *testing.T) {
	db := testutil.SetupDB(t)
	owner := entity.User{Username: "playtime-owner", Password: "password"}
	other := entity.User{Username: "playtime-other", Password: "password"}
	content := entity.Content{TmdbID: 201, Title: "Test Movie", Type: entity.MOVIE}
	game := entity.Game{IgdbID: 202, Name: "Owned Game"}
	for _, value := range []any{&owner, &other, &content, &game} {
		mustCreate(t, db.Create(value).Error)
	}

	movieWatched := entity.Watched{
		UserID:    owner.ID,
		ContentID: &content.ID,
		Status:    entity.FINISHED,
	}
	gameWatched := entity.Watched{
		UserID: owner.ID,
		GameID: &game.ID,
		Status: entity.PLANNED,
	}
	mustCreate(t, db.Create(&movieWatched).Error)
	mustCreate(t, db.Create(&gameWatched).Error)

	service := NewService(db, nil, nil, nil, nil)
	hours := uint(8)
	if _, err := service.updateWatched(owner.ID, movieWatched.ID, domain.WatchedUpdateRequest{
		PlaytimeHours: &hours,
	}); err == nil {
		t.Fatal("setting playtime on a movie unexpectedly succeeded")
	}
	if _, err := service.updateWatched(other.ID, gameWatched.ID, domain.WatchedUpdateRequest{
		PlaytimeHours: &hours,
	}); err == nil {
		t.Fatal("setting another user's playtime unexpectedly succeeded")
	}

	var unchanged entity.Watched
	mustCreate(t, db.First(&unchanged, gameWatched.ID).Error)
	if unchanged.PlaytimeHours != nil {
		t.Fatalf("other-user update changed playtime to %v", unchanged.PlaytimeHours)
	}
}
