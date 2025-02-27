package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

// SetupGameTitleRouter initializes game title routes
func SetupGameTitleRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware) {
	// Get handlers from DI
	gameTitleHandler := handler.GetGameTitleHandler()

	// Public routes
	e.GET("/v1/game-titles", gameTitleHandler.ListGameTitles)
	e.GET("/v1/game-titles/:id", gameTitleHandler.GetGameTitle)
	e.GET("/v1/games/:slug", gameTitleHandler.GetGameTitleBySlug)

	// Admin routes
	admin := e.Group("/v1/admin/game-titles")
	admin.Use(authMiddleware.Authenticate)
	// Ideally add admin check middleware here

	admin.POST("", gameTitleHandler.CreateGameTitle)
	admin.PUT("/:id", gameTitleHandler.UpdateGameTitle)
	admin.DELETE("/:id", gameTitleHandler.DeleteGameTitle)
}