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
	"pasargamex/pkg/errors"
)

type TransactionRepository interface {
	Create(ctx context.Context, transaction *entity.Transaction) error
	GetByID(ctx context.Context, id string) (*entity.Transaction, error)
	Update(ctx context.Context, transaction *entity.Transaction) error
	List(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]*entity.Transaction, int64, error)

	CreateLog(ctx context.Context, log *entity.TransactionLog) error
	ListLogsByTransactionID(ctx context.Context, transactionID string) ([]*entity.TransactionLog, error)

	ListByUserID(ctx context.Context, userID string, role string, status string, limit, offset int) ([]*entity.Transaction, int64, error)
	ListPendingMiddlemanTransactions(ctx context.Context, limit, offset int) ([]*entity.Transaction, int64, error) // Modified to be more generic

	GetTransactionStats(ctx context.Context, userID string, period string) (map[string]interface{}, error)
	HasCompletedTransaction(ctx context.Context, userID, productID string) (bool, error)
}

type firestoreTransactionRepository struct {
	client *firestore.Client
}

func NewFirestoreTransactionRepository(client *firestore.Client) TransactionRepository {
	return &firestoreTransactionRepository{
		client: client,
	}
}

func (r *firestoreTransactionRepository) Create(ctx context.Context, transaction *entity.Transaction) error {
	if transaction.ID == "" {
		transaction.ID = uuid.New().String()
	}

	now := time.Now()
	transaction.CreatedAt = now
	transaction.UpdatedAt = now

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

	// Apply filters dynamically
	for key, value := range filter {
		if key == "middlemanStatus" {
			// Special handling for middlemanStatus if it's an array of allowed values
			if statusValues, ok := value.([]string); ok && len(statusValues) > 0 {
				// Firestore doesn't support OR queries on a single field directly.
				// This means we cannot do "where middlemanStatus == A OR middlemanStatus == B"
				// If you need this, you'd have to do multiple queries and combine results in memory.
				// For now, if middlemanStatus is a slice, we'll just log a warning and skip,
				// or assume it's for in-memory filtering later.
				// For direct Firestore query, it should be a single value.
				log.Printf("Warning: List filter for 'middlemanStatus' received a slice. Firestore does not support OR conditions on a single field directly. Filtering will happen in-memory in UseCase.")
				// We won't apply this filter to the Firestore query here.
				continue
			}
		}
		query = query.Where(key, "==", value)
	}

	countDocs, err := query.Documents(ctx).GetAll()
	if err != nil {
		log.Printf("Firestore error while counting transactions with filter %v: %v", filter, err)
		return nil, 0, errors.Internal("Failed to count transactions", err)
	}
	total := int64(len(countDocs))

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	iter := query.Documents(ctx)
	var transactions []*entity.Transaction

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Firestore error while iterating transactions with filter %v: %v", filter, err)
			return nil, 0, errors.Internal("Failed to iterate transactions", err)
		}

		var transaction entity.Transaction
		if err := doc.DataTo(&transaction); err != nil {
			log.Printf("Error parsing transaction data with filter %v: %v", filter, err)
			return nil, 0, errors.Internal("Failed to parse transaction data", err)
		}
		transactions = append(transactions, &transaction)
	}

	return transactions, total, nil
}

func (r *firestoreTransactionRepository) CreateLog(ctx context.Context, log *entity.TransactionLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}

	log.CreatedAt = time.Now()

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
	var field string
	if role == "buyer" {
		field = "buyerId"
	} else if role == "seller" {
		field = "sellerId"
	} else {
		return nil, 0, errors.BadRequest("Invalid role", nil)
	}

	query := r.client.Collection("transactions").Where(field, "==", userID)

	if status != "" {
		query = query.Where("status", "==", status)
	}

	query = query.OrderBy("createdAt", firestore.Desc)

	countDocs, err := query.Documents(ctx).GetAll()
	if err != nil {
		log.Printf("Firestore error while counting transactions for user %s, role %s, status %s: %v", userID, role, status, err)
		return nil, 0, errors.Internal("Failed to count transactions", err)
	}
	total := int64(len(countDocs))

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	iter := query.Documents(ctx)
	var transactions []*entity.Transaction

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Firestore error while iterating transactions for user %s, role %s, status %s: %v", userID, role, status, err)
			return nil, 0, errors.Internal("Failed to iterate transactions", err)
		}

		var transaction entity.Transaction
		if err := doc.DataTo(&transaction); err != nil {
			log.Printf("Error parsing transaction data for user %s, role %s, status %s: %v", userID, role, status, err)
			return nil, 0, errors.Internal("Failed to parse transaction data", err)
		}
		transactions = append(transactions, &transaction)
	}

	return transactions, total, nil
}

// Modified: ListPendingMiddlemanTransactions now fetches based on pending status and filters middlemanStatus in-memory
func (r *firestoreTransactionRepository) ListPendingMiddlemanTransactions(ctx context.Context, limit, offset int) ([]*entity.Transaction, int64, error) {
	// Fetch all middleman transactions that are still "pending"
	// We will filter by middlemanStatus in the usecase layer as Firestore doesn't support OR conditions on a single field.
	query := r.client.Collection("transactions").
		Where("deliveryMethod", "==", "middleman").
		Where("status", "==", "pending"). // Transactions that are still pending overall
		OrderBy("createdAt", firestore.Asc)

	allDocs, err := query.Documents(ctx).GetAll()
	if err != nil {
		log.Printf("Firestore error while listing all pending middleman transactions: %v", err)
		return nil, 0, errors.Internal("Failed to list all pending middleman transactions for filtering", err)
	}

	var filteredTransactions []*entity.Transaction
	for _, doc := range allDocs {
		var transaction entity.Transaction
		if err := doc.DataTo(&transaction); err != nil {
			log.Printf("Error parsing transaction data for pending middleman transactions: %v", err)
			continue
		}
		// Filter in-memory for relevant middleman statuses
		if transaction.MiddlemanStatus == "" || transaction.MiddlemanStatus == "assigned" || transaction.MiddlemanStatus == "awaiting_funds_confirmation" {
			filteredTransactions = append(filteredTransactions, &transaction)
		}
	}

	total := int64(len(filteredTransactions))

	// Apply pagination to the in-memory filtered list
	start := offset
	end := offset + limit

	if start > len(filteredTransactions) {
		return []*entity.Transaction{}, 0, nil
	}
	if end > len(filteredTransactions) {
		end = len(filteredTransactions)
	}

	paginatedTransactions := filteredTransactions[start:end]

	return paginatedTransactions, total, nil
}

func (r *firestoreTransactionRepository) GetTransactionStats(ctx context.Context, userID string, period string) (map[string]interface{}, error) {
	log.Printf("GetTransactionStats called for user %s with period %s", userID, period)
	return map[string]interface{}{
		"totalTransactions":     0,
		"totalSales":            0.0,
		"totalPurchases":        0.0,
		"completedTransactions": 0,
		"pendingTransactions":   0,
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
