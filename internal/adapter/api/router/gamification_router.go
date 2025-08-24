package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

func SetupGamificationRoutes(e *echo.Echo, h *handler.GamificationHandler, authMiddleware *middleware.AuthMiddleware) {
	// Public routes (no auth required)
	public := e.Group("/api/gamification")
	
	// Webhook for transaction events (called by payment service)
	public.POST("/webhook/transaction", h.TransactionWebhook)

	// Protected routes (require authentication)
	protected := e.Group("/api/gamification")
	protected.Use(authMiddleware.Authenticate)

	// User gamification status and data
	protected.GET("/status", h.GetUserStatus)
	
	// Event tracking
	protected.POST("/track-events", h.TrackEvents)
	protected.POST("/process-events", h.ProcessEvents)
	
	// Achievement management
	protected.POST("/unlock-achievement", h.UnlockAchievement)
	
	// Progress and statistics updates
	protected.POST("/update-progress", h.UpdateProgress)
	protected.POST("/update-stats", h.UpdateStatistics)
	
	// Leaderboard (public but might want to require auth later)
	protected.GET("/leaderboard", h.GetLeaderboard)

	// Admin routes (require admin auth)
	admin := e.Group("/api/admin/gamification")
	admin.Use(authMiddleware.Authenticate) // Should also check for admin role
	
	// Achievement management for admins
	admin.POST("/achievements", h.CreateAchievement)
}