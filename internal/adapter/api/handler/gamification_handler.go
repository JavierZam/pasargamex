package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"pasargamex/internal/usecase"
	"pasargamex/pkg/logger"

	"github.com/gorilla/mux"
)

type GamificationHandler struct {
	gamificationUseCase usecase.GamificationUseCase
	logger              logger.Logger
}

func NewGamificationHandler(
	gamificationUseCase usecase.GamificationUseCase,
	logger logger.Logger,
) *GamificationHandler {
	return &GamificationHandler{
		gamificationUseCase: gamificationUseCase,
		logger:              logger,
	}
}

// GET /api/gamification/status
func (h *GamificationHandler) GetUserStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Initialize gamification if not exists
	if err := h.gamificationUseCase.InitializeUserGamification(r.Context(), userID); err != nil {
		h.logger.Error("Failed to initialize gamification", "userID", userID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to initialize gamification")
		return
	}

	status, err := h.gamificationUseCase.GetUserGamificationStatus(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get gamification status", "userID", userID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get gamification status")
		return
	}

	h.respondWithJSON(w, http.StatusOK, status)
}

// POST /api/gamification/track-events
func (h *GamificationHandler) TrackEvents(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var request struct {
		Events []usecase.GamificationEventRequest `json:"events"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	if len(request.Events) == 0 {
		h.respondWithError(w, http.StatusBadRequest, "No events provided")
		return
	}

	// Validate and set timestamps if missing
	for i, event := range request.Events {
		if event.Timestamp.IsZero() {
			request.Events[i].Timestamp = time.Now()
		}
	}

	if err := h.gamificationUseCase.TrackUserEvent(r.Context(), userID, request.Events); err != nil {
		h.logger.Error("Failed to track events", "userID", userID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to track events")
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"eventsTracked": len(request.Events),
		"message":       "Events tracked successfully",
	})
}

// POST /api/gamification/process-events
func (h *GamificationHandler) ProcessEvents(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	newAchievements, err := h.gamificationUseCase.ProcessUserEvents(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to process events", "userID", userID, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to process events")
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success":         true,
		"newAchievements": newAchievements,
		"count":          len(newAchievements),
	})
}

// POST /api/gamification/unlock-achievement
func (h *GamificationHandler) UnlockAchievement(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var request struct {
		AchievementID string                 `json:"achievementId"`
		TriggerData   map[string]interface{} `json:"triggerData,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	if request.AchievementID == "" {
		h.respondWithError(w, http.StatusBadRequest, "Achievement ID is required")
		return
	}

	if err := h.gamificationUseCase.UnlockAchievement(r.Context(), userID, request.AchievementID, request.TriggerData); err != nil {
		h.logger.Error("Failed to unlock achievement", 
			"userID", userID, 
			"achievementID", request.AchievementID, 
			"error", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to unlock achievement")
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"achievementId": request.AchievementID,
		"message":       "Achievement unlocked successfully",
	})
}

// GET /api/gamification/leaderboard
func (h *GamificationHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default
	
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	leaderboard, err := h.gamificationUseCase.GetLeaderboard(r.Context(), limit)
	if err != nil {
		h.logger.Error("Failed to get leaderboard", "error", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get leaderboard")
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"leaderboard": leaderboard,
		"limit":       limit,
		"count":       len(leaderboard),
	})
}

// POST /api/gamification/update-progress
func (h *GamificationHandler) UpdateProgress(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var request struct {
		ProgressType string `json:"progressType"`
		Value        int64  `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	if request.ProgressType == "" {
		h.respondWithError(w, http.StatusBadRequest, "Progress type is required")
		return
	}

	if err := h.gamificationUseCase.UpdateUserProgress(r.Context(), userID, request.ProgressType, request.Value); err != nil {
		h.logger.Error("Failed to update progress", 
			"userID", userID, 
			"progressType", request.ProgressType, 
			"error", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to update progress")
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"progressType": request.ProgressType,
		"value":        request.Value,
		"message":      "Progress updated successfully",
	})
}

// POST /api/gamification/update-stats
func (h *GamificationHandler) UpdateStatistics(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var request struct {
		StatType  string `json:"statType"`
		Increment int    `json:"increment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	if request.StatType == "" {
		h.respondWithError(w, http.StatusBadRequest, "Stat type is required")
		return
	}

	if request.Increment <= 0 {
		request.Increment = 1
	}

	if err := h.gamificationUseCase.UpdateUserStatistics(r.Context(), userID, request.StatType, request.Increment); err != nil {
		h.logger.Error("Failed to update statistics", 
			"userID", userID, 
			"statType", request.StatType, 
			"error", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to update statistics")
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"statType":  request.StatType,
		"increment": request.Increment,
		"message":   "Statistics updated successfully",
	})
}

// Admin endpoint to create achievements
// POST /api/admin/gamification/achievements
func (h *GamificationHandler) CreateAchievement(w http.ResponseWriter, r *http.Request) {
	// This would typically require admin authentication
	// For now, we'll just respond with not implemented
	h.respondWithError(w, http.StatusNotImplemented, "Admin functionality not implemented")
}

// Webhook endpoint for transaction events
// POST /api/gamification/webhook/transaction
func (h *GamificationHandler) TransactionWebhook(w http.ResponseWriter, r *http.Request) {
	var request struct {
		UserID          string  `json:"userId"`
		TransactionType string  `json:"transactionType"` // "purchase" or "sale"
		Amount          float64 `json:"amount"`
		GameTitle       string  `json:"gameTitle,omitempty"`
		TransactionID   string  `json:"transactionId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	if request.UserID == "" || request.TransactionType == "" {
		h.respondWithError(w, http.StatusBadRequest, "UserID and TransactionType are required")
		return
	}

	// Create transaction event
	events := []usecase.GamificationEventRequest{
		{
			Type:      "transaction_complete",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"type":          request.TransactionType,
				"amount":        request.Amount,
				"gameTitle":     request.GameTitle,
				"transactionId": request.TransactionID,
			},
		},
	}

	if err := h.gamificationUseCase.TrackUserEvent(r.Context(), request.UserID, events); err != nil {
		h.logger.Error("Failed to track transaction event", 
			"userID", request.UserID, 
			"transactionType", request.TransactionType, 
			"error", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to track transaction")
		return
	}

	// Update user progress
	progressType := "purchases"
	if request.TransactionType == "sale" {
		progressType = "sales"
	}

	if err := h.gamificationUseCase.UpdateUserProgress(r.Context(), request.UserID, progressType, int64(request.Amount)); err != nil {
		h.logger.Error("Failed to update user progress", 
			"userID", request.UserID, 
			"progressType", progressType, 
			"error", err)
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Transaction event tracked successfully",
	})
}

// Helper methods
func (h *GamificationHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, map[string]string{"error": message})
}

func (h *GamificationHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}