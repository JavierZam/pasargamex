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
	GetCompletedTransactionCount(ctx context.Context, productID string) (int, error)
	GetPendingTransactionCount(ctx context.Context, productID string) (int, error)
	
	// Midtrans Integration Methods
	GetByMidtransOrderID(ctx context.Context, midtransOrderID string) (*entity.Transaction, error)
	
	// Approval System Methods
	CreateApproval(ctx context.Context, approval *entity.TransactionApproval) error
	GetApprovalsByTransactionID(ctx context.Context, transactionID string) ([]*entity.TransactionApproval, error)
	UpdateApproval(ctx context.Context, approval *entity.TransactionApproval) error
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

	// Debug: First check ALL transactions for this user+product combination
	debugQuery := r.client.Collection("transactions").
		Where("buyerId", "==", userID).
		Where("productId", "==", productID)
	
	debugIter := debugQuery.Documents(ctx)
	log.Printf("DEBUG: All transactions for user %s + product %s:", userID, productID)
	for {
		debugDoc, err := debugIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("DEBUG: Error reading transaction: %v", err)
			break
		}
		var debugTx entity.Transaction
		if err := debugDoc.DataTo(&debugTx); err == nil {
			log.Printf("DEBUG: Transaction ID=%s, PaymentStatus=%s, CredentialsDelivered=%v, Status=%s, CreatedAt=%v", 
				debugDoc.Ref.ID, debugTx.PaymentStatus, debugTx.CredentialsDelivered, debugTx.Status, debugTx.CreatedAt)
		}
	}

	query := r.client.Collection("transactions").
		Where("buyerId", "==", userID).
		Where("productId", "==", productID).
		Where("paymentStatus", "in", []string{"success", "paid"}).
		Where("credentialsDelivered", "==", true).
		Where("status", "==", "credentials_delivered").
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

	// Debug: log the actual transaction data that was found
	var transaction entity.Transaction
	if err := doc.DataTo(&transaction); err == nil {
		log.Printf("Completed transaction found: ID=%s, PaymentStatus=%s, CredentialsDelivered=%v, CreatedAt=%v", 
			doc.Ref.ID, transaction.PaymentStatus, transaction.CredentialsDelivered, transaction.CreatedAt)
	} else {
		log.Printf("Completed transaction found: %v (failed to parse data: %v)", doc.Ref.ID, err)
	}
	return true, nil
}

func (r *firestoreTransactionRepository) GetCompletedTransactionCount(ctx context.Context, productID string) (int, error) {
	log.Printf("Getting completed transaction count for product %s", productID)
	
	query := r.client.Collection("transactions").
		Where("productId", "==", productID).
		Where("paymentStatus", "in", []string{"success", "paid"}).
		Where("credentialsDelivered", "==", true).
		Where("status", "==", "credentials_delivered")
	
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		log.Printf("Error getting completed transaction count: %v", err)
		return 0, err
	}
	
	count := len(docs)
	log.Printf("Found %d completed transactions for product %s", count, productID)
	return count, nil
}

func (r *firestoreTransactionRepository) GetPendingTransactionCount(ctx context.Context, productID string) (int, error) {
	log.Printf("Getting pending transaction count for product %s", productID)
	
	query := r.client.Collection("transactions").
		Where("productId", "==", productID).
		Where("paymentStatus", "==", "pending").
		Where("status", "in", []string{"payment_pending", "payment_processing"})
	
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		log.Printf("Error getting pending transaction count: %v", err)
		return 0, err
	}
	
	count := len(docs)
	log.Printf("Found %d pending transactions for product %s", count, productID)
	return count, nil
}

// GetByMidtransOrderID retrieves a transaction by Midtrans order ID
func (r *firestoreTransactionRepository) GetByMidtransOrderID(ctx context.Context, midtransOrderID string) (*entity.Transaction, error) {
	log.Printf("Getting transaction by Midtrans order ID: %s", midtransOrderID)

	query := r.client.Collection("transactions").Where("midtransOrderId", "==", midtransOrderID).Limit(1)
	iter := query.Documents(ctx)

	doc, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, errors.NotFound("Transaction not found", nil)
		}
		return nil, errors.Internal("Failed to get transaction", err)
	}

	var transaction entity.Transaction
	if err := doc.DataTo(&transaction); err != nil {
		return nil, errors.Internal("Failed to parse transaction", err)
	}

	transaction.ID = doc.Ref.ID
	return &transaction, nil
}

// CreateApproval creates a new transaction approval
func (r *firestoreTransactionRepository) CreateApproval(ctx context.Context, approval *entity.TransactionApproval) error {
	if approval.ID == "" {
		approval.ID = uuid.New().String()
	}

	now := time.Now()
	approval.CreatedAt = now
	approval.UpdatedAt = now

	_, err := r.client.Collection("transaction_approvals").Doc(approval.ID).Set(ctx, approval)
	if err != nil {
		return errors.Internal("Failed to create approval", err)
	}

	log.Printf("Transaction approval created: %s", approval.ID)
	return nil
}

// GetApprovalsByTransactionID retrieves all approvals for a transaction
func (r *firestoreTransactionRepository) GetApprovalsByTransactionID(ctx context.Context, transactionID string) ([]*entity.TransactionApproval, error) {
	log.Printf("Getting approvals for transaction: %s", transactionID)

	query := r.client.Collection("transaction_approvals").Where("transactionId", "==", transactionID).OrderBy("createdAt", firestore.Asc)
	iter := query.Documents(ctx)

	var approvals []*entity.TransactionApproval
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, errors.Internal("Failed to get approvals", err)
		}

		var approval entity.TransactionApproval
		if err := doc.DataTo(&approval); err != nil {
			log.Printf("Failed to parse approval %s: %v", doc.Ref.ID, err)
			continue
		}

		approval.ID = doc.Ref.ID
		approvals = append(approvals, &approval)
	}

	log.Printf("Found %d approvals for transaction %s", len(approvals), transactionID)
	return approvals, nil
}

// UpdateApproval updates an existing transaction approval
func (r *firestoreTransactionRepository) UpdateApproval(ctx context.Context, approval *entity.TransactionApproval) error {
	approval.UpdatedAt = time.Now()

	_, err := r.client.Collection("transaction_approvals").Doc(approval.ID).Set(ctx, approval)
	if err != nil {
		return errors.Internal("Failed to update approval", err)
	}

	log.Printf("Transaction approval updated: %s", approval.ID)
	return nil
}
