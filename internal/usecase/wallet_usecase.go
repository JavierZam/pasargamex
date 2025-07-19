package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
	"pasargamex/pkg/utils"
)

type WalletUseCase struct {
	walletRepo          repository.WalletRepository
	walletTxnRepo       repository.WalletTransactionRepository
	paymentMethodRepo   repository.PaymentMethodRepository
	topupRepo           repository.TopupRepository
	withdrawRepo        repository.WithdrawRepository
	userRepo            repository.UserRepository
}

func NewWalletUseCase(
	walletRepo repository.WalletRepository,
	walletTxnRepo repository.WalletTransactionRepository,
	paymentMethodRepo repository.PaymentMethodRepository,
	topupRepo repository.TopupRepository,
	withdrawRepo repository.WithdrawRepository,
	userRepo repository.UserRepository,
) *WalletUseCase {
	return &WalletUseCase{
		walletRepo:        walletRepo,
		walletTxnRepo:     walletTxnRepo,
		paymentMethodRepo: paymentMethodRepo,
		topupRepo:         topupRepo,
		withdrawRepo:      withdrawRepo,
		userRepo:          userRepo,
	}
}

type CreateWalletInput struct {
	UserID   string
	Currency string
}

type TopupWalletInput struct {
	Amount          float64
	PaymentMethodID string
}

type WithdrawWalletInput struct {
	Amount          float64
	PaymentMethodID string
}

type CreatePaymentMethodInput struct {
	Type          string
	Provider      string
	AccountNumber string
	AccountName   string
	IsDefault     bool
	Details       map[string]interface{}
}

type UpdatePaymentMethodInput struct {
	AccountNumber string
	AccountName   string
	IsDefault     bool
	Details       map[string]interface{}
}

// Wallet Management
func (uc *WalletUseCase) CreateWallet(ctx context.Context, input CreateWalletInput) (*entity.Wallet, error) {
	// Check if user exists
	_, err := uc.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		return nil, errors.NotFound("User", err)
	}

	// Check if wallet already exists
	existingWallet, err := uc.walletRepo.GetWalletByUserID(ctx, input.UserID)
	if err == nil && existingWallet != nil {
		return nil, errors.Conflict("Wallet already exists for user")
	}

	currency := input.Currency
	if currency == "" {
		currency = "IDR"
	}

	wallet := &entity.Wallet{
		ID:        uuid.New().String(),
		UserID:    input.UserID,
		Balance:   0,
		Currency:  currency,
		Status:    "active",
		LastTxnAt: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = uc.walletRepo.CreateWallet(ctx, wallet)
	if err != nil {
		return nil, errors.InternalServer("Failed to create wallet", err)
	}

	return wallet, nil
}

func (uc *WalletUseCase) GetWalletByUserID(ctx context.Context, userID string) (*entity.Wallet, error) {
	wallet, err := uc.walletRepo.GetWalletByUserID(ctx, userID)
	if err != nil {
		return nil, errors.NotFound("Wallet", err)
	}

	return wallet, nil
}

func (uc *WalletUseCase) GetWalletTransactions(ctx context.Context, userID string, pagination *utils.Pagination) ([]entity.WalletTransaction, error) {
	// Verify user owns the wallet
	_, err := uc.walletRepo.GetWalletByUserID(ctx, userID)
	if err != nil {
		return nil, errors.NotFound("Wallet", err)
	}

	transactions, err := uc.walletTxnRepo.GetTransactionsByUserID(ctx, userID, pagination)
	if err != nil {
		return nil, errors.InternalServer("Failed to get wallet transactions", err)
	}

	return transactions, nil
}

// Payment Methods
func (uc *WalletUseCase) CreatePaymentMethod(ctx context.Context, userID string, input CreatePaymentMethodInput) (*entity.PaymentMethod, error) {
	// Check if user exists
	_, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.NotFound("User", err)
	}

	// If this is set as default, unset other defaults
	if input.IsDefault {
		err = uc.paymentMethodRepo.SetDefaultPaymentMethod(ctx, userID, "")
		if err != nil {
			return nil, errors.InternalServer("Failed to update default payment method", err)
		}
	}

	paymentMethod := &entity.PaymentMethod{
		ID:            uuid.New().String(),
		UserID:        userID,
		Type:          input.Type,
		Provider:      input.Provider,
		AccountNumber: input.AccountNumber,
		AccountName:   input.AccountName,
		IsDefault:     input.IsDefault,
		IsActive:      true,
		Details:       input.Details,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err = uc.paymentMethodRepo.CreatePaymentMethod(ctx, paymentMethod)
	if err != nil {
		return nil, errors.InternalServer("Failed to create payment method", err)
	}

	return paymentMethod, nil
}

