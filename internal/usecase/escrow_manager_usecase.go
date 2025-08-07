package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
)

type EscrowManagerUseCase struct {
	transactionRepo repository.TransactionRepository
	walletUseCase   *WalletUseCase
	chatUseCase     *ChatUseCase
}

func NewEscrowManagerUseCase(
	transactionRepo repository.TransactionRepository,
	walletUseCase *WalletUseCase,
	chatUseCase *ChatUseCase,
) *EscrowManagerUseCase {
	return &EscrowManagerUseCase{
		transactionRepo: transactionRepo,
		walletUseCase:   walletUseCase,
		chatUseCase:     chatUseCase,
	}
}

// DeliverCredentials - Seller delivers account credentials
func (uc *EscrowManagerUseCase) DeliverCredentials(ctx context.Context, transactionID, sellerID string, credentials map[string]interface{}) error {
	log.Printf("Delivering credentials for transaction: %s", transactionID)

	// Get transaction
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return err
	}

	// Validate seller
	if transaction.SellerID != sellerID {
		return errors.Forbidden("Only the seller can deliver credentials", nil)
	}

	// Validate transaction state
	if transaction.PaymentStatus != "paid" {
		return errors.BadRequest("Payment must be completed before delivering credentials", nil)
	}

	if transaction.CredentialsDelivered {
		return errors.BadRequest("Credentials already delivered", nil)
	}

	// Update transaction with credentials
	now := time.Now()
	transaction.Credentials = credentials
	transaction.CredentialsDelivered = true
	transaction.CredentialsDeliveredAt = &now
	transaction.Status = "credentials_delivered"
	
	// Set auto-release timer (24 hours from delivery)
	autoReleaseTime := now.Add(24 * time.Hour)
	transaction.AutoReleaseAt = &autoReleaseTime
	
	transaction.UpdatedAt = now

	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return errors.Internal("Failed to update transaction", err)
	}

	// Send notification to buyer via chat
	if transaction.MiddlemanChatID != "" {
		uc.chatUseCase.SendSystemMessage(ctx, transaction.MiddlemanChatID, 
			"🎮 Seller has delivered the account credentials. Please check and confirm within 24 hours.", 
			"credentials_delivered", 
			map[string]interface{}{
				"transaction_id": transactionID,
				"auto_release_at": autoReleaseTime.Unix(),
			})
	}

	log.Printf("Credentials delivered for transaction: %s", transactionID)
	return nil
}

// ConfirmCredentials - Buyer confirms credentials are working
func (uc *EscrowManagerUseCase) ConfirmCredentials(ctx context.Context, transactionID, buyerID string, isWorking bool, notes string) error {
	log.Printf("Buyer confirming credentials for transaction: %s, working: %v", transactionID, isWorking)

	// Get transaction
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return err
	}

	// Validate buyer
	if transaction.BuyerID != buyerID {
		return errors.Forbidden("Only the buyer can confirm credentials", nil)
	}

	// Validate transaction state
	if !transaction.CredentialsDelivered {
		return errors.BadRequest("Credentials must be delivered first", nil)
	}

	now := time.Now()

	if isWorking {
		// Credentials work - release funds immediately
		transaction.BuyerConfirmedCredentials = true
		transaction.BuyerConfirmedAt = &now
		transaction.Status = "completed"
		transaction.CompletedAt = &now
		transaction.EscrowStatus = "released"

		// Release funds to seller
		if err := uc.releaseFundsToSeller(ctx, transaction); err != nil {
			log.Printf("Warning: Failed to release funds to seller: %v", err)
		}

		// Notify via chat
		if transaction.MiddlemanChatID != "" {
			uc.chatUseCase.SendSystemMessage(ctx, transaction.MiddlemanChatID, 
				"✅ Buyer confirmed credentials are working. Funds released to seller. Transaction completed!", 
				"transaction_completed", 
				map[string]interface{}{
					"transaction_id": transactionID,
					"completed_at": now.Unix(),
				})
		}

	} else {
		// Credentials don't work - dispute
		transaction.Status = "disputed"
		transaction.Notes = fmt.Sprintf("Buyer dispute: %s", notes)

		// Notify admin/middleman
		if transaction.MiddlemanChatID != "" {
			uc.chatUseCase.SendSystemMessage(ctx, transaction.MiddlemanChatID, 
				"⚠️ Buyer reported credentials not working. Admin review required.", 
				"credentials_disputed", 
				map[string]interface{}{
					"transaction_id": transactionID,
					"dispute_reason": notes,
				})
		}
	}

	transaction.UpdatedAt = now

	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return errors.Internal("Failed to update transaction", err)
	}

	log.Printf("Credentials confirmation processed for transaction: %s", transactionID)
	return nil
}

// ProcessAutoRelease - Background job to auto-release funds after timer expires
func (uc *EscrowManagerUseCase) ProcessAutoRelease(ctx context.Context) error {
	log.Printf("Processing auto-release for expired transactions")

	// Get transactions ready for auto-release
	filter := map[string]interface{}{
		"status":               "credentials_delivered",
		"credentials_delivered": true,
		"buyer_confirmed_credentials": false,
		"escrow_status":        "held",
	}

	transactions, _, err := uc.transactionRepo.List(ctx, filter, 100, 0)
	if err != nil {
		return err
	}

	now := time.Now()
	releasedCount := 0

	for _, transaction := range transactions {
		// Check if auto-release time has passed
		if transaction.AutoReleaseAt != nil && now.After(*transaction.AutoReleaseAt) {
			log.Printf("Auto-releasing funds for transaction: %s", transaction.ID)

			// Release funds
			transaction.Status = "auto_completed"
			transaction.CompletedAt = &now
			transaction.EscrowStatus = "released"
			transaction.UpdatedAt = now

			if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
				log.Printf("Failed to auto-release transaction %s: %v", transaction.ID, err)
				continue
			}

			// Release funds to seller
			if err := uc.releaseFundsToSeller(ctx, transaction); err != nil {
				log.Printf("Warning: Failed to release funds to seller for transaction %s: %v", transaction.ID, err)
			}

			// Notify via chat
			if transaction.MiddlemanChatID != "" {
				uc.chatUseCase.SendSystemMessage(ctx, transaction.MiddlemanChatID, 
					"⏰ Auto-release: 24 hours passed without buyer dispute. Funds released to seller.", 
					"auto_released", 
					map[string]interface{}{
						"transaction_id": transaction.ID,
						"auto_released_at": now.Unix(),
					})
			}

			releasedCount++
		}
	}

	log.Printf("Auto-release processed: %d transactions released", releasedCount)
	return nil
}

// releaseFundsToSeller transfers funds from escrow to seller wallet
func (uc *EscrowManagerUseCase) releaseFundsToSeller(ctx context.Context, transaction *entity.Transaction) error {
	if uc.walletUseCase != nil {
		// TODO: Implement actual wallet transfer
		log.Printf("TODO: Release Rp %.0f to seller %s for transaction %s", 
			transaction.Amount, transaction.SellerID, transaction.ID)
		return nil
	}
	return nil
}

// StartAutoReleaseJob - Start background job for auto-release
func (uc *EscrowManagerUseCase) StartAutoReleaseJob(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute) // Check every 10 minutes
	
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := uc.ProcessAutoRelease(ctx); err != nil {
					log.Printf("Auto-release job error: %v", err)
				}
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
	
	log.Printf("Auto-release job started (checking every 10 minutes)")
}