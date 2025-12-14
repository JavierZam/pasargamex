package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

func SetupAuthRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware) {

	authHandler := handler.GetAuthHandler()

	e.POST("/v1/auth/register", authHandler.Register)
	e.POST("/v1/auth/login", authHandler.Login)
	e.POST("/v1/auth/oauth", authHandler.OAuthLogin)
	e.POST("/v1/auth/refresh", authHandler.RefreshToken)

	protected := e.Group("/v1/auth")
	protected.Use(authMiddleware.Authenticate)

	protected.POST("/logout", authHandler.Logout)
}
