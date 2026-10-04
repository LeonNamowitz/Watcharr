package stats

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/database/entity"
	"github.com/sbondCo/Watcharr/feature/auth/authmiddleware"
	"github.com/sbondCo/Watcharr/router"
)

type PublicOwnerValidator interface{ ValidatePublicWatchedList(uint, string) error }
type Router struct {
	br        *router.BaseRouter
	service   *Service
	validator PublicOwnerValidator
}

func NewRouter(br *router.BaseRouter, service *Service, validator PublicOwnerValidator) *Router {
	return &Router{br, service, validator}
}
func (r *Router) AddRoutes() {
	private := r.br.Router.Group("/stats").Use(authmiddleware.AuthRequired(nil, r.br.Cfg))
	private.GET("", r.GetPrivateStats)
	private.PUT("/layout", r.UpdateLayout)
	r.br.Router.GET("/public/users/:id/:username/stats", r.GetPublicStats)
}

func (r *Router) UpdateLayout(c *gin.Context) {
	var input struct {
		Media        string   `json:"media" binding:"required,oneof=movie tv game"`
		SectionOrder []string `json:"sectionOrder" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "invalid stats layout"})
		return
	}
	order, err := validateSectionOrder(input.Media, input.SectionOrder)
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	if err := r.service.saveSectionOrder(c.MustGet("userId").(uint), input.Media, order); err != nil {
		c.JSON(http.StatusInternalServerError, router.ErrorResponse{Error: "unable to save stats layout"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sectionOrder": order})
}
func (r *Router) GetPrivateStats(c *gin.Context) { r.respond(c, c.MustGet("userId").(uint), false) }
func (r *Router) GetPublicStats(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: "invalid stats owner"})
		return
	}
	if err = r.validator.ValidatePublicWatchedList(uint(id), c.Param("username")); err != nil {
		c.JSON(http.StatusForbidden, router.ErrorResponse{Error: err.Error()})
		return
	}
	var owner entity.User
	if err = r.br.DB.First(&owner, uint(id)).Error; err != nil {
		c.JSON(http.StatusForbidden, router.ErrorResponse{Error: "failed to load stats owner"})
		return
	}
	r.respond(c, uint(id), owner.PrivateThoughts != nil && *owner.PrivateThoughts)
}
func (r *Router) respond(c *gin.Context, id uint, hideReviews bool) {
	q, err := queryFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	q.HideReviews = hideReviews
	data, err := r.service.GetStats(id, q)
	if err != nil {
		c.JSON(http.StatusBadRequest, router.ErrorResponse{Error: err.Error()})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, data)
}
func queryFromContext(c *gin.Context) (Query, error) {
	q := Query{Scope: ScopeYear, Year: time.Now().UTC().Year(), Media: c.DefaultQuery("media", "movie")}
	if c.Query("year") == "all" {
		q.Scope = ScopeLifetime
		q.Year = 0
	} else if value := c.Query("year"); value != "" {
		y, err := strconv.Atoi(value)
		if err != nil || y < 1 || y > 9999 {
			return Query{}, &queryError{"invalid stats year"}
		}
		q.Year = y
	}
	if q.Media != "movie" && q.Media != "tv" && q.Media != "game" {
		return Query{}, &queryError{"invalid stats media"}
	}
	return q, nil
}

type queryError struct{ message string }

func (e *queryError) Error() string { return e.message }
