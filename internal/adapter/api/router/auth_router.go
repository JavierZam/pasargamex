package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

// SetupAuthRouter initializes auth routes
func SetupAuthRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware) {
	// Get handlers from DI (will be passed from main)
	authHandler := handler.GetAuthHandler()

	// Public routes
	e.POST("/v1/auth/register", authHandler.Register)
	e.POST("/v1/auth/login", authHandler.Login)
	e.POST("/v1/auth/refresh", authHandler.RefreshToken)

	// Protected routes
	protected := e.Group("/v1/auth")
	protected.Use(authMiddleware.Authenticate)

	protected.POST("/logout", authHandler.Logout)
}