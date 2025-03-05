package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

func SetupReviewRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware) {
	reviewHandler := handler.GetReviewHandler()
	
	// Public routes
	reviews := e.Group("/v1/reviews")
	reviews.GET("", reviewHandler.GetReviews)
	
	// Protected routes (require authentication)
	authenticated := e.Group("")
	authenticated.Use(authMiddleware.Authenticate)
	
	authenticated.POST("/v1/transactions/:transactionId/review", reviewHandler.CreateReview)
	authenticated.POST("/v1/reviews/:reviewId/report", reviewHandler.ReportReview)
	
	// Admin routes
	admin := e.Group("/v1/admin/reviews")
	admin.Use(authMiddleware.Authenticate)
	admin.Use(adminMiddleware.AdminOnly)
	
	admin.GET("/reports", reviewHandler.GetReviewReports)
	admin.PATCH("/:reviewId/status", reviewHandler.UpdateReviewStatus)
}