package repository

import (
	"context"
	"pasargamex/internal/domain/entity"
)

type TransactionRepository interface {
	Create(ctx context.Context, transaction *entity.Transaction) error
	GetByID(ctx context.Context, id string) (*entity.Transaction, error)
	Update(ctx context.Context, transaction *entity.Transaction) error
	List(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]*entity.Transaction, int64, error)

	CreateLog(ctx context.Context, log *entity.TransactionLog) error
	ListLogsByTransactionID(ctx context.Context, transactionID string) ([]*entity.TransactionLog, error)

	ListByUserID(ctx context.Context, userID string, role string, status string, limit, offset int) ([]*entity.Transaction, int64, error)
	ListPendingMiddlemanTransactions(ctx context.Context, limit, offset int) ([]*entity.Transaction, int64, error)

	GetTransactionStats(ctx context.Context, userID string, period string) (map[string]interface{}, error)
	HasCompletedTransaction(ctx context.Context, userID, productID string) (bool, error)
	
	// Midtrans Integration Methods
	GetByMidtransOrderID(ctx context.Context, midtransOrderID string) (*entity.Transaction, error)
	
	// Approval System Methods
	CreateApproval(ctx context.Context, approval *entity.TransactionApproval) error
	GetApprovalsByTransactionID(ctx context.Context, transactionID string) ([]*entity.TransactionApproval, error)
	UpdateApproval(ctx context.Context, approval *entity.TransactionApproval) error
}
