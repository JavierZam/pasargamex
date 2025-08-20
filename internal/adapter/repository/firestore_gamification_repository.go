package repository

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
)

type firestoreGamificationRepository struct {
	client *firestore.Client
}

func NewFirestoreGamificationRepository(client *firestore.Client) repository.GamificationRepository {
	return &firestoreGamificationRepository{
		client: client,
	}
}

// User Gamification Data
func (r *firestoreGamificationRepository) GetUserGamification(ctx context.Context, userID string) (*entity.UserGamification, error) {
	doc, err := r.client.Collection("users").Doc(userID).Collection("gamification").Doc("data").Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user gamification: %w", err)
	}

	var gamification entity.UserGamification
	if err := doc.DataTo(&gamification); err != nil {
		return nil, fmt.Errorf("failed to decode gamification data: %w", err)
	}

	return &gamification, nil
}

func (r *firestoreGamificationRepository) CreateUserGamification(ctx context.Context, gamification *entity.UserGamification) error {
	gamification.CreatedAt = time.Now()
	gamification.UpdatedAt = time.Now()

	_, err := r.client.Collection("users").Doc(gamification.UserID).Collection("gamification").Doc("data").Set(ctx, gamification)
	if err != nil {
		return fmt.Errorf("failed to create user gamification: %w", err)
	}

	return nil
}

func (r *firestoreGamificationRepository) UpdateUserGamification(ctx context.Context, gamification *entity.UserGamification) error {
	gamification.UpdatedAt = time.Now()

	_, err := r.client.Collection("users").Doc(gamification.UserID).Collection("gamification").Doc("data").Set(ctx, gamification, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("failed to update user gamification: %w", err)
	}

	return nil
}

// User Achievements
func (r *firestoreGamificationRepository) GetUserAchievements(ctx context.Context, userID string) ([]entity.UserAchievement, error) {
	iter := r.client.Collection("users").Doc(userID).Collection("achievements").Documents(ctx)
	defer iter.Stop()

	var achievements []entity.UserAchievement
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate achievements: %w", err)
		}

		var achievement entity.UserAchievement
		if err := doc.DataTo(&achievement); err != nil {
			return nil, fmt.Errorf("failed to decode achievement: %w", err)
		}
		achievements = append(achievements, achievement)
	}

	return achievements, nil
}

func (r *firestoreGamificationRepository) GetUserAchievement(ctx context.Context, userID, achievementID string) (*entity.UserAchievement, error) {
	doc, err := r.client.Collection("users").Doc(userID).Collection("achievements").Doc(achievementID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user achievement: %w", err)
	}

	var achievement entity.UserAchievement
	if err := doc.DataTo(&achievement); err != nil {
		return nil, fmt.Errorf("failed to decode achievement: %w", err)
	}

	return &achievement, nil
}

func (r *firestoreGamificationRepository) UnlockAchievement(ctx context.Context, userAchievement *entity.UserAchievement) error {
	userAchievement.UnlockedAt = time.Now()

	_, err := r.client.Collection("users").Doc(userAchievement.UserID).Collection("achievements").Doc(userAchievement.AchievementID).Set(ctx, userAchievement)
	if err != nil {
		return fmt.Errorf("failed to unlock achievement: %w", err)
	}

	return nil
}

func (r *firestoreGamificationRepository) UpdateAchievementProgress(ctx context.Context, userID, achievementID string, progress entity.AchievementProgress) error {
	_, err := r.client.Collection("users").Doc(userID).Collection("achievements").Doc(achievementID).Set(ctx, map[string]interface{}{
		"progress": progress,
	}, firestore.MergeAll)

	if err != nil {
		return fmt.Errorf("failed to update achievement progress: %w", err)
	}

	return nil
}

// Master Achievements
func (r *firestoreGamificationRepository) GetAllAchievements(ctx context.Context) ([]entity.Achievement, error) {
	iter := r.client.Collection("achievements").Where("isActive", "==", true).Documents(ctx)
	defer iter.Stop()

	var achievements []entity.Achievement
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate achievements: %w", err)
		}

		var achievement entity.Achievement
		if err := doc.DataTo(&achievement); err != nil {
			return nil, fmt.Errorf("failed to decode achievement: %w", err)
		}
		achievements = append(achievements, achievement)
	}

	return achievements, nil
}

func (r *firestoreGamificationRepository) GetAchievementByID(ctx context.Context, achievementID string) (*entity.Achievement, error) {
	doc, err := r.client.Collection("achievements").Doc(achievementID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get achievement: %w", err)
	}

	var achievement entity.Achievement
	if err := doc.DataTo(&achievement); err != nil {
		return nil, fmt.Errorf("failed to decode achievement: %w", err)
	}

	return &achievement, nil
}

func (r *firestoreGamificationRepository) GetAchievementsByCategory(ctx context.Context, category entity.AchievementCategory) ([]entity.Achievement, error) {
	iter := r.client.Collection("achievements").Where("category", "==", string(category)).Where("isActive", "==", true).Documents(ctx)
	defer iter.Stop()

	var achievements []entity.Achievement
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate achievements: %w", err)
		}

		var achievement entity.Achievement
		if err := doc.DataTo(&achievement); err != nil {
			return nil, fmt.Errorf("failed to decode achievement: %w", err)
		}
		achievements = append(achievements, achievement)
	}

	return achievements, nil
}

