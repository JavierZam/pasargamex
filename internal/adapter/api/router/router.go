package router

import (
	"pasargamex/internal/adapter/api/middleware"
	"pasargamex/internal/adapter/api/handler"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func Setup(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware, authClient *auth.Client, paymentHandler *handler.PaymentHandler) {
	SetupAuthRouter(e, authMiddleware)
	SetupUserRouter(e, authMiddleware, adminMiddleware)
	SetupGameTitleRouter(e, authMiddleware, adminMiddleware)
	SetupProductRouter(e, authMiddleware, adminMiddleware, authClient)
	SetupTransactionRouter(e, authMiddleware, adminMiddleware)
	SetupPaymentRoutes(e, paymentHandler, authMiddleware)
	SetupHealthRouter(e)
	SetupReviewRouter(e, authMiddleware, adminMiddleware)
	SetupFileRouter(e, authMiddleware, adminMiddleware)
	SetupWalletRouter(e, authMiddleware, adminMiddleware)
	SetupAdminRouter(e, authMiddleware, adminMiddleware) // New admin routes
}

func SetupWalletRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware) {
	walletHandler := handler.GetWalletHandler()
	walletRouter := NewWalletRouter(walletHandler)
	walletRouter.SetupRoutes(e, authMiddleware, adminMiddleware)
}
