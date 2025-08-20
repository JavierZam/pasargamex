package repository

import (
	"context"
	"pasargamex/internal/domain/entity"
)

type GamificationRepository interface {
	// User Gamification Data
	GetUserGamification(ctx context.Context, userID string) (*entity.UserGamification, error)
	CreateUserGamification(ctx context.Context, gamification *entity.UserGamification) error
	UpdateUserGamification(ctx context.Context, gamification *entity.UserGamification) error
	
	// User Achievements
	GetUserAchievements(ctx context.Context, userID string) ([]entity.UserAchievement, error)
	GetUserAchievement(ctx context.Context, userID, achievementID string) (*entity.UserAchievement, error)
	UnlockAchievement(ctx context.Context, userAchievement *entity.UserAchievement) error
	UpdateAchievementProgress(ctx context.Context, userID, achievementID string, progress entity.AchievementProgress) error
	
	// Master Achievements
	GetAllAchievements(ctx context.Context) ([]entity.Achievement, error)
	GetAchievementByID(ctx context.Context, achievementID string) (*entity.Achievement, error)
	GetAchievementsByCategory(ctx context.Context, category entity.AchievementCategory) ([]entity.Achievement, error)
	CreateAchievement(ctx context.Context, achievement *entity.Achievement) error
	UpdateAchievement(ctx context.Context, achievement *entity.Achievement) error
	
	// User Titles
	GetAllTitles(ctx context.Context) ([]entity.UserTitle, error)
	GetUserCurrentTitle(ctx context.Context, userID string) (*entity.UserTitle, error)
	GetNextTitle(ctx context.Context, currentLevel int) (*entity.UserTitle, error)
	
	// Gamification Events
	CreateGamificationEvent(ctx context.Context, event *entity.GamificationEvent) error
	GetUnprocessedEvents(ctx context.Context, userID string) ([]entity.GamificationEvent, error)
	MarkEventProcessed(ctx context.Context, eventID string) error
	
	// Statistics and Leaderboards
	GetTopUsers(ctx context.Context, limit int) ([]entity.UserGamification, error)
	UpdateUserStatistics(ctx context.Context, userID string, stats entity.UserStatistics) error
	IncrementSecretTrigger(ctx context.Context, userID, triggerID string, increment int) error
	
	// Batch Operations
	BatchUpdateUserProgress(ctx context.Context, userID string, updates map[string]interface{}) error
}