func (r *firestoreGamificationRepository) CreateAchievement(ctx context.Context, achievement *entity.Achievement) error {
	achievement.CreatedAt = time.Now()
	achievement.UpdatedAt = time.Now()

	_, err := r.client.Collection("achievements").Doc(achievement.ID).Set(ctx, achievement)
	if err != nil {
		return fmt.Errorf("failed to create achievement: %w", err)
	}

	return nil
}

func (r *firestoreGamificationRepository) UpdateAchievement(ctx context.Context, achievement *entity.Achievement) error {
	achievement.UpdatedAt = time.Now()

	_, err := r.client.Collection("achievements").Doc(achievement.ID).Set(ctx, achievement, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("failed to update achievement: %w", err)
	}

	return nil
}

// User Titles
func (r *firestoreGamificationRepository) GetAllTitles(ctx context.Context) ([]entity.UserTitle, error) {
	iter := r.client.Collection("titles").OrderBy("level", firestore.Asc).Documents(ctx)
	defer iter.Stop()

	var titles []entity.UserTitle
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate titles: %w", err)
		}

		var title entity.UserTitle
		if err := doc.DataTo(&title); err != nil {
			return nil, fmt.Errorf("failed to decode title: %w", err)
		}
		titles = append(titles, title)
	}

	return titles, nil
}

func (r *firestoreGamificationRepository) GetUserCurrentTitle(ctx context.Context, userID string) (*entity.UserTitle, error) {
	// Get user's current title ID from gamification data
	gamification, err := r.GetUserGamification(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get the title details
	doc, err := r.client.Collection("titles").Doc(gamification.CurrentTitleID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current title: %w", err)
	}

	var title entity.UserTitle
	if err := doc.DataTo(&title); err != nil {
		return nil, fmt.Errorf("failed to decode title: %w", err)
	}

	title.IsUnlocked = true
	return &title, nil
}

func (r *firestoreGamificationRepository) GetNextTitle(ctx context.Context, currentLevel int) (*entity.UserTitle, error) {
	iter := r.client.Collection("titles").Where("level", ">", currentLevel).OrderBy("level", firestore.Asc).Limit(1).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, nil // No next title
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get next title: %w", err)
	}

	var title entity.UserTitle
	if err := doc.DataTo(&title); err != nil {
		return nil, fmt.Errorf("failed to decode next title: %w", err)
	}

	title.IsUnlocked = false
	return &title, nil
}

// Gamification Events
func (r *firestoreGamificationRepository) CreateGamificationEvent(ctx context.Context, event *entity.GamificationEvent) error {
	event.Timestamp = time.Now()

	_, err := r.client.Collection("gamification_events").Doc(event.ID).Set(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to create gamification event: %w", err)
	}

	return nil
}

func (r *firestoreGamificationRepository) GetUnprocessedEvents(ctx context.Context, userID string) ([]entity.GamificationEvent, error) {
	iter := r.client.Collection("gamification_events").Where("userId", "==", userID).Where("processed", "==", false).Documents(ctx)
	defer iter.Stop()

	var events []entity.GamificationEvent
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate events: %w", err)
		}

		var event entity.GamificationEvent
		if err := doc.DataTo(&event); err != nil {
			return nil, fmt.Errorf("failed to decode event: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

func (r *firestoreGamificationRepository) MarkEventProcessed(ctx context.Context, eventID string) error {
	_, err := r.client.Collection("gamification_events").Doc(eventID).Set(ctx, map[string]interface{}{
		"processed": true,
	}, firestore.MergeAll)

	if err != nil {
		return fmt.Errorf("failed to mark event processed: %w", err)
	}

	return nil
}

// Statistics and Operations
func (r *firestoreGamificationRepository) GetTopUsers(ctx context.Context, limit int) ([]entity.UserGamification, error) {
	iter := r.client.CollectionGroup("gamification").Where("totalPoints", ">", 0).OrderBy("totalPoints", firestore.Desc).Limit(limit).Documents(ctx)
	defer iter.Stop()

	var users []entity.UserGamification
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate top users: %w", err)
		}

		var user entity.UserGamification
		if err := doc.DataTo(&user); err != nil {
			return nil, fmt.Errorf("failed to decode user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *firestoreGamificationRepository) UpdateUserStatistics(ctx context.Context, userID string, stats entity.UserStatistics) error {
	_, err := r.client.Collection("users").Doc(userID).Collection("gamification").Doc("data").Set(ctx, map[string]interface{}{
		"statistics": stats,
		"updatedAt":  time.Now(),
	}, firestore.MergeAll)

	if err != nil {
		return fmt.Errorf("failed to update user statistics: %w", err)
	}

	return nil
}

func (r *firestoreGamificationRepository) IncrementSecretTrigger(ctx context.Context, userID, triggerID string, increment int) error {
	docRef := r.client.Collection("users").Doc(userID).Collection("gamification").Doc("data")

	return r.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(docRef)
		if err != nil {
			return err
		}

		var gamification entity.UserGamification
		if err := doc.DataTo(&gamification); err != nil {
			return err
		}

		if gamification.SecretTriggers == nil {
			gamification.SecretTriggers = make(map[string]int)
		}

		gamification.SecretTriggers[triggerID] += increment
		gamification.UpdatedAt = time.Now()

		return tx.Set(docRef, gamification, firestore.MergeAll)
	})
}

func (r *firestoreGamificationRepository) BatchUpdateUserProgress(ctx context.Context, userID string, updates map[string]interface{}) error {
	updates["updatedAt"] = time.Now()

	_, err := r.client.Collection("users").Doc(userID).Collection("gamification").Doc("data").Set(ctx, updates, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("failed to batch update user progress: %w", err)
	}

	return nil
}