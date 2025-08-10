package usecase

import (
	"context"
	"fmt"
	"time"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
	"pasargamex/pkg/logger"
	"pasargamex/pkg/utils"
)

type FeeCalculator interface {
	CalculateFee(amount float64, paymentMethod string) float64
}

type defaultFeeCalculator struct{}

func (fc *defaultFeeCalculator) CalculateFee(amount float64, paymentMethod string) float64 {
	return amount * 0.025
}

type TransactionUseCase struct {
	transactionRepo repository.TransactionRepository
	productRepo     repository.ProductRepository
	userRepo        repository.UserRepository
	feeCalculator   FeeCalculator
	chatUseCase     *ChatUseCase
	walletUseCase   *WalletUseCase
}

func NewTransactionUseCase(
	transactionRepo repository.TransactionRepository,
	productRepo repository.ProductRepository,
	userRepo repository.UserRepository,
	chatUseCase *ChatUseCase,
	walletUseCase *WalletUseCase,
) *TransactionUseCase {
	return &TransactionUseCase{
		transactionRepo: transactionRepo,
		productRepo:     productRepo,
		userRepo:        userRepo,
		feeCalculator:   &defaultFeeCalculator{},
		chatUseCase:     chatUseCase,
		walletUseCase:   walletUseCase,
	}
}

type CreateTransactionInput struct {
	ProductID      string
	DeliveryMethod string
	PaymentMethod  string // "wallet" or "external"
	Notes          string
}

type ProcessPaymentInput struct {
	PaymentMethod string
	PaymentDetails map[string]interface{}
}

func (uc *TransactionUseCase) CreateTransaction(ctx context.Context, buyerID string, input CreateTransactionInput) (*entity.Transaction, error) {
	product, err := uc.productRepo.GetByID(ctx, input.ProductID)
	if err != nil {
		return nil, err
	}

	seller, err := uc.userRepo.GetByID(ctx, product.SellerID)
	if err != nil {
		return nil, err
	}

	if product.SellerID == buyerID {
		return nil, errors.BadRequest("Cannot buy your own product", nil)
	}

	if product.Status != "active" {
		return nil, errors.BadRequest("Product is not available", nil)
	}

	if seller.VerificationStatus != "verified" {
		return nil, errors.BadRequest("Seller is not verified", nil)
	}

	if input.DeliveryMethod != "instant" && input.DeliveryMethod != "middleman" {
		return nil, errors.BadRequest("Invalid delivery method", nil)
	}

	if input.DeliveryMethod == "instant" && len(product.Credentials) == 0 {
		return nil, errors.BadRequest("Product credentials are not available", nil)
	}

	fee := uc.feeCalculator.CalculateFee(product.Price, "")
	totalAmount := product.Price + fee

	transaction := &entity.Transaction{
		ProductID:      input.ProductID,
		SellerID:       product.SellerID,
		BuyerID:        buyerID,
		Status:         "pending", // Initial status is always pending
		DeliveryMethod: input.DeliveryMethod,
		Amount:         product.Price,
		Fee:            fee,
		TotalAmount:    totalAmount,
		PaymentStatus:  "pending", // Payment status also pending initially
		SellerReviewed: false,
		BuyerReviewed:  false,
		Notes:          input.Notes,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if input.DeliveryMethod == "instant" {
		transaction.Credentials = product.Credentials
	}

	if err := uc.transactionRepo.Create(ctx, transaction); err != nil {
		return nil, err
	}

	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        "pending",
		Notes:         "Transaction created",
		CreatedBy:     buyerID,
		CreatedAt:     time.Now(),
	}

	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		logger.Error("Failed to create transaction log for transaction %s: %v", transaction.ID, err)
	}

	return transaction, nil
}

func (uc *TransactionUseCase) GetTransactionByID(ctx context.Context, userID, transactionID string) (interface{}, error) {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	if transaction.BuyerID != userID && transaction.SellerID != userID && transaction.AdminID != userID { // Allow admin to view
		return nil, errors.Forbidden("You don't have permission to view this transaction", nil)
	}

	return uc.prepareTransactionResponse(transaction, userID), nil
}

