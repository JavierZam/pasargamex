package usecase

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/logger"
)

type GamificationUseCase interface {
	// User Status
	GetUserGamificationStatus(ctx context.Context, userID string) (*entity.GamificationStatusResponse, error)
	InitializeUserGamification(ctx context.Context, userID string) error
	
	// Event Tracking
	TrackUserEvent(ctx context.Context, userID string, events []GamificationEventRequest) error
	ProcessUserEvents(ctx context.Context, userID string) ([]entity.Achievement, error)
	
	// Achievement Management
	UnlockAchievement(ctx context.Context, userID, achievementID string, triggerData map[string]interface{}) error
	CheckAchievementEligibility(ctx context.Context, userID string, achievement entity.Achievement) (bool, *entity.AchievementProgress, error)
	
	// Progress Updates
	UpdateUserProgress(ctx context.Context, userID string, progressType string, value int64) error
	UpdateUserTitle(ctx context.Context, userID string) error
	
	// Statistics
	GetLeaderboard(ctx context.Context, limit int) ([]entity.UserGamification, error)
	UpdateUserStatistics(ctx context.Context, userID string, statType string, increment int) error
}

type gamificationUseCase struct {
	gamificationRepo repository.GamificationRepository
	userRepo         repository.UserRepository
	logger           logger.Logger
}

