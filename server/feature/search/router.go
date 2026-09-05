package search

import (
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

func publicSearchType(value string) (domain.SearchType, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(domain.SearchTypeMulti):
		return domain.SearchTypeMulti, true
	case string(domain.SearchTypeMovie):
		return domain.SearchTypeMovie, true
	case string(domain.SearchTypeShow), string(util.SupportedMediaShow):
		return domain.SearchTypeShow, true
	case string(domain.SearchTypePerson):
		return domain.SearchTypePerson, true
	case string(domain.SearchTypeGame):
		return domain.SearchTypeGame, true
	default:
		return "", false
	}
}

func watchedTypeForSearchType(searchType domain.SearchType) []util.SupportedMedia {
	switch searchType {
	case domain.SearchTypeMovie:
		return []util.SupportedMedia{util.SupportedMediaMovie}
	case domain.SearchTypeShow:
		return []util.SupportedMedia{util.SupportedMediaShow}
	case domain.SearchTypeGame:
		return []util.SupportedMedia{util.SupportedMediaGame}
	default:
		return nil
	}
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
		userId,
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

func (r *Router) getPublicFullSearch(
	c *gin.Context,
	ownerID uint,
	query string,
	searchType domain.SearchType,
	pp util.PaginationParams,
) {
	if r.service == nil {
		c.JSON(http.StatusInternalServerError, router.ErrorResponse{Error: "search is unavailable"})
		return
	}

	resp, err := r.service.Search(domain.SearchRequest{
		Type:         searchType,
		Query:        query,
		PreferMyList: false,
	}, pp, ownerID)
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
	rawType := c.Query("type")
	searchType, valid := publicSearchType(rawType)
	if scope == "list" && strings.Contains(rawType, ",") {
		// Preserve the original public-list API's comma-separated type filter.
		searchType, valid = domain.SearchTypeMulti, true
	}
	if !valid {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "invalid search type"})
		return
	}
	if scope == "all" || searchType == domain.SearchTypePerson {
		r.getPublicFullSearch(c, ownerID, query, searchType, pp)
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
	// The master search calls shows "show", while watched-list filters call
	// them "tv". A single selected result type owns the list filter here;
	// legacy comma-separated watched-list filters continue to bind unchanged.
	if !strings.Contains(rawType, ",") {
		wpr.FilterType = watchedTypeForSearchType(searchType)
	}

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
