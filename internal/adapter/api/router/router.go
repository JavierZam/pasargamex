package router

import (
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

func Setup(e *echo.Echo, authMiddleware *middleware.AuthMiddleware) {
	SetupAuthRouter(e, authMiddleware)
	SetupUserRouter(e, authMiddleware)
	SetupGameTitleRouter(e, authMiddleware)
	SetupProductRouter(e, authMiddleware)
	SetupHealthRouter(e) 
}