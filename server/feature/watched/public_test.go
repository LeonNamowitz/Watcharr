package watched

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/internal/testutil"
	appRouter "github.com/sbondCo/Watcharr/router"
)

func TestPublicWatchedRoutesAreAnonymousAndRespectThoughtPrivacy(t *testing.T) {
	db := testutil.SetupDB(t)
	profilePrivate := false
	thoughtsPrivate := false
	owner := entity.User{
		Username: "owner",
		Password: "password",
		UserSettings: entity.UserSettings{
			Private:         &profilePrivate,
			PrivateThoughts: &thoughtsPrivate,
		},
	}
	mustCreate(t, db.Create(&owner).Error)

	content := entity.Content{
		TmdbID:   101,
		Title:    "Public Movie",
		Overview: "Public synopsis",
		Type:     entity.MOVIE,
	}
	mustCreate(t, db.Create(&content).Error)
	watchedItem := entity.Watched{
		UserID:    owner.ID,
		ContentID: &content.ID,
		Status:    entity.FINISHED,
		Rating:    8,
		Thoughts:  "Owner review",
	}
	mustCreate(t, db.Create(&watchedItem).Error)
	activity := entity.Activity{
		UserID:      owner.ID,
		WatchedID:   watchedItem.ID,
		Type:        entity.ADDED_WATCHED,
		Data:        `{}`,
		CountAsPlay: true,
	}
	mustCreate(t, db.Create(&activity).Error)

	showContent := entity.Content{
		TmdbID:   202,
		Title:    "Public Show",
		Overview: "Public show synopsis",
		Type:     entity.SHOW,
	}
	mustCreate(t, db.Create(&showContent).Error)
	showWatched := entity.Watched{
		UserID:    owner.ID,
		ContentID: &showContent.ID,
		Status:    entity.HOLD,
	}
	mustCreate(t, db.Create(&showWatched).Error)
	mustCreate(t, db.Create(&entity.WatchedSeason{
		UserID:       owner.ID,
		WatchedID:    showWatched.ID,
		SeasonNumber: 2,
		Status:       entity.WATCHING,
		Rating:       9,
	}).Error)
	mustCreate(t, db.Create(&entity.WatchedEpisode{
		UserID:        owner.ID,
		WatchedID:     showWatched.ID,
		SeasonNumber:  2,
		EpisodeNumber: 1,
		Status:        entity.FINISHED,
		Rating:        8,
	}).Error)

	service := NewService(db, nil, nil, nil, watchedSearchUserProvider{})
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/api")
	br := appRouter.NewBaseRouter(db, api, &config.ServerConfig{JWT_SECRET: "test"})
	NewRouter(br, service).AddRoutes()

	listRecorder := request(t, engine, http.MethodGet,
		"/api/public/users/"+strconv.FormatUint(uint64(owner.ID), 10)+"/owner/watched?page=1")
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("anonymous list status = %d, want 200; body=%s",
			listRecorder.Code, listRecorder.Body.String())
	}
	if strings.Contains(listRecorder.Body.String(), "Owner review") {
		t.Fatalf("public list leaked thoughts: %s", listRecorder.Body.String())
	}

	detailPath := "/api/public/users/" + strconv.FormatUint(uint64(owner.ID), 10) +
		"/owner/media/movie/101"
	detailRecorder := request(t, engine, http.MethodGet, detailPath)
	if detailRecorder.Code != http.StatusOK {
		t.Fatalf("anonymous detail status = %d, want 200; body=%s",
			detailRecorder.Code, detailRecorder.Body.String())
	}
	var detail domain.PublicMediaDetailsResponse
	if err := json.Unmarshal(detailRecorder.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decoding public detail failed: %v", err)
	}
	if !detail.ThoughtsPublic || detail.Media.Watched.Thoughts != "Owner review" {
		t.Fatalf("public detail thoughts = (%v, %q), want (true, %q)",
			detail.ThoughtsPublic, detail.Media.Watched.Thoughts, "Owner review")
	}
	if len(detail.Media.Watched.Activity) != 1 || detail.Media.Watched.Plays != 1 {
		t.Fatalf("public detail activity = %d and plays = %d, want 1 and 1",
			len(detail.Media.Watched.Activity), detail.Media.Watched.Plays)
	}

	showDetailPath := "/api/public/users/" + strconv.FormatUint(uint64(owner.ID), 10) +
		"/owner/media/tv/202"
	showDetailRecorder := request(t, engine, http.MethodGet, showDetailPath)
	if showDetailRecorder.Code != http.StatusOK {
		t.Fatalf("anonymous show detail status = %d, want 200; body=%s",
			showDetailRecorder.Code, showDetailRecorder.Body.String())
	}
	var showDetail domain.PublicMediaDetailsResponse
	if err := json.Unmarshal(showDetailRecorder.Body.Bytes(), &showDetail); err != nil {
		t.Fatalf("decoding public show detail failed: %v", err)
	}
	if showDetail.Media.Watched.WatchingSeason != "S2E1" {
		t.Fatalf("public show progress = %q, want S2E1",
			showDetail.Media.Watched.WatchingSeason)
	}
	if len(showDetail.Media.Watched.WatchedSeasons) != 1 ||
		showDetail.Media.Watched.WatchedSeasons[0].Rating != 9 {
		t.Fatalf("public watched seasons = %#v, want one season rated 9",
			showDetail.Media.Watched.WatchedSeasons)
	}
	if len(showDetail.Media.Watched.WatchedEpisodes) != 1 ||
		showDetail.Media.Watched.WatchedEpisodes[0].Rating != 8 {
		t.Fatalf("public watched episodes = %#v, want one episode rated 8",
			showDetail.Media.Watched.WatchedEpisodes)
	}

	thoughtsPrivate = true
	mustCreate(t, db.Model(&entity.User{}).Where("id = ?", owner.ID).
		Update("private_thoughts", thoughtsPrivate).Error)
	detailRecorder = request(t, engine, http.MethodGet, detailPath)
	if detailRecorder.Code != http.StatusOK {
		t.Fatalf("private-thought detail status = %d, want 200; body=%s",
			detailRecorder.Code, detailRecorder.Body.String())
	}
	detail = domain.PublicMediaDetailsResponse{}
	if err := json.Unmarshal(detailRecorder.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decoding private-thought detail failed: %v", err)
	}
	if detail.ThoughtsPublic || detail.Media.Watched.Thoughts != "" {
		t.Fatalf("private thoughts leaked: public=%v thoughts=%q",
			detail.ThoughtsPublic, detail.Media.Watched.Thoughts)
	}

	profilePrivate = true
	mustCreate(t, db.Model(&entity.User{}).Where("id = ?", owner.ID).
		Update("private", profilePrivate).Error)
	privateRecorder := request(t, engine, http.MethodGet,
		"/api/public/users/"+strconv.FormatUint(uint64(owner.ID), 10)+"/owner/watched?page=1")
	if privateRecorder.Code != http.StatusForbidden {
		t.Fatalf("private list status = %d, want 403", privateRecorder.Code)
	}

	mutationRecorder := request(t, engine, http.MethodPost, "/api/watched")
	if mutationRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous mutation status = %d, want 401", mutationRecorder.Code)
	}
}

func TestPublicWatchedRouteRejectsMismatchedOwner(t *testing.T) {
	db := testutil.SetupDB(t)
	profilePrivate := false
	owner := entity.User{
		Username:     "owner",
		Password:     "password",
		UserSettings: entity.UserSettings{Private: &profilePrivate},
	}
	mustCreate(t, db.Create(&owner).Error)

	service := NewService(db, nil, nil, nil, watchedSearchUserProvider{})
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/api")
	br := appRouter.NewBaseRouter(db, api, &config.ServerConfig{JWT_SECRET: "test"})
	NewRouter(br, service).AddRoutes()

	recorder := request(t, engine, http.MethodGet,
		"/api/public/users/"+strconv.FormatUint(uint64(owner.ID), 10)+"/someone-else/watched?page=1")
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("mismatched owner status = %d, want 403", recorder.Code)
	}
}

func request(
	t *testing.T,
	h http.Handler,
	method string,
	path string,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, req)
	return recorder
}
