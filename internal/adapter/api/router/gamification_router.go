package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/gorilla/mux"
)

func SetupGamificationRoutes(r *mux.Router, h *handler.GamificationHandler, authMiddleware *middleware.AuthMiddleware) {
	// Public routes (no auth required)
	public := r.PathPrefix("/api/gamification").Subrouter()
	
	// Webhook for transaction events (called by payment service)
	public.HandleFunc("/webhook/transaction", h.TransactionWebhook).Methods("POST")

	// Protected routes (require authentication)
	protected := r.PathPrefix("/api/gamification").Subrouter()
	protected.Use(authMiddleware.RequireAuth)

	// User gamification status and data
	protected.HandleFunc("/status", h.GetUserStatus).Methods("GET")
	
	// Event tracking
	protected.HandleFunc("/track-events", h.TrackEvents).Methods("POST")
	protected.HandleFunc("/process-events", h.ProcessEvents).Methods("POST")
	
	// Achievement management
	protected.HandleFunc("/unlock-achievement", h.UnlockAchievement).Methods("POST")
	
	// Progress and statistics updates
	protected.HandleFunc("/update-progress", h.UpdateProgress).Methods("POST")
	protected.HandleFunc("/update-stats", h.UpdateStatistics).Methods("POST")
	
	// Leaderboard (public but might want to require auth later)
	protected.HandleFunc("/leaderboard", h.GetLeaderboard).Methods("GET")

	// Admin routes (require admin auth)
	admin := r.PathPrefix("/api/admin/gamification").Subrouter()
	admin.Use(authMiddleware.RequireAuth) // Should also check for admin role
	
	// Achievement management for admins
	admin.HandleFunc("/achievements", h.CreateAchievement).Methods("POST")
}