package content

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cache"
	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/domain"
	"github.com/sbondCo/Watcharr/feature/auth/authmiddleware"
	"github.com/sbondCo/Watcharr/feature/watched/addedtocontent"
	"github.com/sbondCo/Watcharr/media/tmdb"
	"github.com/sbondCo/Watcharr/router"
	"github.com/sbondCo/Watcharr/util"
)

type WatchedProvider interface {
	UpdateWatchedLastViewedSeason(userId uint, id uint, seasonNum int) error
	GetWatchedItemBySupportedMediaId(userId uint, id uint, t util.SupportedMedia) (entity.Watched, error)
	GetWatchedItemsBySupportedMediaIds(userId uint, c []addedtocontent.IdToTypePair) ([]entity.Watched, error)
	GetPublicWatchedItem(userId uint, username string, mediaId uint, mediaType util.SupportedMedia) (entity.Watched, bool, error)
	GetOptionalPublicWatchedItem(userId uint, username string, mediaId uint, mediaType util.SupportedMedia) (*entity.Watched, bool, error)
	ValidatePublicWatchedList(userId uint, username string) error
}

type TMDBProvider interface {
	MovieDetails(tmdb.MovieDetailsOptions) (tmdb.MovieDetails, error)
	MovieCredits(string) (tmdb.ContentCredits, error)
	ShowDetails(tmdb.ShowDetailsOptions) (tmdb.ShowDetails, error)
	ShowCredits(string) (tmdb.ContentCredits, error)
	SeasonDetails(string, string) (tmdb.SeasonDetails, error)
	PersonDetails(string) (tmdb.PersonDetails, error)
	PersonCredits(string) (tmdb.PersonCombinedCredits, error)
	Regions() (tmdb.Regions, error)
}

type Router struct {
	br   *router.BaseRouter
	cs   *Service
	wp   WatchedProvider
	tmdb TMDBProvider
}

func NewRouter(
	br *router.BaseRouter,
	cs *Service,
	wp WatchedProvider,
	tmdb TMDBProvider,
) *Router {
	return &Router{
		br:   br,
		cs:   cs,
		wp:   wp,
		tmdb: tmdb,
	}
}

func (r *Router) AddRoutes() {
	content := r.br.Router.Group("/content").Use(authmiddleware.AuthRequired(nil, r.br.Cfg))
	publicOwnerContent := r.br.Router.Group("/public/users/:id/:username/content").Use(r.PublicOwnerRequired)
	exp := time.Hour * 24

	// NOTE: Some routes use `cache.CachePage`, but others that contain user watched data
	// don't and rather have their caching on the TMDB methods directly.

	// Get movie details (for movie page)
	content.GET("/movie/:id", router.WhereaboutsRequired(r.br.Cfg), r.GetMovieDetails)
	// Get movie cast
	content.GET("/movie/:id/credits", cache.CachePage(r.br.MemStore, exp, r.GetMovieCredits))
	// Get tv details (for tv page)
	content.GET("/tv/:id", router.WhereaboutsRequired(r.br.Cfg), r.GetTvDetails)
	// Get tv cast
	content.GET("/tv/:id/credits", cache.CachePage(r.br.MemStore, exp, r.GetTvCredits))

	publicOwnerContent.GET("/movie/:mediaId", router.WhereaboutsRequired(r.br.Cfg), r.GetPublicMovieDetails)
	publicOwnerContent.GET("/movie/:mediaId/credits", cache.CachePage(r.br.MemStore, exp, r.GetPublicMovieCredits))
	publicOwnerContent.GET("/tv/:mediaId", router.WhereaboutsRequired(r.br.Cfg), r.GetPublicTvDetails)
	publicOwnerContent.GET("/tv/:mediaId/credits", cache.CachePage(r.br.MemStore, exp, r.GetPublicTvCredits))
	publicOwnerContent.GET("/tv/:mediaId/season/:num", r.GetPublicTvSeasonDetails)
	publicOwnerContent.GET("/person/:personId", r.GetPublicPerson)
	publicOwnerContent.GET("/person/:personId/credits", r.GetPublicPersonCredits)
	// Get season details
	// Supports `watchedId` query parameter for saving the requested season as `LastViewedSeason`.
	content.GET("/tv/:id/season/:num", r.GetSeasonDetails)
	// Get person details
	content.GET("/person/:id", cache.CachePage(r.br.MemStore, exp, r.GetPerson))
	// Get person credits
	content.GET("/person/:id/credits", r.GetPersonCredits)
	// Available regions for watch providers
	content.GET("/regions", r.GetRegions)
}

