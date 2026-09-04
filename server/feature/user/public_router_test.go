package user

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/internal/testutil"
	appRouter "github.com/sbondCo/Watcharr/router"
)

func TestPublicUserInfoDoesNotRequireAuthentication(t *testing.T) {
	db := testutil.SetupDB(t)
	profilePrivate := false
	ratingSystem := 1
	ratingStep := 2
	owner := entity.User{
		Username: "alice",
		Password: "password",
		Bio:      "Public bio",
		UserSettings: entity.UserSettings{
			Private:      &profilePrivate,
			RatingSystem: &ratingSystem,
			RatingStep:   &ratingStep,
		},
	}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("creating user failed: %v", err)
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/api")
	br := appRouter.NewBaseRouter(db, api, &config.ServerConfig{JWT_SECRET: "test"})
	NewRouter(br, NewService(db), nil).AddRoutes()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/public/users/"+strconv.FormatUint(uint64(owner.ID), 10)+"/alice",
		nil,
	)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("anonymous public user status = %d, want 200; body=%s",
			recorder.Code, recorder.Body.String())
	}

	var response entity.PublicUser
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decoding public user response failed: %v", err)
	}
	if response.RatingSystem == nil || *response.RatingSystem != ratingSystem {
		t.Fatalf("rating system = %v, want %d", response.RatingSystem, ratingSystem)
	}
	if response.RatingStep == nil || *response.RatingStep != ratingStep {
		t.Fatalf("rating step = %v, want %d", response.RatingStep, ratingStep)
	}
}
