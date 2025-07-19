package router

import (
	"github.com/labstack/echo/v4"

	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"
)

type WalletRouter struct {
	walletHandler *handler.WalletHandler
}

func NewWalletRouter(walletHandler *handler.WalletHandler) *WalletRouter {
	return &WalletRouter{
		walletHandler: walletHandler,
	}
}

func (r *WalletRouter) SetupRoutes(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware) {
	// User wallet routes
	walletGroup := e.Group("/v1/wallet")
	walletGroup.Use(authMiddleware.Authenticate)

	// Wallet management
	walletGroup.POST("/create", r.walletHandler.CreateWallet)
	walletGroup.GET("", r.walletHandler.GetWallet)
	walletGroup.GET("/transactions", r.walletHandler.GetWalletTransactions)

	// Payment methods
	paymentMethodGroup := walletGroup.Group("/payment-methods")
	paymentMethodGroup.POST("", r.walletHandler.CreatePaymentMethod)
	paymentMethodGroup.GET("", r.walletHandler.GetPaymentMethods)
	paymentMethodGroup.PUT("/:id", r.walletHandler.UpdatePaymentMethod)
	paymentMethodGroup.DELETE("/:id", r.walletHandler.DeletePaymentMethod)

	// Topup requests
	topupGroup := walletGroup.Group("/topup")
	topupGroup.POST("", r.walletHandler.CreateTopupRequest)
	topupGroup.GET("", r.walletHandler.GetTopupRequests)

	// Withdraw requests
	withdrawGroup := walletGroup.Group("/withdraw")
	withdrawGroup.POST("", r.walletHandler.CreateWithdrawRequest)
	withdrawGroup.GET("", r.walletHandler.GetWithdrawRequests)

	// Admin routes
	adminGroup := e.Group("/v1/admin/wallet")
	adminGroup.Use(authMiddleware.Authenticate)
	adminGroup.Use(adminMiddleware.AdminOnly)

	// Admin topup management
	adminTopupGroup := adminGroup.Group("/topup")
	adminTopupGroup.POST("/:id/process", r.walletHandler.ProcessTopupRequest)

	// Admin withdraw management
	adminWithdrawGroup := adminGroup.Group("/withdraw")
	adminWithdrawGroup.POST("/:id/process", r.walletHandler.ProcessWithdrawRequest)

	// Admin monitoring endpoints
	adminGroup.GET("/pending-topups", r.walletHandler.GetPendingTopupRequests)
	adminGroup.GET("/pending-withdrawals", r.walletHandler.GetPendingWithdrawRequests)
	adminGroup.GET("/statistics", r.walletHandler.GetWalletStatistics)
}