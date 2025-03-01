package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

// SetupGameTitleRouter initializes game title routes
func SetupGameTitleRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware) {
    // Get handlers from DI
    gameTitleHandler := handler.GetGameTitleHandler()

    // Public routes
    e.GET("/v1/game-titles", gameTitleHandler.ListGameTitles)
    e.GET("/v1/game-titles/:id", gameTitleHandler.GetGameTitle)
    e.GET("/v1/games/:slug", gameTitleHandler.GetGameTitleBySlug)

    // Admin routes - dengan dua middleware
    admin := e.Group("/v1/admin/game-titles")
    admin.Use(authMiddleware.Authenticate) // Pertama, verifikasi token
    admin.Use(adminMiddleware.AdminOnly)   // Kedua, verifikasi role admin

    admin.POST("", gameTitleHandler.CreateGameTitle)
    admin.PUT("/:id", gameTitleHandler.UpdateGameTitle)
    admin.DELETE("/:id", gameTitleHandler.DeleteGameTitle)
}