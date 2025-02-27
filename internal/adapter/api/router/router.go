package router

import (
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

// Setup initializes all routers
func Setup(e *echo.Echo, authMiddleware *middleware.AuthMiddleware) {
	// Initialize all routers
	SetupAuthRouter(e, authMiddleware)
	SetupUserRouter(e, authMiddleware)
	SetupGameTitleRouter(e, authMiddleware)
	// Add other routers as they are implemented
}