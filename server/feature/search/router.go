package search

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/feature/auth/authmiddleware"
	"github.com/sbondCo/Watcharr/feature/watched/addedtocontent"
	"github.com/sbondCo/Watcharr/router"
	"github.com/sbondCo/Watcharr/util"
)

type WatchedProvider interface {
	GetWatchedItemBySupportedMediaId(userId uint, id uint, t util.SupportedMedia) (entity.Watched, error)
	GetWatchedItemsBySupportedMediaIds(userId uint, c []addedtocontent.IdToTypePair) ([]entity.Watched, error)
	GetWatchedPage(userId uint, pp util.PaginationParams, wr domain.WatchedGetPageRequest, extraProps *domain.WatchedGetPageExtraProps) (util.PaginationResponse[entity.Watched, util.None], error)
	ValidatePublicWatchedList(userId uint, username string) error
}

type SearchProvider interface {
	Search(r domain.SearchRequest, pp util.PaginationParams, userId uint) (domain.SearchResponse, error)
}

type Router struct {
	br              *router.BaseRouter
	service         SearchProvider
	watchedProvider WatchedProvider
}

func NewRouter(br *router.BaseRouter, service SearchProvider, watchedProvider WatchedProvider) *Router {
	return &Router{
		br,
		service,
		watchedProvider,
	}
}

func parseSearchTypes(value string) ([]domain.SearchType, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == string(domain.SearchTypeMulti) {
		return nil, true
	}

	searchTypes := make([]domain.SearchType, 0, 3)
	seen := make(map[domain.SearchType]bool)
	for _, rawType := range strings.Split(value, ",") {
		var searchType domain.SearchType
		switch strings.TrimSpace(rawType) {
		case string(domain.SearchTypeMovie):
			searchType = domain.SearchTypeMovie
		case string(domain.SearchTypeShow), string(util.SupportedMediaShow):
			searchType = domain.SearchTypeShow
		case string(domain.SearchTypePerson):
			searchType = domain.SearchTypePerson
		case string(domain.SearchTypeGame):
			searchType = domain.SearchTypeGame
		default:
			return nil, false
		}
		if !seen[searchType] {
			seen[searchType] = true
			searchTypes = append(searchTypes, searchType)
		}
	}

	if len(searchTypes) == 0 || (seen[domain.SearchTypePerson] && len(searchTypes) != 1) {
		return nil, false
	}
	return searchTypes, true
}

func watchedTypesForSearchTypes(searchTypes []domain.SearchType) []util.SupportedMedia {
	filterTypes := make([]util.SupportedMedia, 0, len(searchTypes))
	for _, searchType := range searchTypes {
		switch searchType {
		case domain.SearchTypeMovie:
			filterTypes = append(filterTypes, util.SupportedMediaMovie)
		case domain.SearchTypeShow:
			filterTypes = append(filterTypes, util.SupportedMediaShow)
		case domain.SearchTypeGame:
			filterTypes = append(filterTypes, util.SupportedMediaGame)
		}
	}
	return filterTypes
}

func mergeSearchResponses(
	responses []domain.SearchResponse,
	pp util.PaginationParams,
) domain.SearchResponse {
	merged := domain.SearchResponse{
		PaginationResponse: util.PaginationResponse[domain.Media, domain.SearchResponseMeta]{
			PaginationParams: pp,
		},
	}
	maxResults := 0
	for _, response := range responses {
		merged.TotalResults += response.TotalResults
		if response.TotalPages > merged.TotalPages {
			merged.TotalPages = response.TotalPages
		}
		if len(response.Results) > maxResults {
			maxResults = len(response.Results)
		}
	}
	for resultIndex := 0; resultIndex < maxResults; resultIndex++ {
		for _, response := range responses {
			if resultIndex < len(response.Results) {
				merged.Results = append(merged.Results, response.Results[resultIndex])
			}
		}
	}
	return merged
}

