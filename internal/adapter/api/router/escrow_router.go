package router

import (
	"github.com/labstack/echo/v4"
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"
)

func SetupEscrowRoutes(e *echo.Echo, escrowHandler *handler.EscrowHandler, authMiddleware *middleware.AuthMiddleware) {
	// Escrow routes group
	escrowGroup := e.Group("/v1/escrow")

	// Protected routes (require authentication)
	escrowGroup.POST("/deliver-credentials", escrowHandler.DeliverCredentials, authMiddleware.Authenticate)
	escrowGroup.POST("/confirm-credentials", escrowHandler.ConfirmCredentials, authMiddleware.Authenticate)
	escrowGroup.GET("/transactions/:id/credentials", escrowHandler.GetTransactionCredentials, authMiddleware.Authenticate)
}