func (r *Router) PublicOwnerRequired(c *gin.Context) {
	userId, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			router.ErrorResponse{Error: "invalid public list owner id"},
		)
		return
	}
	if err := r.wp.ValidatePublicWatchedList(uint(userId), c.Param("username")); err != nil {
		c.AbortWithStatusJSON(
			http.StatusForbidden,
			router.ErrorResponse{Error: "failed validating the public list owner"},
		)
		return
	}
	c.Set("publicOwnerId", uint(userId))
	c.Next()
}

func (r *Router) getPublicOwnerID(c *gin.Context) (uint, error) {
	if userId, exists := c.Get("publicOwnerId"); exists {
		return userId.(uint), nil
	}
	userId, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	if err := r.wp.ValidatePublicWatchedList(uint(userId), c.Param("username")); err != nil {
		return 0, err
	}
	return uint(userId), nil
}

func (r *Router) getOptionalPublicOwnerWatched(c *gin.Context, mediaType util.SupportedMedia) (uint, *entity.Watched, bool, error) {
	userId, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0, nil, false, err
	}
	mediaId, err := strconv.ParseUint(c.Param("mediaId"), 10, 64)
	if err != nil {
		return 0, nil, false, err
	}
	watched, thoughtsPublic, err := r.wp.GetOptionalPublicWatchedItem(
		uint(userId),
		c.Param("username"),
		uint(mediaId),
		mediaType,
	)
	return uint(userId), watched, thoughtsPublic, err
}

func (r *Router) addPublicSimilarWatched(userId uint, media *domain.Media) error {
	return addedtocontent.AddList(
		r.wp,
		userId,
		media.Similar,
		func(i int, w *entity.Watched) {
			media.Similar[i].Watched = domain.NewWatchedDtoForPublicLists(w)
		},
	)
}