func mediaMatchesSearchType(mediaType domain.MediaType, searchType domain.SearchType) bool {
	switch searchType {
	case domain.SearchTypeMovie:
		return mediaType == domain.MediaTypeTMDBMovie
	case domain.SearchTypeShow:
		return mediaType == domain.MediaTypeTMDBShow
	case domain.SearchTypePerson:
		return mediaType == domain.MediaTypeTMDBPerson
	case domain.SearchTypeGame:
		return mediaType == domain.MediaTypeIGDBGame
	default:
		return true
	}
}

func filterSearchResponse(
	response domain.SearchResponse,
	searchType domain.SearchType,
) domain.SearchResponse {
	results := make([]domain.Media, 0, len(response.Results))
	for _, result := range response.Results {
		if mediaMatchesSearchType(result.Type, searchType) {
			results = append(results, result)
		}
	}
	if len(results) != len(response.Results) {
		response.Results = results
		response.TotalResults = int64(len(results))
		if len(results) == 0 {
			response.TotalPages = 0
		} else {
			response.TotalPages = 1
		}
	}
	return response
}

func (r *Router) AddRoutes() {
	search := r.br.Router.Group("/search").Use(authmiddleware.AuthRequired(nil, r.br.Cfg))

	// Master search
	search.GET("", router.PaginatedRequest(true), r.GetSearch)
	// Kept for compatibility with authenticated clients.
	search.GET("/list/:id/:username", router.PaginatedRequest(true), r.GetPublicListSearch)
	// Search within a shared list without creating a user session.
	r.br.Router.GET(
		"/public/users/:id/:username/search",
		router.PaginatedRequest(true),
		r.GetPublicListSearch,
	)
}

// NOTE: The handler functions use `copier` to copy values from the response
// structs into a new one that includes the user "Watched" data.
// This was done to avoid adding Watched data to the response structs, as they
// are cached in our in-mem cache, which could cause references to pollute the cache
// resulting in user data being leaked to others.
// We are doing to to explicitly not let that case happen.

func (r *Router) GetSearch(c *gin.Context) {
	userId := c.MustGet("userId").(uint)
	pp := c.MustGet("paginationParams").(util.PaginationParams)
	scope, hasScope := c.GetQuery("scope")
	if hasScope {
		r.getScopedSearch(c, userId, pp, scope)
		return
	}

	// Preserve the original preferMyList API for clients that do not opt in to
	// the explicit list/global scope used by the web interface.
	req := domain.SearchRequest{
		// Defaults...
		Type: domain.SearchTypeMulti,
	}
	if err := c.ShouldBind(&req); err != nil {
		slog.Error("GetSearch: ShouldBind for request params failed!", "error", err)
		c.JSON(
			http.StatusBadRequest,
			router.ErrorResponse{
				Error: "failed to get request parameters or they are invalid",
			},
		)
		return
	}
	resp, err := r.service.Search(req, pp, userId)
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}

	// If we got results to show from our list instead of a normal search,
	// then we can just return the resp here since it will already include
	// our watched info & it is not cached so we don't need to use copier.
	if resp.Meta.FromMyList {
		slog.Debug("GetSearch: FromMyList=true, returning response without further processing.")
		c.JSON(http.StatusOK, resp)
		return
	}

	r.addPrivateWatchedData(c, userId, resp)
}

func (r *Router) addPrivateWatchedData(
	c *gin.Context,
	userID uint,
	resp domain.SearchResponse,
) {
	ww := domain.SearchResponse{}
	if err := copier.Copy(&ww, &resp); err != nil {
		slog.Error("GetSearch: Failed to copy", "error", err)
		c.JSON(
			http.StatusInternalServerError,
			router.ErrorResponse{Error: "failed to prepare response"},
		)
		return
	}
	if err := addedtocontent.AddList(
		r.watchedProvider,
		userID,
		ww.Results,
		func(i int, w *entity.Watched) {
			ww.Results[i].Watched = domain.NewWatchedDtoForLists(w)
		},
	); err != nil {
		slog.Error("GetSearch: Failed to add watched to content!", "error", err)
		c.JSON(
			http.StatusInternalServerError,
			router.ErrorResponse{Error: "failed to add watched data to response"},
		)
		return
	}
	c.JSON(http.StatusOK, ww)
}