func (uc *WalletUseCase) GetPaymentMethods(ctx context.Context, userID string) ([]entity.PaymentMethod, error) {
	paymentMethods, err := uc.paymentMethodRepo.GetPaymentMethodsByUserID(ctx, userID)
	if err != nil {
		return nil, errors.InternalServer("Failed to get payment methods", err)
	}

	return paymentMethods, nil
}

func (uc *WalletUseCase) UpdatePaymentMethod(ctx context.Context, userID string, paymentMethodID string, input UpdatePaymentMethodInput) (*entity.PaymentMethod, error) {
	paymentMethod, err := uc.paymentMethodRepo.GetPaymentMethodByID(ctx, paymentMethodID)
	if err != nil {
		return nil, errors.NotFound("Payment method", err)
	}

	if paymentMethod.UserID != userID {
		return nil, errors.Forbidden("Access denied", nil)
	}

	if input.AccountNumber != "" {
		paymentMethod.AccountNumber = input.AccountNumber
	}
	if input.AccountName != "" {
		paymentMethod.AccountName = input.AccountName
	}
	if input.Details != nil {
		paymentMethod.Details = input.Details
	}

	// If this is set as default, unset other defaults
	if input.IsDefault && !paymentMethod.IsDefault {
		err = uc.paymentMethodRepo.SetDefaultPaymentMethod(ctx, userID, paymentMethodID)
		if err != nil {
			return nil, errors.InternalServer("Failed to update default payment method", err)
		}
		paymentMethod.IsDefault = true
	}

	err = uc.paymentMethodRepo.UpdatePaymentMethod(ctx, paymentMethod)
	if err != nil {
		return nil, errors.InternalServer("Failed to update payment method", err)
	}

	return paymentMethod, nil
}

func (uc *WalletUseCase) DeletePaymentMethod(ctx context.Context, userID string, paymentMethodID string) error {
	paymentMethod, err := uc.paymentMethodRepo.GetPaymentMethodByID(ctx, paymentMethodID)
	if err != nil {
		return errors.NotFound("Payment method", err)
	}

	if paymentMethod.UserID != userID {
		return errors.Forbidden("Access denied", nil)
	}

	err = uc.paymentMethodRepo.DeletePaymentMethod(ctx, paymentMethodID)
	if err != nil {
		return errors.InternalServer("Failed to delete payment method", err)
	}

	return nil
}

