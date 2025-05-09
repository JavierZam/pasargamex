package router

import (
	"pasargamex/internal/adapter/api/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func Setup(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware, authClient *auth.Client) {
    SetupAuthRouter(e, authMiddleware)
    SetupUserRouter(e, authMiddleware, adminMiddleware)
    SetupGameTitleRouter(e, authMiddleware, adminMiddleware)
    SetupProductRouter(e, authMiddleware, adminMiddleware, authClient)
    SetupTransactionRouter(e, authMiddleware, adminMiddleware)
    SetupHealthRouter(e)
    SetupReviewRouter(e, authMiddleware, adminMiddleware)
}