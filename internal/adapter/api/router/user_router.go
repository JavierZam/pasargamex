package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

func SetupUserRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware) {

	userHandler := handler.GetUserHandler()

	// Public seller profiles (no auth required)
	e.GET("/v1/sellers/:id/profile", userHandler.GetPublicSellerProfile)

	users := e.Group("/v1/users")
	users.Use(authMiddleware.Authenticate)

	users.GET("/me", userHandler.GetProfile)
	users.PATCH("/me", userHandler.UpdateProfile)
	users.PUT("/me/password", userHandler.UpdatePassword)
	users.GET("/:id", userHandler.GetUserByID) // New: Get user profile by ID

	users.POST("/me/verification", userHandler.SubmitVerification)

	admin := e.Group("/v1/admin/users")
	admin.Use(authMiddleware.Authenticate)
	admin.Use(adminMiddleware.AdminOnly)

	admin.POST("/:userId/verification", userHandler.ProcessVerification)
	admin.PATCH("/:userId/role", userHandler.UpdateUserRole) // New: Update user role
}
