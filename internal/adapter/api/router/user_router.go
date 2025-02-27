package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

// SetupUserRouter initializes user routes
func SetupUserRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware) {
	// Get handlers from DI
	userHandler := handler.GetUserHandler()

	// All user routes are protected
	users := e.Group("/v1/users")
	users.Use(authMiddleware.Authenticate)

	users.GET("/me", userHandler.GetProfile)
	users.PATCH("/me", userHandler.UpdateProfile)
	users.PUT("/me/password", userHandler.UpdatePassword)
}