package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

func SetupGameTitleRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware) {

	gameTitleHandler := handler.GetGameTitleHandler()

	e.GET("/v1/game-titles", gameTitleHandler.ListGameTitles)
	e.GET("/v1/game-titles/:id", gameTitleHandler.GetGameTitle)
	e.GET("/v1/games/:slug", gameTitleHandler.GetGameTitleBySlug)

	admin := e.Group("/v1/admin/game-titles")
	admin.Use(authMiddleware.Authenticate)
	admin.Use(adminMiddleware.AdminOnly)

	admin.POST("", gameTitleHandler.CreateGameTitle)
	admin.PUT("/:id", gameTitleHandler.UpdateGameTitle)
	admin.DELETE("/:id", gameTitleHandler.DeleteGameTitle)
}