// GetTransactionStatus returns lightweight status information
func (uc *TransactionUseCase) GetTransactionStatus(ctx context.Context, userID, transactionID string) (map[string]interface{}, error) {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	if transaction.BuyerID != userID && transaction.SellerID != userID && transaction.AdminID != userID {
		return nil, errors.Forbidden("You don't have permission to view this transaction", nil)
	}

	// Return lightweight status info for frontend polling
	statusInfo := map[string]interface{}{
		"id":                  transaction.ID,
		"status":              transaction.Status,
		"payment_status":      transaction.PaymentStatus,
		"delivery_method":     transaction.DeliveryMethod,
		"total_amount":        transaction.TotalAmount,
		"created_at":          transaction.CreatedAt,
		"updated_at":          transaction.UpdatedAt,
		"credentials_delivered": transaction.CredentialsDelivered,
	}

	// Add payment info if exists
	if transaction.PaymentAt != nil {
		statusInfo["payment_at"] = transaction.PaymentAt
	}

	// Add delivery info if exists
	if transaction.CredentialsDeliveredAt != nil {
		statusInfo["credentials_delivered_at"] = transaction.CredentialsDeliveredAt
	}

	// Add middleman info if applicable
	if transaction.DeliveryMethod == "middleman" {
		statusInfo["middleman_status"] = transaction.MiddlemanStatus
		statusInfo["admin_id"] = transaction.AdminID
	}

	return statusInfo, nil
}

func (uc *TransactionUseCase) ListTransactions(ctx context.Context, userID, role, status string, page, limit int) ([]interface{}, int64, error) {
	if role != "buyer" && role != "seller" {
		role = "buyer"
	}

	pagination := utils.NewPaginationParams(page, limit)

	transactions, total, err := uc.transactionRepo.ListByUserID(ctx, userID, role, status, pagination.PageSize, pagination.Offset)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]interface{}, len(transactions))
	for i, transaction := range transactions {
		responses[i] = uc.prepareTransactionResponse(transaction, userID)
	}

	return responses, total, nil
}

// Modified: ProcessPayment now handles wallet payment and middleman transactions
func (uc *TransactionUseCase) ProcessPayment(ctx context.Context, userID, transactionID, paymentMethod string, paymentDetails map[string]interface{}) (*entity.Transaction, error) {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	if transaction.BuyerID != userID {
		return nil, errors.Forbidden("Only buyer can make payment", nil)
	}

	if transaction.PaymentStatus == "paid" || transaction.PaymentStatus == "refunded" {
		return nil, errors.BadRequest("Payment already processed or refunded", nil)
	}

	// Process wallet payment
	if paymentMethod == "wallet" {
		if uc.walletUseCase == nil {
			return nil, errors.InternalServer("Wallet service not available", nil)
		}

		// Process wallet payment
		description := fmt.Sprintf("Payment for transaction %s - %s", transaction.ID, transaction.ProductID)
		_, err := uc.walletUseCase.ProcessWalletPayment(ctx, userID, transaction.TotalAmount, description, transaction.ID)
		if err != nil {
			return nil, err // This will return appropriate error (insufficient balance, etc.)
		}
	}

	transaction.PaymentMethod = paymentMethod
	transaction.PaymentDetails = paymentDetails
	transaction.PaymentStatus = "paid" // Mark payment as initiated/paid by buyer

	now := time.Now()
	transaction.PaymentAt = &now

	if transaction.DeliveryMethod == "instant" {
		// For instant delivery, payment means completion
		transaction.Status = "completed"
		transaction.CompletedAt = &now
	} else if transaction.DeliveryMethod == "middleman" {
		// For middleman, payment by buyer means awaiting middleman confirmation
		// Status remains "pending" until middleman confirms funds received
		transaction.Status = "pending"                              // Explicitly keep as pending
		transaction.MiddlemanStatus = "awaiting_funds_confirmation" // New status for middleman
	}

	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		// If transaction update fails and we used wallet, we should refund
		if paymentMethod == "wallet" && uc.walletUseCase != nil {
			refundDescription := fmt.Sprintf("Refund for failed transaction %s", transaction.ID)
			uc.walletUseCase.ProcessWalletRefund(ctx, userID, transaction.TotalAmount, refundDescription, transaction.ID)
		}
		return nil, err
	}

	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        transaction.Status, // Log the current status (pending)
		Notes:         "Payment initiated by buyer via " + paymentMethod,
		CreatedBy:     userID,
		CreatedAt:     time.Now(),
	}

	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		logger.Error("Failed to create payment log for transaction %s: %v", transaction.ID, err)
	}

	// New: Send system message about payment initiation for middleman transactions
	if transaction.DeliveryMethod == "middleman" && transaction.MiddlemanChatID != "" {
		uc.chatUseCase.SendSystemMessage(ctx, transaction.MiddlemanChatID, "Buyer has initiated payment. Awaiting middleman's confirmation of funds received.", "payment_initiated", map[string]interface{}{"transaction_id": transaction.ID, "payment_method": paymentMethod})
	}

	return transaction, nil
}

