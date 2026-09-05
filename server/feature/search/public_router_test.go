package search

import (
	"encoding/json"
	"errors"
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
	pageCalls         int
	validateErr       error
	watchedItems      []entity.Watched
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
	return p.watchedItems, nil
}

func (p *publicSearchWatchedProvider) GetWatchedPage(
	_ uint,
	_ util.PaginationParams,
	request domain.WatchedGetPageRequest,
	_ *domain.WatchedGetPageExtraProps,
) (util.PaginationResponse[entity.Watched, util.None], error) {
	p.pageCalls++
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
	return p.validateErr
}

type publicFullSearchProvider struct {
	request domain.SearchRequest
	userID  uint
	calls   int
	resp    domain.SearchResponse
	err     error
}

func (p *publicFullSearchProvider) Search(
	request domain.SearchRequest,
	_ util.PaginationParams,
	userID uint,
) (domain.SearchResponse, error) {
	p.request = request
	p.userID = userID
	p.calls++
	return p.resp, p.err
}

func newPublicSearchEngine(
	t *testing.T,
	service SearchProvider,
	provider WatchedProvider,
) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/api")
	br := appRouter.NewBaseRouter(nil, api, &config.ServerConfig{JWT_SECRET: "test"})
	NewRouter(br, service, provider).AddRoutes()
	return engine
}

func TestPublicListSearchDoesNotRequireAuthentication(t *testing.T) {
	provider := &publicSearchWatchedProvider{}
	engine := newPublicSearchEngine(t, nil, provider)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/public/users/7/alice/search?query=dune&page=1&type=show&status=dropped&sort=RATING&sortDir=asc",
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
	if len(provider.request.FilterType) != 1 || provider.request.FilterType[0] != "tv" ||
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

func TestPublicListFullSearchUsesOwnerDataWithoutPrivateFields(t *testing.T) {
	content := entity.Content{TmdbID: 202, Type: entity.MOVIE}
	provider := &publicSearchWatchedProvider{watchedItems: []entity.Watched{{
		Content:  &content,
		Status:   entity.FINISHED,
		Rating:   9,
		Thoughts: "private review",
	}}}
	service := &publicFullSearchProvider{resp: domain.SearchResponse{
		PaginationResponse: util.PaginationResponse[domain.Media, domain.SearchResponseMeta]{
			PaginationParams: util.PaginationParams{Page: 1, Limit: 40},
			TotalPages:       1,
			TotalResults:     2,
			Results: []domain.Media{
				{Type: domain.MediaTypeTMDBMovie, IDs: domain.MediaIDs{TMDB: 202}, Name: "Dune"},
				{Type: domain.MediaTypeTMDBPerson, IDs: domain.MediaIDs{TMDB: 303}, Name: "Actor"},
			},
		},
	}}
	engine := newPublicSearchEngine(t, service, provider)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/public/users/7/alice/search?query=dune&page=1&scope=all&type=movie",
		nil,
	)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("full public search status = %d, want 200; body=%s",
			recorder.Code, recorder.Body.String())
	}
	if service.calls != 1 || service.userID != 7 {
		t.Fatalf("full search calls = %d for user %d, want one call for owner 7",
			service.calls, service.userID)
	}
	if service.request.Type != domain.SearchTypeMovie || service.request.PreferMyList {
		t.Fatalf("full search request = %#v", service.request)
	}
	if provider.pageCalls != 0 {
		t.Fatalf("list search called %d times during full search", provider.pageCalls)
	}

	var response domain.SearchResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode full search response: %v", err)
	}
	if len(response.Results) != 2 || response.Results[0].Watched.Rating != 9 {
		t.Fatalf("owner watched data missing from response: %#v", response.Results)
	}
	if response.Results[0].Watched.Thoughts != "" || strings.Contains(recorder.Body.String(), "private review") {
		t.Fatalf("full public search leaked private review: %s", recorder.Body.String())
	}
	if response.Meta.FromMyList {
		t.Fatal("full public search was marked as list-scoped")
	}
	if service.resp.Results[0].Watched.Rating != 0 {
		t.Fatal("owner watched data mutated the external search response")
	}
}