// Topup
func (uc *WalletUseCase) CreateTopupRequest(ctx context.Context, userID string, input TopupWalletInput) (*entity.TopupRequest, error) {
	// Validate amount
	if input.Amount <= 0 {
		return nil, errors.BadRequest("Amount must be greater than 0", nil)
	}

	// Get user wallet
	wallet, err := uc.walletRepo.GetWalletByUserID(ctx, userID)
	if err != nil {
		return nil, errors.NotFound("Wallet", err)
	}

	// Validate payment method
	paymentMethod, err := uc.paymentMethodRepo.GetPaymentMethodByID(ctx, input.PaymentMethodID)
	if err != nil {
		return nil, errors.NotFound("Payment method", err)
	}

	if paymentMethod.UserID != userID {
		return nil, errors.Forbidden("Access denied", nil)
	}

	topupRequest := &entity.TopupRequest{
		ID:              uuid.New().String(),
		UserID:          userID,
		WalletID:        wallet.ID,
		Amount:          input.Amount,
		PaymentMethodID: input.PaymentMethodID,
		Status:          "pending",
		ExpiresAt:       time.Now().Add(24 * time.Hour), // 24 hours expiration
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err = uc.topupRepo.CreateTopupRequest(ctx, topupRequest)
	if err != nil {
		return nil, errors.InternalServer("Failed to create topup request", err)
	}

	return topupRequest, nil
}

func (uc *WalletUseCase) GetTopupRequests(ctx context.Context, userID string, pagination *utils.Pagination) ([]entity.TopupRequest, error) {
	topups, err := uc.topupRepo.GetTopupRequestsByUserID(ctx, userID, pagination)
	if err != nil {
		return nil, errors.InternalServer("Failed to get topup requests", err)
	}

	return topups, nil
}

func (uc *WalletUseCase) ProcessTopupRequest(ctx context.Context, topupID string, adminID string, approve bool, notes string) (*entity.TopupRequest, error) {
	topupRequest, err := uc.topupRepo.GetTopupRequestByID(ctx, topupID)
	if err != nil {
		return nil, errors.NotFound("Topup request", err)
	}

	if topupRequest.Status != "pending" {
		return nil, errors.BadRequest("Topup request already processed", nil)
	}

	if approve {
		// Process the topup
		wallet, err := uc.walletRepo.GetWalletByID(ctx, topupRequest.WalletID)
		if err != nil {
			return nil, errors.NotFound("Wallet", err)
		}

		// Create wallet transaction
		walletTransaction := &entity.WalletTransaction{
			ID:              uuid.New().String(),
			WalletID:        wallet.ID,
			UserID:          topupRequest.UserID,
			Type:            "topup",
			Amount:          topupRequest.Amount,
			PreviousBalance: wallet.Balance,
			NewBalance:      wallet.Balance + topupRequest.Amount,
			Status:          "completed",
			Reference:       topupRequest.ID,
			Description:     fmt.Sprintf("Topup via %s", topupRequest.PaymentMethodID),
			ProcessedAt:     &time.Time{},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		*walletTransaction.ProcessedAt = time.Now()

		err = uc.walletTxnRepo.CreateTransaction(ctx, walletTransaction)
		if err != nil {
			return nil, errors.InternalServer("Failed to create wallet transaction", err)
		}

		// Update wallet balance
		_, err = uc.walletRepo.UpdateWalletBalance(ctx, wallet.ID, topupRequest.Amount)
		if err != nil {
			return nil, errors.InternalServer("Failed to update wallet balance", err)
		}

		topupRequest.Status = "completed"
	} else {
		topupRequest.Status = "failed"
	}

	topupRequest.ProcessedBy = adminID
	topupRequest.AdminNotes = notes
	now := time.Now()
	topupRequest.ProcessedAt = &now

	err = uc.topupRepo.UpdateTopupRequest(ctx, topupRequest)
	if err != nil {
		return nil, errors.InternalServer("Failed to update topup request", err)
	}

	return topupRequest, nil
}

// Withdraw
func (uc *WalletUseCase) CreateWithdrawRequest(ctx context.Context, userID string, input WithdrawWalletInput) (*entity.WithdrawRequest, error) {
	// Validate amount
	if input.Amount <= 0 {
		return nil, errors.BadRequest("Amount must be greater than 0", nil)
	}

	// Get user wallet
	wallet, err := uc.walletRepo.GetWalletByUserID(ctx, userID)
	if err != nil {
		return nil, errors.NotFound("Wallet", err)
	}

	// Check balance
	if wallet.Balance < input.Amount {
		return nil, errors.BadRequest("Insufficient balance", nil)
	}

	// Validate payment method
	paymentMethod, err := uc.paymentMethodRepo.GetPaymentMethodByID(ctx, input.PaymentMethodID)
	if err != nil {
		return nil, errors.NotFound("Payment method", err)
	}

	if paymentMethod.UserID != userID {
		return nil, errors.Forbidden("Access denied", nil)
	}

	// Calculate fee (example: 1% fee)
	fee := input.Amount * 0.01
	netAmount := input.Amount - fee

	withdrawRequest := &entity.WithdrawRequest{
		ID:              uuid.New().String(),
		UserID:          userID,
		WalletID:        wallet.ID,
		Amount:          input.Amount,
		Fee:             fee,
		NetAmount:       netAmount,
		PaymentMethodID: input.PaymentMethodID,
		Status:          "pending",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err = uc.withdrawRepo.CreateWithdrawRequest(ctx, withdrawRequest)
	if err != nil {
		return nil, errors.InternalServer("Failed to create withdraw request", err)
	}

	return withdrawRequest, nil
}

func (uc *WalletUseCase) GetWithdrawRequests(ctx context.Context, userID string, pagination *utils.Pagination) ([]entity.WithdrawRequest, error) {
	withdraws, err := uc.withdrawRepo.GetWithdrawRequestsByUserID(ctx, userID, pagination)
	if err != nil {
		return nil, errors.InternalServer("Failed to get withdraw requests", err)
	}

	return withdraws, nil
}

func (uc *WalletUseCase) ProcessWithdrawRequest(ctx context.Context, withdrawID string, adminID string, approve bool, notes string) (*entity.WithdrawRequest, error) {
	withdrawRequest, err := uc.withdrawRepo.GetWithdrawRequestByID(ctx, withdrawID)
	if err != nil {
		return nil, errors.NotFound("Withdraw request", err)
	}

	if withdrawRequest.Status != "pending" && withdrawRequest.Status != "processing" {
		return nil, errors.BadRequest("Withdraw request already processed", nil)
	}

	if approve {
		// Process the withdrawal
		wallet, err := uc.walletRepo.GetWalletByID(ctx, withdrawRequest.WalletID)
		if err != nil {
			return nil, errors.NotFound("Wallet", err)
		}

		// Check balance again
		if wallet.Balance < withdrawRequest.Amount {
			return nil, errors.BadRequest("Insufficient balance", nil)
		}

		// Create wallet transaction
		walletTransaction := &entity.WalletTransaction{
			ID:              uuid.New().String(),
			WalletID:        wallet.ID,
			UserID:          withdrawRequest.UserID,
			Type:            "withdraw",
			Amount:          -withdrawRequest.Amount,
			PreviousBalance: wallet.Balance,
			NewBalance:      wallet.Balance - withdrawRequest.Amount,
			Status:          "completed",
			Reference:       withdrawRequest.ID,
			Description:     fmt.Sprintf("Withdrawal to %s", withdrawRequest.PaymentMethodID),
			ProcessedAt:     &time.Time{},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		*walletTransaction.ProcessedAt = time.Now()

		err = uc.walletTxnRepo.CreateTransaction(ctx, walletTransaction)
		if err != nil {
			return nil, errors.InternalServer("Failed to create wallet transaction", err)
		}

		// Update wallet balance
		_, err = uc.walletRepo.UpdateWalletBalance(ctx, wallet.ID, -withdrawRequest.Amount)
		if err != nil {
			return nil, errors.InternalServer("Failed to update wallet balance", err)
		}

		withdrawRequest.Status = "completed"
	} else {
		withdrawRequest.Status = "rejected"
	}

	withdrawRequest.ProcessedBy = adminID
	withdrawRequest.AdminNotes = notes
	now := time.Now()
	withdrawRequest.ProcessedAt = &now

	err = uc.withdrawRepo.UpdateWithdrawRequest(ctx, withdrawRequest)
	if err != nil {
		return nil, errors.InternalServer("Failed to update withdraw request", err)
	}

	return withdrawRequest, nil
}

// Wallet Payment (for transactions)
func (uc *WalletUseCase) ProcessWalletPayment(ctx context.Context, userID string, amount float64, description string, reference string) (*entity.WalletTransaction, error) {
	// Get user wallet
	wallet, err := uc.walletRepo.GetWalletByUserID(ctx, userID)
	if err != nil {
		return nil, errors.NotFound("Wallet", err)
	}

	// Check balance
	if wallet.Balance < amount {
		return nil, errors.BadRequest("Insufficient balance", nil)
	}

	// Create wallet transaction
	walletTransaction := &entity.WalletTransaction{
		ID:              uuid.New().String(),
		WalletID:        wallet.ID,
		UserID:          userID,
		Type:            "payment",
		Amount:          -amount,
		PreviousBalance: wallet.Balance,
		NewBalance:      wallet.Balance - amount,
		Status:          "completed",
		Reference:       reference,
		Description:     description,
		ProcessedAt:     &time.Time{},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	*walletTransaction.ProcessedAt = time.Now()

	err = uc.walletTxnRepo.CreateTransaction(ctx, walletTransaction)
	if err != nil {
		return nil, errors.InternalServer("Failed to create wallet transaction", err)
	}

	// Update wallet balance
	_, err = uc.walletRepo.UpdateWalletBalance(ctx, wallet.ID, -amount)
	if err != nil {
		return nil, errors.InternalServer("Failed to update wallet balance", err)
	}

	return walletTransaction, nil
}

// Wallet Refund (for failed transactions)
func (uc *WalletUseCase) ProcessWalletRefund(ctx context.Context, userID string, amount float64, description string, reference string) (*entity.WalletTransaction, error) {
	// Get user wallet
	wallet, err := uc.walletRepo.GetWalletByUserID(ctx, userID)
	if err != nil {
		return nil, errors.NotFound("Wallet", err)
	}

	// Create wallet transaction
	walletTransaction := &entity.WalletTransaction{
		ID:              uuid.New().String(),
		WalletID:        wallet.ID,
		UserID:          userID,
		Type:            "refund",
		Amount:          amount,
		PreviousBalance: wallet.Balance,
		NewBalance:      wallet.Balance + amount,
		Status:          "completed",
		Reference:       reference,
		Description:     description,
		ProcessedAt:     &time.Time{},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	*walletTransaction.ProcessedAt = time.Now()

	err = uc.walletTxnRepo.CreateTransaction(ctx, walletTransaction)
	if err != nil {
		return nil, errors.InternalServer("Failed to create wallet transaction", err)
	}

	// Update wallet balance
	_, err = uc.walletRepo.UpdateWalletBalance(ctx, wallet.ID, amount)
	if err != nil {
		return nil, errors.InternalServer("Failed to update wallet balance", err)
	}

	return walletTransaction, nil
}

// Statistics
type WalletStatistics struct {
	TotalWallets      int     `json:"total_wallets"`
	TotalBalance      float64 `json:"total_balance"`
	PendingTopups     int     `json:"pending_topups"`
	PendingWithdraws  int     `json:"pending_withdrawals"`
	DailyTransactions int     `json:"daily_transactions"`
	TransactionVolume float64 `json:"transaction_volume"`
}

func (uc *WalletUseCase) GetWalletStatistics(ctx context.Context) (*WalletStatistics, error) {
	stats := &WalletStatistics{}
	
	// Get total wallets count
	totalWallets, err := uc.walletRepo.GetWalletCount(ctx)
	if err != nil {
		stats.TotalWallets = 0
	} else {
		stats.TotalWallets = totalWallets
	}
	
	// Get total balance across all wallets
	totalBalance, err := uc.walletRepo.GetTotalBalance(ctx)
	if err != nil {
		stats.TotalBalance = 0.0
	} else {
		stats.TotalBalance = totalBalance
	}
	
	// Get pending topups
	pendingTopupPagination := &utils.Pagination{Page: 1, Limit: 1000}
	pendingTopups, err := uc.topupRepo.GetPendingTopupRequests(ctx, pendingTopupPagination)
	if err != nil {
		stats.PendingTopups = 0
	} else {
		stats.PendingTopups = len(pendingTopups)
	}
	
	// Get pending withdraws
	pendingWithdrawPagination := &utils.Pagination{Page: 1, Limit: 1000}
	pendingWithdraws, err := uc.withdrawRepo.GetPendingWithdrawRequests(ctx, pendingWithdrawPagination)
	if err != nil {
		stats.PendingWithdraws = 0
	} else {
		stats.PendingWithdraws = len(pendingWithdraws)
	}
	
	// Get daily transaction count
	dailyCount, err := uc.walletTxnRepo.GetDailyTransactionCount(ctx)
	if err != nil {
		stats.DailyTransactions = 0
	} else {
		stats.DailyTransactions = dailyCount
	}
	
	// Get daily transaction volume
	dailyVolume, err := uc.walletTxnRepo.GetDailyTransactionVolume(ctx)
	if err != nil {
		stats.TransactionVolume = 0.0
	} else {
		stats.TransactionVolume = dailyVolume
	}
	
	return stats, nil
}

func (uc *WalletUseCase) GetPendingTopupRequests(ctx context.Context, pagination *utils.Pagination) ([]entity.TopupRequest, error) {
	return uc.topupRepo.GetPendingTopupRequests(ctx, pagination)
}

func (uc *WalletUseCase) GetPendingWithdrawRequests(ctx context.Context, pagination *utils.Pagination) ([]entity.WithdrawRequest, error) {
	return uc.withdrawRepo.GetPendingWithdrawRequests(ctx, pagination)
}