// New: ConfirmMiddlemanPayment confirms that middleman has received funds
func (uc *TransactionUseCase) ConfirmMiddlemanPayment(ctx context.Context, adminID, transactionID string) (*entity.Transaction, error) {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	if transaction.AdminID != adminID {
		return nil, errors.Forbidden("Only the assigned middleman can confirm payment", nil)
	}

	if transaction.DeliveryMethod != "middleman" {
		return nil, errors.BadRequest("Transaction is not a middleman transaction", nil)
	}

	if transaction.PaymentStatus != "paid" || transaction.Status != "pending" || transaction.MiddlemanStatus != "awaiting_funds_confirmation" {
		return nil, errors.BadRequest("Transaction is not in the correct state for payment confirmation", nil)
	}

	transaction.Status = "processing"              // Now it's truly processing
	transaction.MiddlemanStatus = "funds_received" // Funds confirmed by middleman
	transaction.UpdatedAt = time.Now()

	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return nil, err
	}

	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        "processing",
		Notes:         "Middleman confirmed funds received",
		CreatedBy:     adminID,
		CreatedAt:     time.Now(),
	}

	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		logger.Error("Failed to create middleman payment confirmation log for transaction %s: %v", transaction.ID, err)
	}

	// Send system message about funds received
	if transaction.MiddlemanChatID != "" {
		uc.chatUseCase.SendSystemMessage(ctx, transaction.MiddlemanChatID, "Middleman confirmed funds received. Seller, please provide credentials to Buyer.", "funds_received_confirmed", map[string]interface{}{"transaction_id": transaction.ID})
	}

	return transaction, nil
}

func (uc *TransactionUseCase) AssignMiddleman(ctx context.Context, adminID, transactionID string) (*entity.Transaction, error) {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	if transaction.DeliveryMethod != "middleman" {
		return nil, errors.BadRequest("Transaction is not using middleman delivery method", nil)
	}

	if transaction.Status != "pending" || transaction.MiddlemanStatus != "" { // Middleman can be assigned when transaction is pending and no middleman yet
		return nil, errors.BadRequest("Transaction is not ready for middleman assignment. Status must be 'pending' and no middleman assigned yet.", nil)
	}

	transaction.AdminID = adminID
	transaction.MiddlemanStatus = "assigned" // Mark as assigned
	transaction.UpdatedAt = time.Now()

	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return nil, err
	}

	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        transaction.Status, // Log the current status (pending)
		Notes:         "Middleman assigned",
		CreatedBy:     adminID,
		CreatedAt:     time.Now(),
	}

	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		logger.Error("Failed to create middleman assignment log for transaction %s: %v", transaction.ID, err)
	}

	// Create the middleman chat room here
	middlemanChat, err := uc.chatUseCase.CreateMiddlemanChat(ctx, CreateMiddlemanChatInput{
		BuyerID:        transaction.BuyerID,
		SellerID:       transaction.SellerID,
		MiddlemanID:    adminID, // The assigned admin is the middleman
		ProductID:      transaction.ProductID,
		TransactionID:  transaction.ID,
		InitialMessage: "Welcome to your secure transaction chat! I'm your middleman. Please follow my instructions to complete the transaction.",
	})
	if err != nil {
		logger.Error("Failed to create middleman chat for transaction %s: %v", transaction.ID, err)
	} else {
		transaction.MiddlemanChatID = middlemanChat.Chat.ID
		if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
			logger.Error("Failed to update transaction %s with middleman chat ID: %v", transaction.ID, err)
		}
	}

	// Send system message about middleman assignment
	if transaction.MiddlemanChatID != "" {
		uc.chatUseCase.SendSystemMessage(ctx, transaction.MiddlemanChatID, "Middleman assigned. Buyer, please initiate payment to the middleman.", "middleman_assigned", map[string]interface{}{"transaction_id": transaction.ID, "middleman_id": adminID})
	}

	return transaction, nil
}

