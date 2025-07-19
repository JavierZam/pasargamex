package repository

import (
	"context"
	"pasargamex/internal/domain/entity"
	"pasargamex/pkg/utils"
)

type WalletRepository interface {
	CreateWallet(ctx context.Context, wallet *entity.Wallet) error
	GetWalletByID(ctx context.Context, walletID string) (*entity.Wallet, error)
	GetWalletByUserID(ctx context.Context, userID string) (*entity.Wallet, error)
	UpdateWallet(ctx context.Context, wallet *entity.Wallet) error
	UpdateWalletBalance(ctx context.Context, walletID string, amount float64) (*entity.Wallet, error)
	GetWalletCount(ctx context.Context) (int, error)
	GetTotalBalance(ctx context.Context) (float64, error)
}

type WalletTransactionRepository interface {
	CreateTransaction(ctx context.Context, transaction *entity.WalletTransaction) error
	GetTransactionByID(ctx context.Context, transactionID string) (*entity.WalletTransaction, error)
	GetTransactionsByWalletID(ctx context.Context, walletID string, pagination *utils.Pagination) ([]entity.WalletTransaction, error)
	GetTransactionsByUserID(ctx context.Context, userID string, pagination *utils.Pagination) ([]entity.WalletTransaction, error)
	UpdateTransaction(ctx context.Context, transaction *entity.WalletTransaction) error
	GetTransactionsByType(ctx context.Context, userID string, txnType string, pagination *utils.Pagination) ([]entity.WalletTransaction, error)
	GetDailyTransactionCount(ctx context.Context) (int, error)
	GetDailyTransactionVolume(ctx context.Context) (float64, error)
}

type PaymentMethodRepository interface {
	CreatePaymentMethod(ctx context.Context, paymentMethod *entity.PaymentMethod) error
	GetPaymentMethodByID(ctx context.Context, paymentMethodID string) (*entity.PaymentMethod, error)
	GetPaymentMethodsByUserID(ctx context.Context, userID string) ([]entity.PaymentMethod, error)
	UpdatePaymentMethod(ctx context.Context, paymentMethod *entity.PaymentMethod) error
	DeletePaymentMethod(ctx context.Context, paymentMethodID string) error
	SetDefaultPaymentMethod(ctx context.Context, userID string, paymentMethodID string) error
}

type TopupRepository interface {
	CreateTopupRequest(ctx context.Context, topup *entity.TopupRequest) error
	GetTopupRequestByID(ctx context.Context, topupID string) (*entity.TopupRequest, error)
	GetTopupRequestsByUserID(ctx context.Context, userID string, pagination *utils.Pagination) ([]entity.TopupRequest, error)
	UpdateTopupRequest(ctx context.Context, topup *entity.TopupRequest) error
	GetPendingTopupRequests(ctx context.Context, pagination *utils.Pagination) ([]entity.TopupRequest, error)
}

type WithdrawRepository interface {
	CreateWithdrawRequest(ctx context.Context, withdraw *entity.WithdrawRequest) error
	GetWithdrawRequestByID(ctx context.Context, withdrawID string) (*entity.WithdrawRequest, error)
	GetWithdrawRequestsByUserID(ctx context.Context, userID string, pagination *utils.Pagination) ([]entity.WithdrawRequest, error)
	UpdateWithdrawRequest(ctx context.Context, withdraw *entity.WithdrawRequest) error
	GetPendingWithdrawRequests(ctx context.Context, pagination *utils.Pagination) ([]entity.WithdrawRequest, error)
}