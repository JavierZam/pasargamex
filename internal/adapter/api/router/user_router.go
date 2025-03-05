package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

// SetupUserRouter initializes user routes
func SetupUserRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware) {
	// Get handlers from DI
	userHandler := handler.GetUserHandler()

	// All user routes are protected
	users := e.Group("/v1/users")
	users.Use(authMiddleware.Authenticate)

	users.GET("/me", userHandler.GetProfile)
	users.PATCH("/me", userHandler.UpdateProfile)
	users.PUT("/me/password", userHandler.UpdatePassword)

	users.POST("/me/verification", userHandler.SubmitVerification)

	// Admin routes for verification
	admin := e.Group("/v1/admin/users")
    admin.Use(authMiddleware.Authenticate) // Pertama, verifikasi token
    admin.Use(adminMiddleware.AdminOnly)   // Kedua, verifikasi role admin
	
	// Add admin verification processing endpoint
	admin.POST("/:userId/verification", userHandler.ProcessVerification)
}