func (uc *TransactionUseCase) VerifyAndCompleteMiddleman(ctx context.Context, adminID, transactionID string, credentials map[string]interface{}) (*entity.Transaction, error) {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	if transaction.AdminID != adminID {
		return nil, errors.Forbidden("Only assigned admin can complete this transaction", nil)
	}

	if transaction.Status != "processing" || transaction.MiddlemanStatus != "funds_received" { // Ensure funds are confirmed
		return nil, errors.BadRequest("Transaction is not ready for completion. Funds must be confirmed by middleman.", nil)
	}

	transaction.Status = "completed"
	transaction.MiddlemanStatus = "completed"
	transaction.Credentials = credentials

	now := time.Now()
	transaction.CompletedAt = &now
	transaction.UpdatedAt = now

	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return nil, err
	}

	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        "completed",
		Notes:         "Transaction completed by middleman",
		CreatedBy:     adminID,
		CreatedAt:     time.Now(),
	}

	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		logger.Error("Failed to create middleman completion log for transaction %s: %v", transaction.ID, err)
	}

	if transaction.DeliveryMethod == "middleman" && transaction.MiddlemanChatID != "" {
		uc.chatUseCase.SendSystemMessage(ctx, transaction.MiddlemanChatID, "Transaction completed successfully! Funds released to seller.", "transaction_completed", map[string]interface{}{"transaction_id": transaction.ID})
	}

	return transaction, nil
}

func (uc *TransactionUseCase) GetTransactionLogs(ctx context.Context, userID, transactionID string) ([]*entity.TransactionLog, error) {
	_, err := uc.GetTransactionByID(ctx, userID, transactionID)
	if err != nil {
		return nil, err
	}

	logs, err := uc.transactionRepo.ListLogsByTransactionID(ctx, transactionID)
	if err != nil {
		return nil, errors.Internal("Failed to get transaction logs", err)
	}

	return logs, nil
}

func (uc *TransactionUseCase) CancelTransaction(ctx context.Context, userID, transactionID, reason string) (*entity.Transaction, error) {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	if transaction.Status == "completed" || transaction.Status == "cancelled" {
		return nil, errors.BadRequest("Transaction cannot be cancelled in current status", nil)
	}

	isBuyer := transaction.BuyerID == userID
	isSeller := transaction.SellerID == userID
	isAdmin := transaction.AdminID == userID

	if !isBuyer && !isSeller && !isAdmin {
		return nil, errors.Forbidden("You don't have permission to cancel this transaction", nil)
	}

	if !uc.isValidStatusTransition(transaction.Status, "cancelled", transaction.DeliveryMethod, isBuyer, isSeller) && !isAdmin {
		return nil, errors.BadRequest("Cannot cancel transaction in current status", nil)
	}

	transaction.Status = "cancelled"
	transaction.CancellationReason = reason

	now := time.Now()
	transaction.CancelledAt = &now
	transaction.UpdatedAt = now

	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return nil, err
	}

	var createdBy string
	var notes string

	if isBuyer {
		createdBy = transaction.BuyerID
		notes = "Transaction cancelled by buyer"
	} else if isSeller {
		createdBy = transaction.SellerID
		notes = "Transaction cancelled by seller"
	} else {
		createdBy = transaction.AdminID
		notes = "Transaction cancelled by admin"
	}

	if reason != "" {
		notes += ": " + reason
	}

	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        "cancelled",
		Notes:         notes,
		CreatedBy:     createdBy,
		CreatedAt:     time.Now(),
	}

	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		logger.Error("Failed to create cancellation log for transaction %s: %v", transaction.ID, err)
	}

	if transaction.DeliveryMethod == "middleman" && transaction.MiddlemanChatID != "" {
		uc.chatUseCase.SendSystemMessage(ctx, transaction.MiddlemanChatID, "Transaction cancelled. Reason: "+reason, "transaction_cancelled", map[string]interface{}{"transaction_id": transaction.ID, "reason": reason})
	}

	return transaction, nil
}

