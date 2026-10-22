package watched

import (
	"testing"

	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/internal/testutil"
)

func TestUpdateWatchedPreservesReplayAndMultipleActivities(t *testing.T) {
	db := testutil.SetupDB(t)
	user := entity.User{Username: "replay-owner", Password: "password"}
	game := entity.Game{IgdbID: 987, Name: "Replay Game"}
	mustCreate(t, db.Create(&user).Error)
	mustCreate(t, db.Create(&game).Error)
	watched := entity.Watched{UserID: user.ID, GameID: &game.ID, Status: entity.FINISHED}
	mustCreate(t, db.Create(&watched).Error)
	service := NewService(db, nil, nil, nil)
	hours := uint(12)
	response, err := service.UpdateWatched(user.ID, watched.ID, domain.WatchedUpdateRequest{
		Status: entity.FINISHED, Rating: 8.5, PlaytimeHours: &hours,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.NewActivities) != 3 {
		t.Fatalf("got %d activities, want rating, replay, and playtime", len(response.NewActivities))
	}
	plays := 0
	for _, a := range response.NewActivities {
		if a.CountAsPlay {
			plays++
		}
		if a.Type == entity.RATING_CHANGED && a.Data != `{"rating":8.5}` {
			t.Fatalf("fractional rating lost: %s", a.Data)
		}
	}
	if plays != 1 {
		t.Fatalf("got %d plays, want one replay", plays)
	}
	noPlay := false
	response, err = service.UpdateWatched(user.ID, watched.ID, domain.WatchedUpdateRequest{Status: entity.FINISHED, LetCountAsPlay: &noPlay})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.NewActivities) != 1 || response.NewActivities[0].CountAsPlay {
		t.Fatalf("LetCountAsPlay=false was ignored: %+v", response)
	}
}