func TestPublicListPersonSearchExpandsWithoutExplicitScope(t *testing.T) {
	provider := &publicSearchWatchedProvider{}
	service := &publicFullSearchProvider{}
	engine := newPublicSearchEngine(t, service, provider)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/public/users/7/alice/search?query=leon&type=person",
		nil,
	)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("person search status = %d, want 200; body=%s",
			recorder.Code, recorder.Body.String())
	}
	if service.calls != 1 || service.request.Type != domain.SearchTypePerson {
		t.Fatalf("person search request = %#v; calls=%d", service.request, service.calls)
	}
	if provider.pageCalls != 0 {
		t.Fatalf("person search called owner list %d times", provider.pageCalls)
	}
}

func TestPublicFullSearchRejectsPrivateOwnerBeforeExternalSearch(t *testing.T) {
	provider := &publicSearchWatchedProvider{validateErr: errors.New("private list")}
	service := &publicFullSearchProvider{}
	engine := newPublicSearchEngine(t, service, provider)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/public/users/7/alice/search?query=dune&scope=all",
		nil,
	)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("private owner status = %d, want 403", recorder.Code)
	}
	if service.calls != 0 {
		t.Fatalf("external search called %d times for private owner", service.calls)
	}
}

func TestGameSearchReturnsEmptyWhenIGDBIsDisabled(t *testing.T) {
	service := NewService(nil, &config.ServerConfig{}, nil, nil)
	response, err := service.Search(
		domain.SearchRequest{Type: domain.SearchTypeGame, Query: "hades"},
		util.PaginationParams{Page: 1, Limit: 40},
		7,
	)
	if err != nil {
		t.Fatalf("disabled IGDB search returned error: %v", err)
	}
	if len(response.Results) != 0 || response.TotalResults != 0 {
		t.Fatalf("disabled IGDB search returned results: %#v", response)
	}
}

func TestPublicListSearchKeepsLegacyCommaSeparatedTypes(t *testing.T) {
	provider := &publicSearchWatchedProvider{}
	engine := newPublicSearchEngine(t, nil, provider)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/public/users/7/alice/search?query=dune&type=tv,movie,game",
		nil,
	)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("legacy list search status = %d, want 200; body=%s",
			recorder.Code, recorder.Body.String())
	}
	if len(provider.request.FilterType) != 3 ||
		provider.request.FilterType[0] != util.SupportedMediaShow ||
		provider.request.FilterType[1] != util.SupportedMediaMovie ||
		provider.request.FilterType[2] != util.SupportedMediaGame {
		t.Fatalf("legacy type filters = %#v", provider.request.FilterType)
	}
}

func TestPublicSearchRejectsInvalidScopeAndType(t *testing.T) {
	for name, query := range map[string]string{
		"scope": "query=dune&scope=somewhere",
		"type":  "query=dune&type=book",
	} {
		t.Run(name, func(t *testing.T) {
			provider := &publicSearchWatchedProvider{}
			service := &publicFullSearchProvider{}
			engine := newPublicSearchEngine(t, service, provider)
			req := httptest.NewRequest(
				http.MethodGet,
				"/api/public/users/7/alice/search?"+query,
				nil,
			)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("invalid %s status = %d, want 400", name, recorder.Code)
			}
			if service.calls != 0 || provider.pageCalls != 0 {
				t.Fatalf("invalid %s reached a search provider", name)
			}
		})
	}
}

func TestPublicFullSearchReturnsProviderError(t *testing.T) {
	provider := &publicSearchWatchedProvider{}
	service := &publicFullSearchProvider{err: errors.New("provider failed")}
	engine := newPublicSearchEngine(t, service, provider)
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/public/users/7/alice/search?query=dune&scope=all",
		nil,
	)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("provider error status = %d, want 400", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "provider failed") {
		t.Fatalf("provider error response = %s", recorder.Body.String())
	}
}
