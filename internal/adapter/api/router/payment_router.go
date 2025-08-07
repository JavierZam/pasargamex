package router

import (
	"github.com/labstack/echo/v4"
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"
)

func SetupPaymentRoutes(e *echo.Echo, paymentHandler *handler.PaymentHandler, authMiddleware *middleware.AuthMiddleware) {
	// Payment routes group
	paymentGroup := e.Group("/v1/payments")

	// Protected routes (require authentication)
	paymentGroup.POST("/transactions", paymentHandler.CreateSecureTransaction, authMiddleware.Authenticate)
	paymentGroup.POST("/transactions/instant", paymentHandler.CreateInstantTransaction, authMiddleware.Authenticate) // Simplified for chat UI
	paymentGroup.GET("/transactions/:id/status", paymentHandler.GetPaymentStatus, authMiddleware.Authenticate)

	// Public webhook routes (no authentication required - Midtrans calls these)
	paymentGroup.POST("/midtrans/callback", paymentHandler.MidtransCallback)
	paymentGroup.POST("/midtrans/notification", paymentHandler.MidtransCallback) // Alternative endpoint
}