func (r *Router) GetPublicMovieCredits(c *gin.Context) {
	if _, err := r.getPublicOwnerID(c); err != nil {
		c.JSON(http.StatusForbidden, router.ErrorResponse{Error: "failed fetching the public list item"})
		return
	}
	content, err := r.tmdb.MovieCredits(c.Param("mediaId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, content)
}

func (r *Router) GetPublicTvCredits(c *gin.Context) {
	if _, err := r.getPublicOwnerID(c); err != nil {
		c.JSON(http.StatusForbidden, router.ErrorResponse{Error: "failed fetching the public list item"})
		return
	}
	content, err := r.tmdb.ShowCredits(c.Param("mediaId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, content)
}

// GetPublicTvSeasonDetails returns the normal read-only TMDB season and episode
// overview after validating the requested public list owner.
func (r *Router) GetPublicTvSeasonDetails(c *gin.Context) {
	if c.Param("num") == "" {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "a season number was not provided"})
		return
	}
	if _, err := r.getPublicOwnerID(c); err != nil {
		c.JSON(http.StatusForbidden, router.ErrorResponse{Error: "failed fetching the public list item"})
		return
	}
	content, err := r.tmdb.SeasonDetails(c.Param("mediaId"), c.Param("num"))
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, content)
}

// GetPublicMovieDetails returns the normal movie page data with the public
// list owner's watched entry attached instead of the visitor's.
func (r *Router) GetPublicMovieDetails(c *gin.Context) {
	userId, watched, thoughtsPublic, err := r.getOptionalPublicOwnerWatched(
		c,
		util.SupportedMediaMovie,
	)
	if err != nil {
		c.JSON(http.StatusForbidden, router.ErrorResponse{Error: "failed fetching the public list item"})
		return
	}
	content, err := r.tmdb.MovieDetails(tmdb.MovieDetailsOptions{
		ID:      c.Param("mediaId"),
		Country: c.MustGet("userCountry").(string),
		Params: map[string]string{
			"append_to_response": "videos,watch/providers,similar",
		},
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	media := content.AsMedia()
	if watched != nil {
		media.Watched = domain.NewWatchedDtoForPublicContentPage(watched, thoughtsPublic)
	} else {
		thoughtsPublic = false
	}
	if err := r.addPublicSimilarWatched(userId, &media); err != nil {
		slog.Error("GetPublicMovieDetails: Failed to add public watched data to similar content", "error", err)
	}
	c.JSON(http.StatusOK, domain.PublicMediaDetailsResponse{
		Media:          media,
		ThoughtsPublic: thoughtsPublic,
	})
}

// GetPublicTvDetails returns the normal show page data with the public list
// owner's watched entry attached instead of the visitor's.
func (r *Router) GetPublicTvDetails(c *gin.Context) {
	userId, watched, thoughtsPublic, err := r.getOptionalPublicOwnerWatched(
		c,
		util.SupportedMediaShow,
	)
	if err != nil {
		c.JSON(http.StatusForbidden, router.ErrorResponse{Error: "failed fetching the public list item"})
		return
	}
	content, err := r.tmdb.ShowDetails(tmdb.ShowDetailsOptions{
		ID:      c.Param("mediaId"),
		Country: c.MustGet("userCountry").(string),
		Params: map[string]string{
			"append_to_response": "videos,watch/providers,similar,external_ids,keywords",
		},
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	media := content.AsMedia()
	if watched != nil {
		media.Watched = domain.NewWatchedDtoForPublicContentPage(watched, thoughtsPublic)
	} else {
		thoughtsPublic = false
	}
	if err := r.addPublicSimilarWatched(userId, &media); err != nil {
		slog.Error("GetPublicTvDetails: Failed to add public watched data to similar content", "error", err)
	}
	c.JSON(http.StatusOK, domain.PublicMediaDetailsResponse{
		Media:          media,
		ThoughtsPublic: thoughtsPublic,
	})
}

func (r *Router) GetMovieDetails(c *gin.Context) {
	userId := c.MustGet("userId").(uint)
	if c.Param("id") == "" {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "an id was not provided"})
		return
	}
	content, err := r.tmdb.MovieDetails(tmdb.MovieDetailsOptions{
		ID:      c.Param("id"),
		Country: c.MustGet("userCountry").(string),
		Params: map[string]string{
			"append_to_response": "videos,watch/providers,similar",
		},
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	contentAsMedia := content.AsMedia()
	if err := addedtocontent.AddSingularAndList(
		r.wp,
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
		slog.Error("GetMovieDetails: Failed to add watched to content!", "error", err)
		c.JSON(
			http.StatusInternalServerError,
			router.ErrorResponse{Error: "failed to add watched data to response"},
		)
		return
	}
	c.JSON(http.StatusOK, contentAsMedia)
}

func (r *Router) GetMovieCredits(c *gin.Context) {
	if c.Param("id") == "" {
		c.Status(400)
		return
	}
	content, err := r.tmdb.MovieCredits(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, content)
}

func (r *Router) GetTvDetails(c *gin.Context) {
	userId := c.MustGet("userId").(uint)
	if c.Param("id") == "" {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "an id was not provided"})
		return
	}
	// 1. Get details
	content, err := r.tmdb.ShowDetails(tmdb.ShowDetailsOptions{
		ID:      c.Param("id"),
		Country: c.MustGet("userCountry").(string),
		Params: map[string]string{
			"append_to_response": "videos,watch/providers,similar,external_ids,keywords",
		},
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	contentAsMedia := content.AsMedia()
	if err := addedtocontent.AddSingularAndList(
		r.wp,
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
		slog.Error("GetTvDetails: Failed to add watched to content!", "error", err)
		c.JSON(
			http.StatusInternalServerError,
			router.ErrorResponse{Error: "failed to add watched data to response"},
		)
		return
	}
	c.JSON(http.StatusOK, contentAsMedia)
}

func (r *Router) GetTvCredits(c *gin.Context) {
	if c.Param("id") == "" {
		c.Status(400)
		return
	}
	content, err := r.tmdb.ShowCredits(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, content)
}

// Get season details
// Supports `watchedId` query parameter for saving the requested season as `LastViewedSeason`.
func (r *Router) GetSeasonDetails(c *gin.Context) {
	if c.Param("id") == "" || c.Param("num") == "" {
		c.Status(400)
		return
	}
	content, err := r.tmdb.SeasonDetails(c.Param("id"), c.Param("num"))
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	// If a `watchedId` is passed, we should update it with this season
	// number, so the LastViewedSeason field is up to date (this seemed
	// better than making a new request for just saving this).
	// We will attach a `watcharr-lastviewedseason-saved` header if
	// this part succeeds so the client can decide on showing an error.
	if watchedIdQ := c.Query("watchedId"); watchedIdQ != "" {
		userId := c.MustGet("userId").(uint)
		watchedId, err := strconv.ParseUint(watchedIdQ, 10, 64)
		if err != nil {
			slog.Error("get season details route: Processing watchedId param failed", "error", err.Error(), "id", watchedIdQ)
		} else {
			if seasonNum, err := strconv.ParseInt(c.Param("num"), 10, 64); err == nil {
				if err = r.wp.UpdateWatchedLastViewedSeason(
					userId, uint(watchedId), int(seasonNum),
				); err == nil {
					c.Header("watcharr-lastviewedseason-saved", "1")
				}
			} else {
				slog.Error("get season details route: Parsing season number as int failed", "error", err.Error(), "season_num", c.Param("num"))
			}
		}
	} else {
		slog.Debug("get season details route: No watchedId parameter found.. not doing anything.")
	}
	c.JSON(http.StatusOK, content)
}

func (r *Router) GetPerson(c *gin.Context) {
	if c.Param("id") == "" {
		c.Status(400)
		return
	}
	content, err := r.tmdb.PersonDetails(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, content.AsPersonDetailsResponse())
}

func (r *Router) GetPublicPerson(c *gin.Context) {
	if _, err := r.getPublicOwnerID(c); err != nil {
		c.JSON(http.StatusForbidden, router.ErrorResponse{Error: "failed validating the public list owner"})
		return
	}
	if c.Param("personId") == "" {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "a person id was not provided"})
		return
	}
	if _, err := strconv.ParseUint(c.Param("personId"), 10, 64); err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "invalid person id"})
		return
	}
	content, err := r.tmdb.PersonDetails(c.Param("personId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, content.AsPersonDetailsResponse())
}

func (r *Router) getPersonCreditsResponse(
	personId string,
	creditsType string,
) (domain.PersonCreditsResponse, error) {
	creditsType = strings.ToLower(creditsType)
	if creditsType == "" {
		person, err := r.tmdb.PersonDetails(personId)
		if err != nil {
			return domain.PersonCreditsResponse{}, err
		}
		creditsType = strings.ToLower(person.KnownForDepartment)
	}
	content, err := r.tmdb.PersonCredits(personId)
	if err != nil {
		return domain.PersonCreditsResponse{}, err
	}

	resp := domain.PersonCreditsResponse{}
	resp.HasActing = len(content.Cast) > 0
	for i := range content.Crew {
		if strings.EqualFold(content.Crew[i].Department, "Directing") {
			resp.HasDirecting = true
			break
		}
	}
	switch creditsType {
	case "directing":
		for i := range content.Crew {
			if strings.EqualFold(content.Crew[i].Department, "Directing") {
				resp.Credits = append(resp.Credits, content.Crew[i].AsMedia())
			}
		}
	default:
		for i := range content.Cast {
			resp.Credits = append(resp.Credits, content.Cast[i].AsMedia())
		}
	}
	return resp, nil
}

func (r *Router) GetPersonCredits(c *gin.Context) {
	userId := c.MustGet("userId").(uint)
	if c.Param("id") == "" {
		c.Status(400)
		return
	}
	resp, err := r.getPersonCreditsResponse(c.Param("id"), c.Query("creditsType"))
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	if err := addedtocontent.AddList(
		r.wp,
		userId,
		resp.Credits,
		func(i int, w *entity.Watched) {
			resp.Credits[i].Watched = domain.NewWatchedDtoForLists(w)
		},
	); err != nil {
		slog.Error("GetPersonCredits: Failed to add watched to content!", "error", err)
		c.JSON(
			http.StatusInternalServerError,
			router.ErrorResponse{Error: "failed to add watched data to response"},
		)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (r *Router) GetPublicPersonCredits(c *gin.Context) {
	userId, err := r.getPublicOwnerID(c)
	if err != nil {
		c.JSON(http.StatusForbidden, router.ErrorResponse{Error: "failed validating the public list owner"})
		return
	}
	if c.Param("personId") == "" {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "a person id was not provided"})
		return
	}
	if _, err := strconv.ParseUint(c.Param("personId"), 10, 64); err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "invalid person id"})
		return
	}
	creditsType := strings.ToLower(c.Query("creditsType"))
	if creditsType != "" && creditsType != "acting" && creditsType != "directing" {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "invalid credits type"})
		return
	}
	resp, err := r.getPersonCreditsResponse(
		c.Param("personId"),
		creditsType,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	if err := addedtocontent.AddList(
		r.wp,
		userId,
		resp.Credits,
		func(i int, w *entity.Watched) {
			resp.Credits[i].Watched = domain.NewWatchedDtoForPublicLists(w)
		},
	); err != nil {
		slog.Error("GetPublicPersonCredits: Failed to add watched data", "error", err)
		c.JSON(
			http.StatusInternalServerError,
			router.ErrorResponse{Error: "failed to add watched data to response"},
		)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (r *Router) GetRegions(c *gin.Context) {
	re, err := r.tmdb.Regions()
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, re)
}
