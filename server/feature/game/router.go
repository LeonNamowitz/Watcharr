package game

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/feature/auth/authmiddleware"
	"github.com/sbondCo/Watcharr/feature/watched/addedtocontent"
	"github.com/sbondCo/Watcharr/media/igdb"
	"github.com/sbondCo/Watcharr/router"
	"github.com/sbondCo/Watcharr/util"
)

type WatchedProvider interface {
	UpdateWatchedLastViewedSeason(userId uint, id uint, seasonNum int) error
	GetWatchedItemBySupportedMediaId(userId uint, id uint, t util.SupportedMedia) (entity.Watched, error)
	GetWatchedItemsBySupportedMediaIds(userId uint, c []addedtocontent.IdToTypePair) ([]entity.Watched, error)
	GetPublicWatchedItem(userId uint, username string, mediaId uint, mediaType util.SupportedMedia) (entity.Watched, bool, error)
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
	gamer := r.br.Router.Group("/game").Use(authmiddleware.AuthRequired(nil, r.br.Cfg))
	publicOwnerContent := r.br.Router.Group("/public/users/:id/:username/content")

	// TODO This config init can be moved to NewRouter, then `gdb` can be accessible in Router for all service funcs.
	r.br.Cfg.TWITCH.OnTokenRefreshed(func() {
		// Save new token to config when we refresh it.
		slog.Debug("GameRoutes: token refreshed.. saving to config.")
		if err := r.br.Cfg.Write(); err != nil {
			slog.Error("GameRoutes: failed to save refreshed token to config.", "error", err)
		}
	})
	err := r.br.Cfg.TWITCH.Init()
	// Save cfg if init succeeded, this will save our access token
	if err != nil {
		slog.Error("GameRoutes: Twitch init failed!", "error", err)
	}

	// Game details for game page
	gamer.GET("/:id", r.GetGameDetails)
	publicOwnerContent.GET("/game/:mediaId", r.GetPublicGameDetails)

	// IMPORTANT: Routes below only for admins!
	gamer.Use(authmiddleware.AuthRequired(r.br.DB, r.br.Cfg), authmiddleware.AdminRequired())
	{
		gamer.POST("/config", r.UpdateConfig)
	}
}

// GetPublicGameDetails returns the normal game page data with the public list
// owner's watched entry attached instead of the visitor's.
func (r *Router) GetPublicGameDetails(c *gin.Context) {
	userId, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "invalid user id"})
		return
	}
	mediaId, err := strconv.ParseUint(c.Param("mediaId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "invalid media id"})
		return
	}
	watched, thoughtsPublic, err := r.watchedProvider.GetPublicWatchedItem(
		uint(userId),
		c.Param("username"),
		uint(mediaId),
		util.SupportedMediaGame,
	)
	if err != nil {
		c.JSON(http.StatusForbidden, router.ErrorResponse{Error: "failed fetching the public list item"})
		return
	}
	content, err := r.br.Cfg.TWITCH.GameDetails(c.Param("mediaId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	media := content.AsMedia()
	media.Watched = domain.NewWatchedDtoForPublicContentPage(&watched, thoughtsPublic)
	if err := addedtocontent.AddList(
		r.watchedProvider,
		uint(userId),
		media.Similar,
		func(i int, w *entity.Watched) {
			media.Similar[i].Watched = domain.NewWatchedDtoForPublicLists(w)
		},
	); err != nil {
		slog.Error("GetPublicGameDetails: Failed to add public watched data to similar content", "error", err)
	}
	c.JSON(http.StatusOK, domain.PublicMediaDetailsResponse{
		Media:          media,
		ThoughtsPublic: thoughtsPublic,
	})
}

func (r *Router) GetGameDetails(c *gin.Context) {
	userId := c.MustGet("userId").(uint)
	if c.Param("id") == "" {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "an id was not provided"})
		return
	}
	content, err := r.br.Cfg.TWITCH.GameDetails(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	contentAsMedia := content.AsMedia()
	if err := addedtocontent.AddSingularAndList(
		r.watchedProvider,
		userId,
		contentAsMedia,
		func(w *entity.Watched) {
			contentAsMedia.Watched = domain.NewWatchedDtoForContentPage(w)
		},
		[]*addedtocontent.AddListCall[domain.Media]{
			addedtocontent.NewAddListCall(
				contentAsMedia.Similar,
				func(i int, w *entity.Watched) {
					contentAsMedia.Similar[i].Watched = domain.NewWatchedDtoForLists(w)
				},
			),
		},
	); err != nil {
		slog.Error("GetGameDetails: Failed to add watched to content!", "error", err)
		c.JSON(
			http.StatusInternalServerError,
			router.ErrorResponse{Error: "failed to add watched data to response"},
		)
		return
	}
	c.JSON(http.StatusOK, contentAsMedia)
}

func (r *Router) UpdateConfig(c *gin.Context) {
	var ar igdb.IGDB
	err := c.ShouldBindJSON(&ar)
	if err == nil {
		err := r.br.Cfg.SaveTwitchConfig(ar)
		if err != nil {
			c.JSON(http.StatusForbidden, router.ErrorResponse{Error: err.Error()})
			return
		}
		// gdb = &b.cfg.TWITCH
		c.Status(http.StatusOK)
		return
	}
	c.AbortWithStatusJSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
}