func (r *Router) getGlobalSearch(
	query string,
	searchTypes []domain.SearchType,
	pp util.PaginationParams,
	userID uint,
) (domain.SearchResponse, error) {
	if r.service == nil {
		return domain.SearchResponse{}, errors.New("search is unavailable")
	}
	if len(searchTypes) == 0 {
		searchTypes = []domain.SearchType{domain.SearchTypeMulti}
	}
	if len(searchTypes) == 1 {
		response, err := r.service.Search(domain.SearchRequest{
			Type:         searchTypes[0],
			Query:        query,
			PreferMyList: false,
		}, pp, userID)
		return filterSearchResponse(response, searchTypes[0]), err
	}

	responses := make([]domain.SearchResponse, 0, len(searchTypes))
	for _, searchType := range searchTypes {
		// IGDB search is not paginated. This mirrors the existing broad search,
		// which only adds games to its first page.
		if searchType == domain.SearchTypeGame && pp.Page > 1 {
			continue
		}
		response, err := r.service.Search(domain.SearchRequest{
			Type:         searchType,
			Query:        query,
			PreferMyList: false,
		}, pp, userID)
		if err != nil {
			return domain.SearchResponse{}, err
		}
		responses = append(responses, filterSearchResponse(response, searchType))
	}
	return mergeSearchResponses(responses, pp), nil
}

func (r *Router) getScopedSearch(
	c *gin.Context,
	userID uint,
	pp util.PaginationParams,
	rawScope string,
) {
	query := strings.TrimSpace(c.Query("query"))
	if query == "" {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "a query is required"})
		return
	}
	scope := strings.ToLower(strings.TrimSpace(rawScope))
	if scope != "list" && scope != "all" {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "invalid search scope"})
		return
	}
	searchTypes, valid := parseSearchTypes(c.Query("type"))
	if !valid {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "invalid search type"})
		return
	}
	if scope == "list" {
		if len(searchTypes) == 1 && searchTypes[0] == domain.SearchTypePerson {
			c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "people search is global only"})
			return
		}
		wpr := domain.WatchedGetPageRequest{
			Sort:    domain.WatchedSortDateAdded,
			SortDir: domain.WatchedSortDirAsc,
		}
		if err := c.ShouldBindQuery(&wpr); err != nil {
			c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "failed to get request parameters or they are invalid"})
			return
		}
		wpr.FilterType = watchedTypesForSearchTypes(searchTypes)
		wp, err := r.watchedProvider.GetWatchedPage(
			userID,
			pp,
			wpr,
			&domain.WatchedGetPageExtraProps{Query: query},
		)
		if err != nil {
			c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "failed to search list"})
			return
		}
		c.JSON(http.StatusOK, domain.SearchResponse{
			PaginationResponse: util.PaginationResponse[domain.Media, domain.SearchResponseMeta]{
				PaginationParams: wp.PaginationParams,
				TotalPages:       wp.TotalPages,
				TotalResults:     wp.TotalResults,
				Results:          domain.NewWatchedGetPageResponse(wp.Results),
				Meta:             domain.SearchResponseMeta{FromMyList: true},
			},
		})
		return
	}

	resp, err := r.getGlobalSearch(query, searchTypes, pp, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	r.addPrivateWatchedData(c, userID, resp)
}

