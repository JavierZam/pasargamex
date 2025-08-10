package usecase

import (
	"context"
	"log"
	"time"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
)

type AntiFraudUseCase struct {
	transactionRepo repository.TransactionRepository
	userRepo        repository.UserRepository
}

func NewAntiFraudUseCase(
	transactionRepo repository.TransactionRepository,
	userRepo repository.UserRepository,
) *AntiFraudUseCase {
	return &AntiFraudUseCase{
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
	}
}

// DetectCredentialSharingFraud detects potential credential sharing attacks
func (uc *AntiFraudUseCase) DetectCredentialSharingFraud(ctx context.Context, transactionID string) (*entity.Transaction, error) {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	// RED FLAGS for credential sharing fraud:
	
	// 1. Quick dispute after credentials accessed
	if transaction.CredentialsAccessedAt != nil && transaction.DisputeCreatedAt != nil {
		timeBetween := transaction.DisputeCreatedAt.Sub(*transaction.CredentialsAccessedAt)
		if timeBetween < 1*time.Hour {
			transaction.SecurityFlags = append(transaction.SecurityFlags, "quick_dispute_after_access")
			transaction.FraudScore += 0.3
			log.Printf("FRAUD ALERT: Quick dispute after credential access for transaction %s", transactionID)
		}
	}

	// 2. Multiple similar disputes from same user
	// TODO: Implement user dispute pattern analysis

	// 3. Credential accessed but immediately reported as invalid
	if transaction.CredentialsAccessed && transaction.IsDisputed {
		transaction.SecurityFlags = append(transaction.SecurityFlags, "accessed_then_disputed")
		transaction.FraudScore += 0.2
	}

	// 4. Check if buyer has history of disputes after accessing credentials
	buyerDisputePattern, err := uc.analyzeUserDisputePattern(ctx, transaction.BuyerID)
	if err == nil && buyerDisputePattern.HighRisk {
		transaction.SecurityFlags = append(transaction.SecurityFlags, "high_risk_dispute_pattern")
		transaction.FraudScore += 0.4
		log.Printf("FRAUD ALERT: High-risk dispute pattern for user %s", transaction.BuyerID)
	}

	// Update transaction with fraud analysis
	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return nil, err
	}

	return transaction, nil
}

type UserDisputePattern struct {
	HighRisk          bool    `json:"high_risk"`
	DisputeRate       float64 `json:"dispute_rate"`       // % of transactions disputed
	AccessThenDispute int     `json:"access_then_dispute"` // Count of accessed-then-disputed
	RecentDisputes    int     `json:"recent_disputes"`    // Disputes in last 30 days
}

func (uc *AntiFraudUseCase) analyzeUserDisputePattern(ctx context.Context, userID string) (*UserDisputePattern, error) {
	// Get user's transaction history
	transactions, _, err := uc.transactionRepo.ListByUserID(ctx, userID, "buyer", "", 100, 0)
	if err != nil {
		return nil, err
	}

	pattern := &UserDisputePattern{}
	totalTransactions := len(transactions)
	disputedCount := 0
	accessThenDisputeCount := 0
	recentDisputeCount := 0
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	for _, tx := range transactions {
		if tx.IsDisputed {
			disputedCount++
			
			// Check if disputed after accessing credentials
			if tx.CredentialsAccessed {
				accessThenDisputeCount++
			}
			
			// Check recent disputes
			if tx.DisputeCreatedAt != nil && tx.DisputeCreatedAt.After(thirtyDaysAgo) {
				recentDisputeCount++
			}
		}
	}

	if totalTransactions > 0 {
		pattern.DisputeRate = float64(disputedCount) / float64(totalTransactions)
	}
	pattern.AccessThenDispute = accessThenDisputeCount
	pattern.RecentDisputes = recentDisputeCount

	// Determine if high risk
	pattern.HighRisk = pattern.DisputeRate > 0.3 || // >30% dispute rate
		accessThenDisputeCount > 2 || // >2 access-then-dispute cases
		recentDisputeCount > 3 // >3 recent disputes

	return pattern, nil
}

// ImplementCredentialAccessLogging logs all credential access with metadata
func (uc *AntiFraudUseCase) LogCredentialAccess(ctx context.Context, transactionID, userID, ipAddress, userAgent string) error {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return err
	}

	now := time.Now()
	transaction.CredentialsAccessed = true
	transaction.CredentialsAccessedAt = &now

	// SECURITY: Log credential access for audit
	log.Printf("CREDENTIAL ACCESS: TransactionID=%s, UserID=%s, IP=%s, UserAgent=%s, Time=%s", 
		transactionID, userID, ipAddress, userAgent, now.Format(time.RFC3339))

	// Check for suspicious access patterns
	if transaction.CredentialsDeliveredAt != nil {
		accessDelay := now.Sub(*transaction.CredentialsDeliveredAt)
		if accessDelay > 24*time.Hour {
			transaction.SecurityFlags = append(transaction.SecurityFlags, "delayed_credential_access")
			log.Printf("SECURITY ALERT: Delayed credential access (%v) for transaction %s", accessDelay, transactionID)
		}
		
		if accessDelay < 1*time.Minute {
			transaction.SecurityFlags = append(transaction.SecurityFlags, "immediate_credential_access")
			// This could be normal, but worth tracking
		}
	}

	// TODO: Store access logs in separate audit table for legal compliance

	return uc.transactionRepo.Update(ctx, transaction)
}

// PreventRefundAbuse implements checks before processing refunds
func (uc *AntiFraudUseCase) ValidateRefundRequest(ctx context.Context, transactionID string, reason string) error {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return err
	}

	// CRITICAL: Don't allow refunds for accessed credentials without strong evidence
	if transaction.CredentialsAccessed && reason == "credential_invalid" {
		
		// Check if credentials were accessed recently (potential sharing)
		if transaction.CredentialsAccessedAt != nil {
			accessAge := time.Since(*transaction.CredentialsAccessedAt)
			if accessAge < 1*time.Hour {
				log.Printf("FRAUD ALERT: Refund request for recently accessed credentials - TransactionID: %s", transactionID)
				return errors.BadRequest("Refund cannot be processed immediately after credential access. Please wait 24 hours or provide additional evidence.", nil)
			}
		}

		// Check user's dispute pattern
		pattern, err := uc.analyzeUserDisputePattern(ctx, transaction.BuyerID)
		if err == nil && pattern.HighRisk {
			log.Printf("FRAUD ALERT: High-risk user requesting refund for accessed credentials - UserID: %s", transaction.BuyerID)
			return errors.BadRequest("Your refund request requires manual review due to account history. Please contact support.", nil)
		}
	}

	return nil
}