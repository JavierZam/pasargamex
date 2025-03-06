package repository

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
)

type firestoreTransactionRepository struct {
	client *firestore.Client
}

func NewFirestoreTransactionRepository(client *firestore.Client) repository.TransactionRepository {
	return &firestoreTransactionRepository{
		client: client,
	}
}

func (r *firestoreTransactionRepository) Create(ctx context.Context, transaction *entity.Transaction) error {
	// Generate ID if not provided
	if transaction.ID == "" {
		transaction.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	transaction.CreatedAt = now
	transaction.UpdatedAt = now

	// Save to Firestore
	_, err := r.client.Collection("transactions").Doc(transaction.ID).Set(ctx, transaction)
	if err != nil {
		return errors.Internal("Failed to create transaction", err)
	}

	return nil
}

func (r *firestoreTransactionRepository) GetByID(ctx context.Context, id string) (*entity.Transaction, error) {
	doc, err := r.client.Collection("transactions").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, errors.NotFound("Transaction", err)
		}
		return nil, errors.Internal("Failed to get transaction", err)
	}

	var transaction entity.Transaction
	if err := doc.DataTo(&transaction); err != nil {
		return nil, errors.Internal("Failed to parse transaction data", err)
	}

	return &transaction, nil
}

func (r *firestoreTransactionRepository) Update(ctx context.Context, transaction *entity.Transaction) error {
	transaction.UpdatedAt = time.Now()

	_, err := r.client.Collection("transactions").Doc(transaction.ID).Set(ctx, transaction)
	if err != nil {
		return errors.Internal("Failed to update transaction", err)
	}

	return nil
}

func (r *firestoreTransactionRepository) List(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]*entity.Transaction, int64, error) {
	collection := r.client.Collection("transactions")
	query := collection.OrderBy("createdAt", firestore.Desc)

	// Apply filters
	for key, value := range filter {
		query = query.Where(key, "==", value)
	}

	// Get total count
	countQuery := query
	countDocs, err := countQuery.Documents(ctx).GetAll()
	if err != nil {
		return nil, 0, errors.Internal("Failed to count transactions", err)
	}
	total := int64(len(countDocs))

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// Execute query
	iter := query.Documents(ctx)
	var transactions []*entity.Transaction

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, errors.Internal("Failed to iterate transactions", err)
		}

		var transaction entity.Transaction
		if err := doc.DataTo(&transaction); err != nil {
			return nil, 0, errors.Internal("Failed to parse transaction data", err)
		}
		transactions = append(transactions, &transaction)
	}

	return transactions, total, nil
}

func (r *firestoreTransactionRepository) CreateLog(ctx context.Context, log *entity.TransactionLog) error {
	// Generate ID if not provided
	if log.ID == "" {
		log.ID = uuid.New().String()
	}

	// Set timestamp
	log.CreatedAt = time.Now()

	// Save to Firestore
	_, err := r.client.Collection("transaction_logs").Doc(log.ID).Set(ctx, log)
	if err != nil {
		return errors.Internal("Failed to create transaction log", err)
	}

	return nil
}

func (r *firestoreTransactionRepository) ListLogsByTransactionID(ctx context.Context, transactionID string) ([]*entity.TransactionLog, error) {
	query := r.client.Collection("transaction_logs").
		Where("transactionId", "==", transactionID).
		OrderBy("createdAt", firestore.Asc)

	iter := query.Documents(ctx)
	var logs []*entity.TransactionLog

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, errors.Internal("Failed to iterate transaction logs", err)
		}

		var log entity.TransactionLog
		if err := doc.DataTo(&log); err != nil {
			return nil, errors.Internal("Failed to parse transaction log data", err)
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

func (r *firestoreTransactionRepository) ListByUserID(ctx context.Context, userID string, role string, status string, limit, offset int) ([]*entity.Transaction, int64, error) {
	// Determine the field to query based on role
	var field string
	if role == "buyer" {
		field = "buyerId"
	} else if role == "seller" {
		field = "sellerId"
	} else {
		return nil, 0, errors.BadRequest("Invalid role", nil)
	}

	query := r.client.Collection("transactions").Where(field, "==", userID)

	// Add status filter if provided
	if status != "" {
		query = query.Where("status", "==", status)
	}

	// Order by created date, most recent first
	query = query.OrderBy("createdAt", firestore.Desc)

	// Get total count
	countDocs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, 0, errors.Internal("Failed to count transactions", err)
	}
	total := int64(len(countDocs))

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// Execute query
	iter := query.Documents(ctx)
	var transactions []*entity.Transaction

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, errors.Internal("Failed to iterate transactions", err)
		}

		var transaction entity.Transaction
		if err := doc.DataTo(&transaction); err != nil {
			return nil, 0, errors.Internal("Failed to parse transaction data", err)
		}
		transactions = append(transactions, &transaction)
	}

	return transactions, total, nil
}

func (r *firestoreTransactionRepository) ListPendingMiddlemanTransactions(ctx context.Context, limit, offset int) ([]*entity.Transaction, int64, error) {
	query := r.client.Collection("transactions").
		Where("deliveryMethod", "==", "middleman").
		Where("status", "==", "processing").
		Where("middlemanStatus", "==", "pending_assignment").
		OrderBy("createdAt", firestore.Asc)

	// Get total count
	countDocs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, 0, errors.Internal("Failed to count pending middleman transactions", err)
	}
	total := int64(len(countDocs))

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// Execute query
	iter := query.Documents(ctx)
	var transactions []*entity.Transaction

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, errors.Internal("Failed to iterate transactions", err)
		}

		var transaction entity.Transaction
		if err := doc.DataTo(&transaction); err != nil {
			return nil, 0, errors.Internal("Failed to parse transaction data", err)
		}
		transactions = append(transactions, &transaction)
	}

	return transactions, total, nil
}

func (r *firestoreTransactionRepository) GetTransactionStats(ctx context.Context, userID string, period string) (map[string]interface{}, error) {
	// Implementation depends on requirements
	// This is just a placeholder
	log.Printf("GetTransactionStats called for user %s with period %s", userID, period)
	
	return map[string]interface{}{
		"totalTransactions": 0,
		"totalSales": 0.0,
		"totalPurchases": 0.0,
		"completedTransactions": 0,
		"pendingTransactions": 0,
	}, nil
}

func (r *firestoreTransactionRepository) HasCompletedTransaction(ctx context.Context, userID, productID string) (bool, error) {
    log.Printf("Checking if user %s has completed transaction for product %s", userID, productID)
    
    query := r.client.Collection("transactions").
        Where("buyerId", "==", userID).
        Where("productId", "==", productID).
        Where("status", "==", "completed").
        Where("paymentStatus", "==", "paid").
        Limit(1)

    iter := query.Documents(ctx)
    doc, err := iter.Next()
    
    if err != nil {
        if err == iterator.Done {
            log.Printf("No completed transaction found")
            return false, nil
        }
        log.Printf("Error checking completed transaction: %v", err)
        return false, err
    }

    log.Printf("Completed transaction found: %v", doc.Ref.ID)
    return true, nil
}