func (uc *TransactionUseCase) CreateDispute(ctx context.Context, userID, transactionID, reason string) (*entity.Transaction, error) {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	if transaction.Status != "processing" {
		return nil, errors.BadRequest("Transaction cannot be disputed in current status", nil)
	}

	isBuyer := transaction.BuyerID == userID
	isSeller := transaction.SellerID == userID

	if !isBuyer && !isSeller {
		return nil, errors.Forbidden("Only buyer or seller can create a dispute", nil)
	}

	transaction.Status = "disputed"
	transaction.UpdatedAt = time.Now()

	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return nil, err
	}

	var createdBy string
	var notes string

	if isBuyer {
		createdBy = transaction.BuyerID
		notes = "Dispute created by buyer"
	} else {
		createdBy = transaction.SellerID
		notes = "Dispute created by seller"
	}

	if reason != "" {
		notes += ": " + reason
	}

	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        "disputed",
		Notes:         notes,
		CreatedBy:     createdBy,
		CreatedAt:     time.Now(),
	}

	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		logger.Error("Failed to create dispute log for transaction %s: %v", transaction.ID, err)
	}

	if transaction.DeliveryMethod == "middleman" && transaction.MiddlemanChatID != "" {
		uc.chatUseCase.SendSystemMessage(ctx, transaction.MiddlemanChatID, "Transaction disputed. Reason: "+reason, "transaction_disputed", map[string]interface{}{"transaction_id": transaction.ID, "reason": reason})
	}

	return transaction, nil
}

func (uc *TransactionUseCase) ResolveDispute(ctx context.Context, adminID, transactionID, resolution string, refund bool) (*entity.Transaction, error) {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	if transaction.Status != "disputed" {
		return nil, errors.BadRequest("Transaction is not in disputed status", nil)
	}

	var newStatus string
	now := time.Now()

	if refund {
		newStatus = "cancelled"
		transaction.Status = newStatus
		transaction.CancelledAt = &now
		transaction.PaymentStatus = "refunded"
		transaction.RefundedAt = &now
	} else {
		newStatus = "completed"
		transaction.Status = newStatus
		transaction.CompletedAt = &now
	}

	transaction.UpdatedAt = now

	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return nil, err
	}

	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        newStatus,
		Notes:         "Dispute resolved by admin: " + resolution,
		CreatedBy:     adminID,
		CreatedAt:     now,
	}

	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		logger.Error("Failed to create dispute resolution log for transaction %s: %v", transaction.ID, err)
	}

	if transaction.DeliveryMethod == "middleman" && transaction.MiddlemanChatID != "" {
		uc.chatUseCase.SendSystemMessage(ctx, transaction.MiddlemanChatID, "Dispute resolved by middleman. Status: "+newStatus, "dispute_resolved", map[string]interface{}{"transaction_id": transaction.ID, "resolution": resolution, "refund": refund})
	}

	return transaction, nil
}

func (uc *TransactionUseCase) ConfirmDelivery(ctx context.Context, buyerID, transactionID string) (*entity.Transaction, error) {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	if transaction.BuyerID != buyerID {
		return nil, errors.Forbidden("Only buyer can confirm delivery", nil)
	}

	if transaction.Status != "processing" {
		return nil, errors.BadRequest("Transaction is not in processing status", nil)
	}

	transaction.Status = "completed"
	now := time.Now()
	transaction.CompletedAt = &now
	transaction.UpdatedAt = now

	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return nil, err
	}

	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        "completed",
		Notes:         "Delivery confirmed by buyer",
		CreatedBy:     buyerID,
		CreatedAt:     now,
	}

	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		logger.Error("Failed to create delivery confirmation log for transaction %s: %v", transaction.ID, err)
	}

	if transaction.DeliveryMethod == "middleman" && transaction.MiddlemanChatID != "" {
		uc.chatUseCase.SendSystemMessage(ctx, transaction.MiddlemanChatID, "Delivery confirmed by buyer. Transaction completed.", "delivery_confirmed", map[string]interface{}{"transaction_id": transaction.ID})
	}

	return transaction, nil
}

func (uc *TransactionUseCase) isValidStatusTransition(currentStatus, newStatus, deliveryMethod string, isBuyer, isSeller bool) bool {
	validTransitions := map[string]map[string]struct {
		allowed bool
		roles   []string
	}{
		"pending": {
			"awaiting_payment": {true, []string{"buyer"}}, // Buyer initiates payment for instant
			"cancelled":        {true, []string{"buyer", "seller"}},
			// New: For middleman, payment confirmation moves it to processing
		},
		"awaiting_middleman_payment_confirmation": { // New status for middleman flow
			"processing": {true, []string{"admin"}}, // Only admin (middleman) can move to processing
			"cancelled":  {true, []string{"admin"}},
		},
		"processing": {
			"completed": {true, []string{"admin", "buyer"}},
			"cancelled": {true, []string{"admin"}},
			"disputed":  {true, []string{"buyer", "seller"}},
		},
		"disputed": {
			"completed": {true, []string{"admin"}},
			"cancelled": {true, []string{"admin"}},
		},
	}
	// Special case for instant delivery payment directly to completed
	if deliveryMethod == "instant" && currentStatus == "pending" && newStatus == "completed" {
		return true
	}
	// Special case for middleman payment initiated by buyer
	if deliveryMethod == "middleman" && currentStatus == "pending" && newStatus == "awaiting_funds_confirmation" { // This is not a status change, but internal flag
		return true
	}

	if transition, exists := validTransitions[currentStatus][newStatus]; exists {
		for _, role := range transition.roles {
			if (role == "buyer" && isBuyer) || (role == "seller" && isSeller) || role == "system" || role == "admin" {
				return true
			}
		}
	}

	return false
}

