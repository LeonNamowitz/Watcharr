package main

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sbondCo/Watcharr/config"
	"github.com/sbondCo/Watcharr/feature/content"
	"github.com/sbondCo/Watcharr/feature/game"
	"github.com/sbondCo/Watcharr/feature/search"
	"github.com/sbondCo/Watcharr/feature/user"
	"github.com/sbondCo/Watcharr/feature/watched"
	appRouter "github.com/sbondCo/Watcharr/router"
)

func TestPublicRouteRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/api")
	br := appRouter.NewBaseRouter(
		nil,
		api,
		&config.ServerConfig{JWT_SECRET: "test"},
	)

	// Gin panics during registration when related wildcard routes conflict.
	// Register these together so the public API namespace stays compatible as
	// routes are added across feature packages.
	watched.NewRouter(br, nil).AddRoutes()
	search.NewRouter(br, nil, nil).AddRoutes()
	user.NewRouter(br, nil, nil).AddRoutes()
	content.NewRouter(br, nil, nil, nil).AddRoutes()
	game.NewRouter(br, nil, nil).AddRoutes()
}
