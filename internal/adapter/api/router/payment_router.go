package router

import (
	"github.com/labstack/echo/v4"
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"
)

func SetupPaymentRoutes(e *echo.Echo, paymentHandler *handler.PaymentHandler, authMiddleware *middleware.AuthMiddleware) {
	// Payment routes group with rate limiting
	paymentGroup := e.Group("/v1/payments")

	// Protected routes (require authentication + rate limiting)
	paymentGroup.POST("/transactions", paymentHandler.CreateSecureTransaction, 
		middleware.PaymentRateLimit(), authMiddleware.Authenticate)
	paymentGroup.POST("/transactions/instant", paymentHandler.CreateInstantTransaction, 
		middleware.PaymentRateLimit(), authMiddleware.Authenticate) // Simplified for chat UI
	paymentGroup.GET("/transactions/:id/status", paymentHandler.GetPaymentStatus, 
		middleware.GeneralRateLimit(), authMiddleware.Authenticate)

	// Public webhook routes (rate limited but no auth - Midtrans calls these)
	paymentGroup.POST("/midtrans/callback", paymentHandler.MidtransCallback, 
		middleware.WebhookRateLimit())
	paymentGroup.POST("/midtrans/notification", paymentHandler.MidtransCallback, 
		middleware.WebhookRateLimit()) // Alternative endpoint
}