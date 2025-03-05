package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

// SetupTransactionRouter initializes transaction routes
func SetupTransactionRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware) {
	// Get handlers from DI
	transactionHandler := handler.GetTransactionHandler()

	// All transaction routes require authentication
	transactions := e.Group("/v1/transactions")
	transactions.Use(authMiddleware.Authenticate)

	// User endpoints
	transactions.POST("", transactionHandler.CreateTransaction)
	transactions.GET("", transactionHandler.ListTransactions)
	transactions.GET("/:id", transactionHandler.GetTransaction)
	transactions.POST("/:id/payment", transactionHandler.ProcessPayment)
	transactions.POST("/:id/confirm", transactionHandler.ConfirmDelivery)
	transactions.POST("/:id/dispute", transactionHandler.CreateDispute)
	transactions.POST("/:id/cancel", transactionHandler.CancelTransaction)
	transactions.GET("/:id/logs", transactionHandler.GetTransactionLogs)

	// Admin endpoints
	admin := e.Group("/v1/admin/transactions")
	admin.Use(authMiddleware.Authenticate)
	admin.Use(adminMiddleware.AdminOnly)

	admin.GET("", transactionHandler.ListAdminTransactions)
	admin.GET("/pending-middleman", transactionHandler.ListPendingMiddlemanTransactions)
	admin.POST("/:id/assign", transactionHandler.AssignMiddleman)
	admin.POST("/:id/complete", transactionHandler.CompleteMiddleman)
	admin.POST("/:id/resolve-dispute", transactionHandler.ResolveDispute)
}