func (uc *TransactionUseCase) ListAdminTransactions(ctx context.Context, adminID string, filter map[string]interface{}, page, limit int) ([]*entity.Transaction, int64, error) {
	user, err := uc.userRepo.GetByID(ctx, adminID)
	if err != nil {
		return nil, 0, errors.NotFound("Admin user", err)
	}

	if user.Role != "admin" {
		return nil, 0, errors.Forbidden("Only admin can access this resource", nil)
	}

	pagination := utils.NewPaginationParams(page, limit)

	transactions, total, err := uc.transactionRepo.List(ctx, filter, pagination.PageSize, pagination.Offset)
	if err != nil {
		return nil, 0, errors.Internal("Failed to list transactions", err)
	}

	return transactions, total, nil
}

func (uc *TransactionUseCase) ListPendingMiddlemanTransactions(ctx context.Context, adminID string, page, limit int) ([]*entity.Transaction, int64, error) {
	user, err := uc.userRepo.GetByID(ctx, adminID)
	if err != nil {
		return nil, 0, errors.NotFound("Admin user", err)
	}

	if user.Role != "admin" {
		return nil, 0, errors.Forbidden("Only admin can access this resource", nil)
	}

	pagination := utils.NewPaginationParams(page, limit)

	transactions, total, err := uc.transactionRepo.ListPendingMiddlemanTransactions(ctx, pagination.PageSize, pagination.Offset)
	if err != nil {
		return nil, 0, errors.Internal("Failed to list pending middleman transactions", err)
	}

	return transactions, total, nil
}

func (uc *TransactionUseCase) prepareTransactionResponse(transaction *entity.Transaction, userID string) interface{} {
	type TransactionResponse struct {
		ID              string                 `json:"id"`
		ProductID       string                 `json:"product_id"`
		SellerID        string                 `json:"seller_id"`
		BuyerID         string                 `json:"buyer_id"`
		Status          string                 `json:"status"`
		DeliveryMethod  string                 `json:"delivery_method"`
		Amount          float64                `json:"amount"`
		Fee             float64                `json:"fee"`
		TotalAmount     float64                `json:"total_amount"`
		PaymentMethod   string                 `json:"payment_method,omitempty"`
		PaymentStatus   string                 `json:"payment_status"`
		PaymentDetails  map[string]interface{} `json:"payment_details,omitempty"`
		AdminID         string                 `json:"admin_id,omitempty"`
		MiddlemanStatus string                 `json:"middleman_status,omitempty"`
		Notes           string                 `json:"notes,omitempty"`
		MiddlemanChatID string                 `json:"middleman_chat_id,omitempty"` // New: Middleman Chat ID

		Credentials map[string]interface{} `json:"credentials,omitempty"`
	}

	response := TransactionResponse{
		ID:              transaction.ID,
		ProductID:       transaction.ProductID,
		SellerID:        transaction.SellerID,
		BuyerID:         transaction.BuyerID,
		Status:          transaction.Status,
		DeliveryMethod:  transaction.DeliveryMethod,
		Amount:          transaction.Amount,
		Fee:             transaction.Fee,
		TotalAmount:     transaction.TotalAmount,
		PaymentMethod:   transaction.PaymentMethod,
		PaymentStatus:   transaction.PaymentStatus,
		PaymentDetails:  transaction.PaymentDetails,
		AdminID:         transaction.AdminID,
		MiddlemanStatus: transaction.MiddlemanStatus,
		Notes:           transaction.Notes,
		MiddlemanChatID: transaction.MiddlemanChatID, // Assign new field
	}

	if (transaction.SellerID == userID) ||
		(transaction.BuyerID == userID && transaction.PaymentStatus == "paid") { // Credentials visible to buyer only after payment is "paid" (initiated)
		response.Credentials = transaction.Credentials
	}

	return response
}