type GamificationEventRequest struct {
	Type      string                 `json:"type"`
	TriggerID string                 `json:"triggerId,omitempty"`
	Count     int                    `json:"count,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

func NewGamificationUseCase(
	gamificationRepo repository.GamificationRepository,
	userRepo repository.UserRepository,
	logger logger.Logger,
) GamificationUseCase {
	return &gamificationUseCase{
		gamificationRepo: gamificationRepo,
		userRepo:         userRepo,
		logger:           logger,
	}
}

// Get complete user gamification status
func (uc *gamificationUseCase) GetUserGamificationStatus(ctx context.Context, userID string) (*entity.GamificationStatusResponse, error) {
	// Get user gamification data
	userGamification, err := uc.gamificationRepo.GetUserGamification(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user gamification: %w", err)
	}

	// Get user achievements
	userAchievements, err := uc.gamificationRepo.GetUserAchievements(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user achievements: %w", err)
	}

	// Get all master achievements
	allAchievements, err := uc.gamificationRepo.GetAllAchievements(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all achievements: %w", err)
	}

	// Get current title
	currentTitle, err := uc.gamificationRepo.GetUserCurrentTitle(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current title: %w", err)
	}

	// Get next title
	nextTitle, err := uc.gamificationRepo.GetNextTitle(ctx, currentTitle.Level)
	if err != nil {
		uc.logger.Warn("Failed to get next title", "error", err)
	}

	// Map user achievements
	userAchievementMap := make(map[string]entity.UserAchievement)
	for _, ua := range userAchievements {
		userAchievementMap[ua.AchievementID] = ua
	}

	// Build achievement status response
	achievementStatuses := make([]entity.AchievementStatusResponse, 0, len(allAchievements))
	newAchievements := make([]entity.Achievement, 0)

	for _, achievement := range allAchievements {
		status := entity.AchievementStatusResponse{
			Achievement: achievement,
			Unlocked:    false,
		}

		if userAchievement, exists := userAchievementMap[achievement.ID]; exists {
			status.Unlocked = true
			status.UnlockedAt = &userAchievement.UnlockedAt
			status.Progress = &userAchievement.Progress

			// Check if achievement was unlocked recently (within 24 hours)
			if time.Since(userAchievement.UnlockedAt) < 24*time.Hour {
				newAchievements = append(newAchievements, achievement)
			}
		} else {
			// Calculate progress for locked achievements
			eligible, progress, err := uc.CheckAchievementEligibility(ctx, userID, achievement)
			if err != nil {
				uc.logger.Warn("Failed to check achievement eligibility", "achievementId", achievement.ID, "error", err)
			} else if progress != nil {
				status.Progress = progress
			}

			// Auto-unlock if eligible
			if eligible && progress != nil && progress.Completed {
				if err := uc.UnlockAchievement(ctx, userID, achievement.ID, nil); err != nil {
					uc.logger.Error("Failed to auto-unlock achievement", "achievementId", achievement.ID, "error", err)
				} else {
					status.Unlocked = true
					now := time.Now()
					status.UnlockedAt = &now
					newAchievements = append(newAchievements, achievement)
				}
			}
		}

		achievementStatuses = append(achievementStatuses, status)
	}

	// Calculate stats
	stats := entity.GamificationStats{
		TotalPoints:          userGamification.TotalPoints,
		AchievementsUnlocked: len(userAchievements),
		TotalAchievements:    len(allAchievements),
		SecretsFound:         uc.countSecretAchievements(userAchievements),
		Level:                userGamification.CurrentLevel,
		NextLevelPoints:      1000, // Fixed for now
	}

	response := &entity.GamificationStatusResponse{
		User:            userGamification,
		CurrentTitle:    currentTitle,
		NextTitle:       nextTitle,
		Achievements:    achievementStatuses,
		NewAchievements: newAchievements,
		Stats:           stats,
	}

	return response, nil
}

// Initialize gamification for new user
func (uc *gamificationUseCase) InitializeUserGamification(ctx context.Context, userID string) error {
	// Check if user already has gamification data
	existing, err := uc.gamificationRepo.GetUserGamification(ctx, userID)
	if err == nil && existing != nil {
		return nil // Already initialized
	}

	// Create initial gamification data
	gamification := &entity.UserGamification{
		UserID:         userID,
		TotalPoints:    0,
		CurrentLevel:   1,
		CurrentTitleID: "human", // Default title
		TotalSales:     0,
		TotalPurchases: 0,
		JoinedAt:       time.Now(),
		LastActiveAt:   time.Now(),
		Streaks: entity.UserStreaks{
			LoginDays:       1,
			LastLoginDate:   time.Now().Format("2006-01-02"),
			TradingDays:     0,
			LastTradingDate: "",
		},
		SecretTriggers: make(map[string]int),
		Statistics:     entity.UserStatistics{},
	}

	if err := uc.gamificationRepo.CreateUserGamification(ctx, gamification); err != nil {
		return fmt.Errorf("failed to create user gamification: %w", err)
	}

	// Check for milestone achievements (like "Founding Father")
	if err := uc.checkMilestoneAchievements(ctx, userID); err != nil {
		uc.logger.Warn("Failed to check milestone achievements", "userID", userID, "error", err)
	}

	return nil
}

// Track user events (batch processing)
func (uc *gamificationUseCase) TrackUserEvent(ctx context.Context, userID string, events []GamificationEventRequest) error {
	for _, eventReq := range events {
		event := &entity.GamificationEvent{
			ID:        fmt.Sprintf("%s_%d", userID, time.Now().UnixNano()),
			UserID:    userID,
			EventType: eventReq.Type,
			TriggerID: eventReq.TriggerID,
			Timestamp: eventReq.Timestamp,
			Data:      eventReq.Data,
			Processed: false,
		}

		if err := uc.gamificationRepo.CreateGamificationEvent(ctx, event); err != nil {
			return fmt.Errorf("failed to create event: %w", err)
		}

		// Handle secret triggers immediately
		if eventReq.Type == "secret_trigger" && eventReq.TriggerID != "" {
			count := eventReq.Count
			if count == 0 {
				count = 1
			}
			
			if err := uc.gamificationRepo.IncrementSecretTrigger(ctx, userID, eventReq.TriggerID, count); err != nil {
				uc.logger.Error("Failed to increment secret trigger", "triggerID", eventReq.TriggerID, "error", err)
			}
		}
	}

	// Process events immediately for better UX
	go func() {
		newAchievements, err := uc.ProcessUserEvents(context.Background(), userID)
		if err != nil {
			uc.logger.Error("Failed to process user events", "userID", userID, "error", err)
		} else if len(newAchievements) > 0 {
			uc.logger.Info("New achievements unlocked", "userID", userID, "count", len(newAchievements))
		}
	}()

	return nil
}

// Process pending events and check for new achievements
func (uc *gamificationUseCase) ProcessUserEvents(ctx context.Context, userID string) ([]entity.Achievement, error) {
	events, err := uc.gamificationRepo.GetUnprocessedEvents(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unprocessed events: %w", err)
	}

	var newAchievements []entity.Achievement

	for _, event := range events {
		// Process different event types
		switch event.EventType {
		case "secret_trigger":
			achievements, err := uc.processSecretTriggerEvent(ctx, userID, event)
			if err != nil {
				uc.logger.Error("Failed to process secret trigger event", "eventID", event.ID, "error", err)
				continue
			}
			newAchievements = append(newAchievements, achievements...)

		case "transaction_complete":
			achievements, err := uc.processTransactionEvent(ctx, userID, event)
			if err != nil {
				uc.logger.Error("Failed to process transaction event", "eventID", event.ID, "error", err)
				continue
			}
			newAchievements = append(newAchievements, achievements...)
		}

		// Mark event as processed
		if err := uc.gamificationRepo.MarkEventProcessed(ctx, event.ID); err != nil {
			uc.logger.Error("Failed to mark event as processed", "eventID", event.ID, "error", err)
		}
	}

	return newAchievements, nil
}

// Unlock specific achievement for user
func (uc *gamificationUseCase) UnlockAchievement(ctx context.Context, userID, achievementID string, triggerData map[string]interface{}) error {
	// Get achievement details
	achievement, err := uc.gamificationRepo.GetAchievementByID(ctx, achievementID)
	if err != nil {
		return fmt.Errorf("failed to get achievement: %w", err)
	}

	// Check if already unlocked
	existing, err := uc.gamificationRepo.GetUserAchievement(ctx, userID, achievementID)
	if err == nil && existing != nil {
		return nil // Already unlocked
	}

	// Create user achievement
	userAchievement := &entity.UserAchievement{
		AchievementID: achievementID,
		UserID:        userID,
		IsSecret:      achievement.IsSecret,
		Category:      achievement.Category,
		Points:        achievement.Points,
		Progress: entity.AchievementProgress{
			Current:   1,
			Target:    1,
			Completed: true,
		},
		TriggerData: triggerData,
	}

	if err := uc.gamificationRepo.UnlockAchievement(ctx, userAchievement); err != nil {
		return fmt.Errorf("failed to unlock achievement: %w", err)
	}

	// Update user points
	gamification, err := uc.gamificationRepo.GetUserGamification(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user gamification: %w", err)
	}

	gamification.TotalPoints += achievement.Points
	gamification.CurrentLevel = (gamification.TotalPoints / 1000) + 1

	if err := uc.gamificationRepo.UpdateUserGamification(ctx, gamification); err != nil {
		return fmt.Errorf("failed to update user points: %w", err)
	}

	// Check if user qualifies for new title
	if err := uc.UpdateUserTitle(ctx, userID); err != nil {
		uc.logger.Error("Failed to update user title", "userID", userID, "error", err)
	}

	uc.logger.Info("Achievement unlocked", "userID", userID, "achievementID", achievementID, "points", achievement.Points)

	return nil
}

// Helper functions
func (uc *gamificationUseCase) processSecretTriggerEvent(ctx context.Context, userID string, event entity.GamificationEvent) ([]entity.Achievement, error) {
	var newAchievements []entity.Achievement

	// Get user's current trigger counts
	gamification, err := uc.gamificationRepo.GetUserGamification(ctx, userID)
	if err != nil {
		return nil, err
	}

	triggerID := event.TriggerID
	currentCount := gamification.SecretTriggers[triggerID]

	// Check trigger-based achievements
	achievements := map[string]int{
		"logo_clicks":              25, // Logo Lover
		"floating_button_clicks":   50, // Click Master  
		"chat_dashboard_switches":  10, // No One Text You Yet
	}

	if targetCount, exists := achievements[triggerID]; exists && currentCount >= targetCount {
		var achievementID string
		switch triggerID {
		case "logo_clicks":
			achievementID = "logo_lover"
		case "floating_button_clicks":
			achievementID = "click_master"
		case "chat_dashboard_switches":
			achievementID = "no_one_text_you_yet"
		}

		if achievementID != "" {
			if err := uc.UnlockAchievement(ctx, userID, achievementID, map[string]interface{}{
				"triggerCount": currentCount,
				"triggerType":  triggerID,
			}); err == nil {
				achievement, _ := uc.gamificationRepo.GetAchievementByID(ctx, achievementID)
				if achievement != nil {
					newAchievements = append(newAchievements, *achievement)
				}
			}
		}
	}

	return newAchievements, nil
}

func (uc *gamificationUseCase) processTransactionEvent(ctx context.Context, userID string, event entity.GamificationEvent) ([]entity.Achievement, error) {
	var newAchievements []entity.Achievement

	// Extract transaction data
	transactionType, _ := event.Data["type"].(string)
	amount, _ := event.Data["amount"].(float64)

	// Update user statistics based on transaction
	gamification, err := uc.gamificationRepo.GetUserGamification(ctx, userID)
	if err != nil {
		return nil, err
	}

	if transactionType == "purchase" {
		gamification.TotalPurchases += int64(amount)
		gamification.Statistics.TotalTransactions++

		// Check first purchase achievement
		if gamification.Statistics.TotalTransactions == 1 {
			if err := uc.UnlockAchievement(ctx, userID, "first_purchase", event.Data); err == nil {
				achievement, _ := uc.gamificationRepo.GetAchievementByID(ctx, "first_purchase")
				if achievement != nil {
					newAchievements = append(newAchievements, *achievement)
				}
			}
		}
	} else if transactionType == "sale" {
		gamification.TotalSales += int64(amount)
		gamification.Statistics.SuccessfulSales++

		// Check first sale achievement
		if gamification.Statistics.SuccessfulSales == 1 {
			if err := uc.UnlockAchievement(ctx, userID, "first_sale", event.Data); err == nil {
				achievement, _ := uc.gamificationRepo.GetAchievementByID(ctx, "first_sale")
				if achievement != nil {
					newAchievements = append(newAchievements, *achievement)
				}
			}
		}
	}

	// Update gamification data
	if err := uc.gamificationRepo.UpdateUserGamification(ctx, gamification); err != nil {
		return nil, err
	}

	return newAchievements, nil
}

func (uc *gamificationUseCase) CheckAchievementEligibility(ctx context.Context, userID string, achievement entity.Achievement) (bool, *entity.AchievementProgress, error) {
	gamification, err := uc.gamificationRepo.GetUserGamification(ctx, userID)
	if err != nil {
		return false, nil, err
	}

	switch achievement.Requirement.Type {
	case "sales_amount":
		current := gamification.TotalSales
		target := achievement.Requirement.Value
		progress := &entity.AchievementProgress{
			Current:   int(current),
			Target:    int(target),
			Completed: current >= target,
		}
		return current >= target, progress, nil

	case "purchase_amount":
		current := gamification.TotalPurchases
		target := achievement.Requirement.Value
		progress := &entity.AchievementProgress{
			Current:   int(current),
			Target:    int(target),
			Completed: current >= target,
		}
		return current >= target, progress, nil

	case "secret_trigger":
		triggerID := achievement.Requirement.Condition
		current := gamification.SecretTriggers[triggerID]
		target := int(achievement.Requirement.Value)
		progress := &entity.AchievementProgress{
			Current:   current,
			Target:    target,
			Completed: current >= target,
		}
		return current >= target, progress, nil
	}

	return false, nil, nil
}

func (uc *gamificationUseCase) UpdateUserTitle(ctx context.Context, userID string) error {
	gamification, err := uc.gamificationRepo.GetUserGamification(ctx, userID)
	if err != nil {
		return err
	}

	// Get all titles
	titles, err := uc.gamificationRepo.GetAllTitles(ctx)
	if err != nil {
		return err
	}

	// Find highest qualified title
	var qualifiedTitle *entity.UserTitle
	for _, title := range titles {
		if title.Requirement.Type == "sales" && gamification.TotalSales >= title.Requirement.Value {
			if qualifiedTitle == nil || title.Level > qualifiedTitle.Level {
				qualifiedTitle = &title
			}
		}
	}

	// Update title if changed
	if qualifiedTitle != nil && qualifiedTitle.ID != gamification.CurrentTitleID {
		gamification.CurrentTitleID = qualifiedTitle.ID
		return uc.gamificationRepo.UpdateUserGamification(ctx, gamification)
	}

	return nil
}

func (uc *gamificationUseCase) checkMilestoneAchievements(ctx context.Context, userID string) error {
	// Check "Founding Father" achievement for first 1000 users
	// This would require additional logic to count total verified users
	return nil
}

func (uc *gamificationUseCase) countSecretAchievements(achievements []entity.UserAchievement) int {
	count := 0
	for _, achievement := range achievements {
		if achievement.IsSecret {
			count++
		}
	}
	return count
}

func (uc *gamificationUseCase) UpdateUserProgress(ctx context.Context, userID string, progressType string, value int64) error {
	updates := map[string]interface{}{}

	switch progressType {
	case "sales":
		updates["totalSales"] = value
	case "purchases":
		updates["totalPurchases"] = value
	}

	return uc.gamificationRepo.BatchUpdateUserProgress(ctx, userID, updates)
}

func (uc *gamificationUseCase) GetLeaderboard(ctx context.Context, limit int) ([]entity.UserGamification, error) {
	return uc.gamificationRepo.GetTopUsers(ctx, limit)
}

func (uc *gamificationUseCase) UpdateUserStatistics(ctx context.Context, userID string, statType string, increment int) error {
	gamification, err := uc.gamificationRepo.GetUserGamification(ctx, userID)
	if err != nil {
		return err
	}

	switch statType {
	case "product_views":
		gamification.Statistics.ProductViews += increment
	case "search_queries":
		gamification.Statistics.SearchQueries += increment
	case "positive_reviews":
		gamification.Statistics.PositiveReviews += increment
	}

	return uc.gamificationRepo.UpdateUserStatistics(ctx, userID, gamification.Statistics)
}