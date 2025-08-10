package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

func SetupTransactionRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware) {

	transactionHandler := handler.GetTransactionHandler()

	transactions := e.Group("/v1/transactions")
	transactions.Use(authMiddleware.Authenticate)

	transactions.POST("", transactionHandler.CreateTransaction)
	transactions.GET("", transactionHandler.ListTransactions)
	transactions.GET("/:id", transactionHandler.GetTransaction)
	transactions.GET("/:id/status", transactionHandler.GetTransactionStatus) // Lightweight status endpoint
	transactions.POST("/:id/payment", transactionHandler.ProcessPayment) // Buyer initiates payment
	transactions.POST("/:id/confirm", transactionHandler.ConfirmDelivery)
	transactions.POST("/:id/dispute", transactionHandler.CreateDispute)
	transactions.POST("/:id/cancel", transactionHandler.CancelTransaction)
	transactions.GET("/:id/logs", transactionHandler.GetTransactionLogs)

	admin := e.Group("/v1/admin/transactions")
	admin.Use(authMiddleware.Authenticate)
	admin.Use(adminMiddleware.AdminOnly)

	admin.GET("", transactionHandler.ListAdminTransactions)
	admin.GET("/pending-middleman", transactionHandler.ListPendingMiddlemanTransactions)
	admin.POST("/:id/assign", transactionHandler.AssignMiddleman)
	admin.POST("/:id/confirm-payment", transactionHandler.ConfirmMiddlemanPayment) // New: Admin confirms middleman payment
	admin.POST("/:id/complete", transactionHandler.CompleteMiddleman)
	admin.POST("/:id/resolve-dispute", transactionHandler.ResolveDispute)
}
