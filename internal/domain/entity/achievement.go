package entity

import (
	"time"
)

type Achievement struct {
	ID          string                 `firestore:"id" json:"id"`
	Title       string                 `firestore:"title" json:"title"`
	Description string                 `firestore:"description" json:"description"`
	Icon        string                 `firestore:"icon" json:"icon"`
	Category    AchievementCategory    `firestore:"category" json:"category"`
	Rarity      AchievementRarity      `firestore:"rarity" json:"rarity"`
	Points      int                    `firestore:"points" json:"points"`
	Requirement AchievementRequirement `firestore:"requirement" json:"requirement"`
	IsSecret    bool                   `firestore:"isSecret" json:"isSecret"`
	IsActive    bool                   `firestore:"isActive" json:"isActive"`
	Hint        string                 `firestore:"hint,omitempty" json:"hint,omitempty"`
	CreatedAt   time.Time              `firestore:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time              `firestore:"updatedAt" json:"updatedAt"`
}

type UserAchievement struct {
	AchievementID string                    `firestore:"achievementId" json:"achievementId"`
	UserID        string                    `firestore:"userId" json:"userId"`
	UnlockedAt    time.Time                 `firestore:"unlockedAt" json:"unlockedAt"`
	IsSecret      bool                      `firestore:"isSecret" json:"isSecret"`
	Category      AchievementCategory       `firestore:"category" json:"category"`
	Points        int                       `firestore:"points" json:"points"`
	Progress      AchievementProgress       `firestore:"progress" json:"progress"`
	TriggerData   map[string]interface{}    `firestore:"triggerData,omitempty" json:"triggerData,omitempty"`
}

type UserGamification struct {
	UserID           string                    `firestore:"userId" json:"userId"`
	TotalPoints      int                       `firestore:"totalPoints" json:"totalPoints"`
	CurrentLevel     int                       `firestore:"currentLevel" json:"currentLevel"`
	CurrentTitleID   string                    `firestore:"currentTitleId" json:"currentTitleId"`
	TotalSales       int64                     `firestore:"totalSales" json:"totalSales"`
	TotalPurchases   int64                     `firestore:"totalPurchases" json:"totalPurchases"`
	JoinedAt         time.Time                 `firestore:"joinedAt" json:"joinedAt"`
	LastActiveAt     time.Time                 `firestore:"lastActiveAt" json:"lastActiveAt"`
	Streaks          UserStreaks               `firestore:"streaks" json:"streaks"`
	SecretTriggers   map[string]int            `firestore:"secretTriggers" json:"secretTriggers"`
	Statistics       UserStatistics            `firestore:"statistics" json:"statistics"`
	CreatedAt        time.Time                 `firestore:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time                 `firestore:"updatedAt" json:"updatedAt"`
}

type UserTitle struct {
	ID          string            `firestore:"id" json:"id"`
	Name        string            `firestore:"name" json:"name"`
	Description string            `firestore:"description" json:"description"`
	Icon        string            `firestore:"icon" json:"icon"`
	Level       int               `firestore:"level" json:"level"`
	Requirement TitleRequirement  `firestore:"requirement" json:"requirement"`
	Color       string            `firestore:"color" json:"color"`
	Gradient    string            `firestore:"gradient" json:"gradient"`
	IsUnlocked  bool              `json:"isUnlocked"` // Computed field
}

type GamificationEvent struct {
	ID        string                 `firestore:"id" json:"id"`
	UserID    string                 `firestore:"userId" json:"userId"`
	EventType string                 `firestore:"eventType" json:"eventType"`
	TriggerID string                 `firestore:"triggerId,omitempty" json:"triggerId,omitempty"`
	Timestamp time.Time              `firestore:"timestamp" json:"timestamp"`
	Data      map[string]interface{} `firestore:"data" json:"data"`
	Processed bool                   `firestore:"processed" json:"processed"`
}

type AchievementRequirement struct {
	Type      string                 `firestore:"type" json:"type"`
	Value     int64                  `firestore:"value" json:"value"`
	Condition string                 `firestore:"condition,omitempty" json:"condition,omitempty"`
	Data      map[string]interface{} `firestore:"data,omitempty" json:"data,omitempty"`
}

type AchievementProgress struct {
	Current   int  `firestore:"current" json:"current"`
	Target    int  `firestore:"target" json:"target"`
	Completed bool `firestore:"completed" json:"completed"`
}

type TitleRequirement struct {
	Type  string `firestore:"type" json:"type"`
	Value int64  `firestore:"value" json:"value"`
}

type UserStreaks struct {
	LoginDays       int    `firestore:"loginDays" json:"loginDays"`
	LastLoginDate   string `firestore:"lastLoginDate" json:"lastLoginDate"`
	TradingDays     int    `firestore:"tradingDays" json:"tradingDays"`
	LastTradingDate string `firestore:"lastTradingDate" json:"lastTradingDate"`
}

type UserStatistics struct {
	TotalTransactions  int `firestore:"totalTransactions" json:"totalTransactions"`
	SuccessfulSales    int `firestore:"successfulSales" json:"successfulSales"`
	PositiveReviews    int `firestore:"positiveReviews" json:"positiveReviews"`
	HelpedUsers        int `firestore:"helpedUsers" json:"helpedUsers"`
	ProductViews       int `firestore:"productViews" json:"productViews"`
	SearchQueries      int `firestore:"searchQueries" json:"searchQueries"`
}

type AchievementCategory string

const (
	AchievementCategoryTrading   AchievementCategory = "trading"
	AchievementCategorySocial    AchievementCategory = "social"
	AchievementCategorySecret    AchievementCategory = "secret"
	AchievementCategoryMilestone AchievementCategory = "milestone"
	AchievementCategorySpecial   AchievementCategory = "special"
)

type AchievementRarity string

const (
	AchievementRarityCommon    AchievementRarity = "common"
	AchievementRarityRare      AchievementRarity = "rare"
	AchievementRarityEpic      AchievementRarity = "epic"
	AchievementRarityLegendary AchievementRarity = "legendary"
	AchievementRarityMythic    AchievementRarity = "mythic"
)

// Achievement Status Response
type AchievementStatusResponse struct {
	Achievement Achievement `json:"achievement"`
	Unlocked    bool        `json:"unlocked"`
	UnlockedAt  *time.Time  `json:"unlockedAt,omitempty"`
	Progress    *AchievementProgress `json:"progress,omitempty"`
}

// Gamification Status Response
type GamificationStatusResponse struct {
	User                *UserGamification            `json:"user"`
	CurrentTitle        *UserTitle                   `json:"currentTitle"`
	NextTitle           *UserTitle                   `json:"nextTitle,omitempty"`
	Achievements        []AchievementStatusResponse  `json:"achievements"`
	NewAchievements     []Achievement                `json:"newAchievements"`
	Stats               GamificationStats            `json:"stats"`
}

type GamificationStats struct {
	TotalPoints          int `json:"totalPoints"`
	AchievementsUnlocked int `json:"achievementsUnlocked"`
	TotalAchievements    int `json:"totalAchievements"`
	SecretsFound         int `json:"secretsFound"`
	Level                int `json:"level"`
	NextLevelPoints      int `json:"nextLevelPoints"`
}