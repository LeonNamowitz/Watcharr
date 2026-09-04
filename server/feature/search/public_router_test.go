package search

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/feature/watched/addedtocontent"
	appRouter "github.com/sbondCo/Watcharr/router"
	"github.com/sbondCo/Watcharr/util"
)

type publicSearchWatchedProvider struct {
	validatedUserId   uint
	validatedUsername string
	request           domain.WatchedGetPageRequest
}

func (p *publicSearchWatchedProvider) GetWatchedItemBySupportedMediaId(
	uint,
	uint,
	util.SupportedMedia,
) (entity.Watched, error) {
	return entity.Watched{}, nil
}

func (p *publicSearchWatchedProvider) GetWatchedItemsBySupportedMediaIds(
	uint,
	[]addedtocontent.IdToTypePair,
) ([]entity.Watched, error) {
	return nil, nil
}

func (p *publicSearchWatchedProvider) GetWatchedPage(
	_ uint,
	_ util.PaginationParams,
	request domain.WatchedGetPageRequest,
	_ *domain.WatchedGetPageExtraProps,
) (util.PaginationResponse[entity.Watched, util.None], error) {
	p.request = request
	content := entity.Content{TmdbID: 101, Title: "Dune", Type: entity.MOVIE}
	return util.PaginationResponse[entity.Watched, util.None]{
		PaginationParams: util.PaginationParams{Page: 1, Limit: 40},
		TotalPages:       1,
		TotalResults:     1,
		Results: []entity.Watched{{
			Content:  &content,
			Status:   entity.FINISHED,
			Rating:   8,
			Thoughts: "must not appear in list search",
		}},
	}, nil
}

func (p *publicSearchWatchedProvider) ValidatePublicWatchedList(
	userId uint,
	username string,
) error {
	p.validatedUserId = userId
	p.validatedUsername = username
	return nil
}

func TestPublicListSearchDoesNotRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/api")
	provider := &publicSearchWatchedProvider{}
	br := appRouter.NewBaseRouter(nil, api, &config.ServerConfig{JWT_SECRET: "test"})
	NewRouter(br, nil, provider).AddRoutes()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/public/users/7/alice/search?query=dune&page=1&type=game&status=dropped&sort=RATING&sortDir=asc",
		nil,
	)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("anonymous public search status = %d, want 200; body=%s",
			recorder.Code, recorder.Body.String())
	}
	if provider.validatedUserId != 7 || provider.validatedUsername != "alice" {
		t.Fatalf("validated owner = (%d, %q), want (7, %q)",
			provider.validatedUserId, provider.validatedUsername, "alice")
	}
	if len(provider.request.FilterType) != 1 || provider.request.FilterType[0] != "game" ||
		len(provider.request.FilterStatus) != 1 || provider.request.FilterStatus[0] != "dropped" ||
		provider.request.Sort != domain.WatchedSortRating ||
		provider.request.SortDir != domain.WatchedSortDirAsc {
		t.Fatalf("public search controls were not bound: %#v", provider.request)
	}
	if body := recorder.Body.String(); body == "" {
		t.Fatal("anonymous public search returned an empty body")
	} else if strings.Contains(body, "must not appear") {
		t.Fatalf("public list search leaked thoughts: %s", body)
	}
}
