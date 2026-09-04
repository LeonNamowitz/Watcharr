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

type Router struct {
	br              *router.BaseRouter
	service         *Service
	watchedProvider WatchedProvider
}

func NewRouter(br *router.BaseRouter, service *Service, watchedProvider WatchedProvider) *Router {
	return &Router{
		br,
		service,
		watchedProvider,
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

// Search a public user's watched titles with owner-scoped pagination.
func (r *Router) GetPublicListSearch(c *gin.Context) {
	pp := c.MustGet("paginationParams").(util.PaginationParams)
	query := strings.TrimSpace(c.Query("query"))
	if query == "" {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "a query is required"})
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
