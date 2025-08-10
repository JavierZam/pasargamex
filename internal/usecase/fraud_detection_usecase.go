package usecase

import (
	"context"
	"log"
	"strings"
	"time"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
)

type FraudDetectionUseCase struct {
	transactionRepo repository.TransactionRepository
	userRepo        repository.UserRepository
}

func NewFraudDetectionUseCase(
	transactionRepo repository.TransactionRepository,
	userRepo repository.UserRepository,
) *FraudDetectionUseCase {
	return &FraudDetectionUseCase{
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
	}
}

type FraudAnalysisResult struct {
	Score      float64  `json:"score"`
	RiskLevel  string   `json:"risk_level"`  // low, medium, high, critical
	Flags      []string `json:"flags"`
	Reasons    []string `json:"reasons"`
	Action     string   `json:"action"`      // allow, review, block
	ReviewBy   string   `json:"review_by"`   // auto, human, admin
}

func (uc *FraudDetectionUseCase) AnalyzeTransaction(ctx context.Context, transaction *entity.Transaction, buyer *entity.User, seller *entity.User, product *entity.Product) (*FraudAnalysisResult, error) {
	result := &FraudAnalysisResult{
		Score:     0.0,
		RiskLevel: "low",
		Flags:     []string{},
		Reasons:   []string{},
		Action:    "allow",
		ReviewBy:  "auto",
	}

	// 1. NEW USER RISK
	if time.Since(buyer.CreatedAt) < 24*time.Hour {
		result.Score += 0.2
		result.Flags = append(result.Flags, "new_buyer")
		result.Reasons = append(result.Reasons, "Buyer account created less than 24 hours ago")
	}

	if time.Since(seller.CreatedAt) < 7*24*time.Hour {
		result.Score += 0.15
		result.Flags = append(result.Flags, "new_seller")
		result.Reasons = append(result.Reasons, "Seller account created less than 7 days ago")
	}

	// 2. HIGH VALUE TRANSACTION
	if transaction.TotalAmount > 1000000 { // > 1M IDR
		result.Score += 0.3
		result.Flags = append(result.Flags, "high_value")
		result.Reasons = append(result.Reasons, "High value transaction")
	}

	// 3. RAPID TRANSACTIONS (Rate limiting check)
	buyerRecentCount, err := uc.getUserRecentTransactionCount(ctx, buyer.ID, 1*time.Hour)
	if err == nil && buyerRecentCount > 5 {
		result.Score += 0.4
		result.Flags = append(result.Flags, "rapid_transactions")
		result.Reasons = append(result.Reasons, "Multiple transactions in short time")
	}

	// 4. SELLER REPUTATION
	if seller.SellerReviewCount < 5 {
		result.Score += 0.1
		result.Flags = append(result.Flags, "low_seller_reviews")
		result.Reasons = append(result.Reasons, "Seller has few reviews")
	}

	if seller.SellerRating < 4.0 {
		result.Score += 0.2
		result.Flags = append(result.Flags, "low_seller_rating")
		result.Reasons = append(result.Reasons, "Seller has low rating")
	}

	// 5. PRODUCT RISK FACTORS
	gameTitle := strings.ToLower(product.Title)
	if strings.Contains(gameTitle, "valorant") || strings.Contains(gameTitle, "csgo") || strings.Contains(gameTitle, "pubg") {
		result.Score += 0.1
		result.Flags = append(result.Flags, "high_risk_game")
		result.Reasons = append(result.Reasons, "High-risk game category")
	}

	// 6. ACCOUNT VERIFICATION
	// Note: User entity doesn't have IsEmailVerified field, using Status instead
	if buyer.Status != "active" {
		result.Score += 0.15
		result.Flags = append(result.Flags, "inactive_buyer")
		result.Reasons = append(result.Reasons, "Buyer account not active")
	}

	if seller.VerificationStatus != "verified" {
		result.Score += 0.25
		result.Flags = append(result.Flags, "unverified_seller")
		result.Reasons = append(result.Reasons, "Seller not fully verified")
	}

	// 7. CALCULATE RISK LEVEL AND ACTION
	uc.calculateRiskLevelAndAction(result)

	log.Printf("Fraud analysis for transaction %s: Score=%.2f, Risk=%s, Action=%s", 
		transaction.ID, result.Score, result.RiskLevel, result.Action)

	return result, nil
}

func (uc *FraudDetectionUseCase) calculateRiskLevelAndAction(result *FraudAnalysisResult) {
	if result.Score >= 0.8 {
		result.RiskLevel = "critical"
		result.Action = "block"
		result.ReviewBy = "admin"
	} else if result.Score >= 0.6 {
		result.RiskLevel = "high"
		result.Action = "review"
		result.ReviewBy = "human"
	} else if result.Score >= 0.4 {
		result.RiskLevel = "medium"
		result.Action = "review"
		result.ReviewBy = "auto"
	} else if result.Score >= 0.2 {
		result.RiskLevel = "low"
		result.Action = "allow"
		result.ReviewBy = "auto"
	} else {
		result.RiskLevel = "minimal"
		result.Action = "allow"
		result.ReviewBy = "auto"
	}
}

func (uc *FraudDetectionUseCase) getUserRecentTransactionCount(ctx context.Context, userID string, duration time.Duration) (int, error) {
	// This would query transactions created within the duration
	// Implementation would depend on your transaction repository
	// For now, return 0 to avoid compilation error
	return 0, nil
}

// Post-transaction monitoring
func (uc *FraudDetectionUseCase) MonitorCredentialAccess(ctx context.Context, transactionID string, accessIP string) error {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return err
	}

	now := time.Now()
	transaction.CredentialsAccessed = true
	transaction.CredentialsAccessedAt = &now

	// Check for suspicious access patterns
	if transaction.CredentialsDeliveredAt != nil {
		accessDelay := now.Sub(*transaction.CredentialsDeliveredAt)
		if accessDelay > 24*time.Hour {
			transaction.SecurityFlags = append(transaction.SecurityFlags, "delayed_credential_access")
			log.Printf("SECURITY ALERT: Delayed credential access for transaction %s", transactionID)
		}
	}

	return uc.transactionRepo.Update(ctx, transaction)
}

// Detect account recovery fraud
func (uc *FraudDetectionUseCase) DetectAccountRecovery(ctx context.Context, transactionID string, reportedBy string) error {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return err
	}

	// Mark transaction as potentially fraudulent
	transaction.SecurityFlags = append(transaction.SecurityFlags, "account_recovery_reported")
	transaction.FraudScore = 0.9 // High fraud score

	log.Printf("FRAUD ALERT: Account recovery reported for transaction %s by %s", transactionID, reportedBy)

	// Automatically create dispute
	// Implementation would create dispute entity

	return uc.transactionRepo.Update(ctx, transaction)
}