func (r *Router) getPublicFullSearch(
	c *gin.Context,
	ownerID uint,
	query string,
	searchTypes []domain.SearchType,
	pp util.PaginationParams,
) {
	resp, err := r.getGlobalSearch(query, searchTypes, pp, ownerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}

	publicResp := domain.SearchResponse{}
	if err := copier.CopyWithOption(
		&publicResp,
		&resp,
		copier.Option{DeepCopy: true},
	); err != nil {
		slog.Error("GetPublicListSearch: Failed to copy full search response", "error", err)
		c.JSON(
			http.StatusInternalServerError,
			router.ErrorResponse{Error: "failed to prepare response"},
		)
		return
	}
	ownerContent := make([]domain.Media, 0, len(publicResp.Results))
	ownerContentIndexes := make([]int, 0, len(publicResp.Results))
	for i, result := range publicResp.Results {
		switch result.Type {
		case domain.MediaTypeTMDBMovie,
			domain.MediaTypeTMDBShow,
			domain.MediaTypeIGDBGame:
			ownerContent = append(ownerContent, result)
			ownerContentIndexes = append(ownerContentIndexes, i)
		}
	}
	if err := addedtocontent.AddList(
		r.watchedProvider,
		ownerID,
		ownerContent,
		func(i int, w *entity.Watched) {
			publicResp.Results[ownerContentIndexes[i]].Watched = domain.NewWatchedDtoForPublicLists(w)
		},
	); err != nil {
		slog.Error("GetPublicListSearch: Failed to add owner watched data", "error", err)
		c.JSON(
			http.StatusInternalServerError,
			router.ErrorResponse{Error: "failed to add watched data to response"},
		)
		return
	}
	publicResp.Meta.FromMyList = false
	c.JSON(http.StatusOK, publicResp)
}

// Search a public user's watched titles, or expand to the configured external
// providers, while keeping watched metadata scoped to the public list owner.
func (r *Router) GetPublicListSearch(c *gin.Context) {
	pp := c.MustGet("paginationParams").(util.PaginationParams)
	query := strings.TrimSpace(c.Query("query"))
	if query == "" {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "a query is required"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "invalid user id"})
		return
	}
	ownerID := uint(id)
	if err := r.watchedProvider.ValidatePublicWatchedList(
		ownerID,
		c.Param("username"),
	); err != nil {
		c.JSON(http.StatusForbidden, router.ErrorResponse{Error: err.Error()})
		return
	}

	scope := strings.ToLower(strings.TrimSpace(c.DefaultQuery("scope", "list")))
	if scope != "list" && scope != "all" {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "invalid search scope"})
		return
	}
	searchTypes, valid := parseSearchTypes(c.Query("type"))
	if !valid {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "invalid search type"})
		return
	}
	isPersonSearch := len(searchTypes) == 1 && searchTypes[0] == domain.SearchTypePerson
	if scope == "all" || isPersonSearch {
		r.getPublicFullSearch(c, ownerID, query, searchTypes, pp)
		return
	}

	wpr := domain.WatchedGetPageRequest{
		Sort:    domain.WatchedSortDateAdded,
		SortDir: domain.WatchedSortDirAsc,
	}
	if err := c.ShouldBindQuery(&wpr); err != nil {
		c.JSON(
			http.StatusBadRequest,
			router.ErrorResponse{
				Error: "failed to get request parameters or they are invalid",
			},
		)
		return
	}
	// Search controls use "show", while watched-list filters use "tv".
	// Normalize both current and legacy values before querying the list.
	wpr.FilterType = watchedTypesForSearchTypes(searchTypes)

	wp, err := r.watchedProvider.GetWatchedPage(
		ownerID,
		pp,
		wpr,
		&domain.WatchedGetPageExtraProps{Query: query},
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "failed to search list"})
		return
	}

	resp := domain.SearchResponse{
		PaginationResponse: util.PaginationResponse[domain.Media, domain.SearchResponseMeta]{
			PaginationParams: wp.PaginationParams,
			TotalPages:       wp.TotalPages,
			TotalResults:     wp.TotalResults,
			Results:          domain.NewWatchedPublicGetPageResponse(wp.Results),
			Meta:             domain.SearchResponseMeta{FromMyList: true},
		},
	}
	c.JSON(http.StatusOK, resp)
}
