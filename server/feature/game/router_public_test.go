package game

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/cache"
	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/feature/watched/addedtocontent"
	"github.com/sbondCo/Watcharr/media/igdb"
	appRouter "github.com/sbondCo/Watcharr/router"
	"github.com/sbondCo/Watcharr/util"
)

type publicGameWatchedProvider struct {
	optionalCalled bool
}

func (p *publicGameWatchedProvider) UpdateWatchedLastViewedSeason(uint, uint, int) error {
	return nil
}

func (p *publicGameWatchedProvider) GetWatchedItemBySupportedMediaId(uint, uint, util.SupportedMedia) (entity.Watched, error) {
	return entity.Watched{}, nil
}

func (p *publicGameWatchedProvider) GetWatchedItemsBySupportedMediaIds(uint, []addedtocontent.IdToTypePair) ([]entity.Watched, error) {
	return nil, nil
}

func (p *publicGameWatchedProvider) GetOptionalPublicWatchedItem(uint, string, uint, util.SupportedMedia) (*entity.Watched, bool, error) {
	p.optionalCalled = true
	return nil, true, nil
}

func TestGetPublicGameDetailsAllowsGameNotOnOwnersList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	igdb.GameStore.Flush()
	t.Cleanup(igdb.GameStore.Flush)

	const mediaID = "8123"
	igdb.GameStore.Set(
		cache.CreateCacheKey("GameDetails", mediaID),
		[]igdb.GameDetailsResponse{{ID: 8123, Name: "Off-list game"}},
		time.Minute,
	)

	engine := gin.New()
	baseRouter := appRouter.NewBaseRouter(
		nil,
		engine.Group("/api"),
		&config.ServerConfig{},
	)
	provider := &publicGameWatchedProvider{}
	r := NewRouter(baseRouter, nil, provider)
	engine.GET("/api/public/users/:id/:username/content/game/:mediaId", r.GetPublicGameDetails)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/public/users/7/owner/content/game/"+mediaID,
		nil,
	)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !provider.optionalCalled {
		t.Fatal("expected optional public watched lookup")
	}

	var response domain.PublicMediaDetailsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Media.Name != "Off-list game" {
		t.Fatalf("expected game details, got %#v", response.Media)
	}
	if response.Media.Watched.ID != 0 {
		t.Fatalf("expected no watched entry, got %#v", response.Media.Watched)
	}
	var rawResponse struct {
		Media map[string]json.RawMessage `json:"media"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &rawResponse); err != nil {
		t.Fatalf("decode raw response: %v", err)
	}
	if _, exists := rawResponse.Media["watched"]; exists {
		t.Fatalf("expected watched data to be omitted: %s", recorder.Body.String())
	}
}
