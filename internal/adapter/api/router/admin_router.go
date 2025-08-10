package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

func SetupAdminRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware) {
	
	// Simple admin handler for basic endpoints (without dependencies for now)
	// TODO: Inject proper use cases when needed
	adminHandler := &handler.AdminHandler{}

	// Admin routes - require authentication and admin role
	admin := e.Group("/v1/admin")
	admin.Use(authMiddleware.Authenticate)
	admin.Use(adminMiddleware.AdminOnly)

	// Dashboard and stats
	admin.GET("/stats", adminHandler.GetDashboardStats)
	admin.GET("/health", adminHandler.GetSystemHealth)

	// User management
	admin.GET("/users", adminHandler.ListUsers)
	admin.GET("/users/verifications", adminHandler.GetUserVerifications)
	admin.PATCH("/users/:id/verify", adminHandler.VerifyUser)

	// Transaction management (already exists in transaction router)
	// admin.GET("/transactions", handled by transaction router)
	// admin.GET("/transactions/stats", handled by transaction router)
	admin.GET("/transactions/stats", adminHandler.GetTransactionStats)
}