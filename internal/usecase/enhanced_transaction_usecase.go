package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"crypto/rand"
	"encoding/hex"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/internal/domain/service"
	"pasargamex/internal/infrastructure/websocket"
	"pasargamex/pkg/errors"
)

type EnhancedTransactionUseCase struct {
	transactionRepo repository.TransactionRepository
	productRepo     repository.ProductRepository
	userRepo        repository.UserRepository
	feeCalculator   FeeCalculator
	paymentGateway  service.PaymentGatewayService
	chatUseCase     *ChatUseCase
	walletUseCase   *WalletUseCase
	wsManager       *websocket.Manager
}

func NewEnhancedTransactionUseCase(
	transactionRepo repository.TransactionRepository,
	productRepo repository.ProductRepository,
	userRepo repository.UserRepository,
	paymentGateway service.PaymentGatewayService,
	chatUseCase *ChatUseCase,
	walletUseCase *WalletUseCase,
	wsManager *websocket.Manager,
) *EnhancedTransactionUseCase {
	return &EnhancedTransactionUseCase{
		transactionRepo: transactionRepo,
		productRepo:     productRepo,
		userRepo:        userRepo,
		feeCalculator:   &defaultFeeCalculator{},
		paymentGateway:  paymentGateway,
		chatUseCase:     chatUseCase,
		walletUseCase:   walletUseCase,
		wsManager:       wsManager,
	}
}

type CreateSecureTransactionInput struct {
	ProductID      string
	DeliveryMethod string
	PaymentMethod  string // "midtrans_snap", "midtrans_bank_transfer", "wallet"
	MiddlemanID    string // Required for middleman delivery
	Notes          string
	Embed          bool   // For Midtrans: true = embed/popup, false = redirect
	
	// Customer details for payment gateway
	CustomerDetails service.CustomerDetails
}

type SecureTransactionResponse struct {
	Transaction     *entity.Transaction `json:"transaction"`
	PaymentToken    string              `json:"payment_token,omitempty"`
	PaymentURL      string              `json:"payment_url,omitempty"`
	VirtualAccounts []service.VaNumber  `json:"virtual_accounts,omitempty"`
}

func (uc *EnhancedTransactionUseCase) CreateSecureTransaction(ctx context.Context, buyerID string, input CreateSecureTransactionInput) (*SecureTransactionResponse, error) {
	log.Printf("Creating secure transaction for buyer: %s, product: %s", buyerID, input.ProductID)

	// 1. Validate product and seller
	product, err := uc.productRepo.GetByID(ctx, input.ProductID)
	if err != nil {
		return nil, err
	}

	seller, err := uc.userRepo.GetByID(ctx, product.SellerID)
	if err != nil {
		return nil, err
	}

	buyer, err := uc.userRepo.GetByID(ctx, buyerID)
	if err != nil {
		return nil, err
	}

	// 2. Business validations
	if product.SellerID == buyerID {
		return nil, errors.BadRequest("Cannot buy your own product", nil)
	}

	if product.Status != "active" {
		return nil, errors.BadRequest("Product is not available", nil)
	}

	if seller.VerificationStatus != "verified" {
		return nil, errors.BadRequest("Seller is not verified", nil)
	}

	// CRITICAL: Check product stock availability with race condition protection
	// For single-use products (like game accounts), treat stock as 1 if not set
	maxStock := product.Stock
	if maxStock == 0 && len(product.Credentials) > 0 {
		maxStock = 1 // Single-use product with credentials
	}
	
	if maxStock > 0 {
		// Check if product is already sold out
		if product.SoldCount >= maxStock {
			return nil, errors.BadRequest("Product is sold out", nil)
		}
		
		// Additional check: count pending + completed transactions to prevent overselling
		pendingCount, err := uc.transactionRepo.GetPendingTransactionCount(ctx, input.ProductID)
		if err != nil {
			log.Printf("Error checking pending transaction count: %v", err)
		} else {
			totalReserved := product.SoldCount + pendingCount
			if totalReserved >= maxStock {
				return nil, errors.BadRequest("Product is sold out (reserved)", nil)
			}
		}
	}

	if input.DeliveryMethod != "instant" && input.DeliveryMethod != "middleman" {
		return nil, errors.BadRequest("Invalid delivery method", nil)
	}

	// FRAUD DETECTION: Analyze transaction for fraud risk
	fraudUseCase := NewFraudDetectionUseCase(uc.transactionRepo, uc.userRepo)
	
	// Create temporary transaction for fraud analysis (after price calculation)
	fee := uc.feeCalculator.CalculateFee(product.Price, input.PaymentMethod)
	totalAmount := product.Price + fee
	
	tempTransaction := &entity.Transaction{
		ProductID:   input.ProductID,
		BuyerID:     buyerID,
		SellerID:    product.SellerID,
		TotalAmount: totalAmount,
	}
	
	fraudResult, err := fraudUseCase.AnalyzeTransaction(ctx, tempTransaction, buyer, seller, product)
	if err != nil {
		log.Printf("Fraud analysis failed: %v", err)
		// Continue with transaction but log the error
	} else {
		log.Printf("Fraud analysis result: Score=%.2f, Risk=%s, Action=%s", 
			fraudResult.Score, fraudResult.RiskLevel, fraudResult.Action)
			
		// Block high-risk transactions
		if fraudResult.Action == "block" {
			return nil, errors.BadRequest("Transaction blocked for security reasons. Please contact support.", nil)
		}
		
		// Flag for review but allow to proceed
		if fraudResult.Action == "review" {
			log.Printf("SECURITY: Transaction %s flagged for review: %v", tempTransaction.ID, fraudResult.Reasons)
		}
	}

	if input.DeliveryMethod == "instant" && len(product.Credentials) == 0 {
		return nil, errors.BadRequest("Product credentials are not available", nil)
	}

	if input.DeliveryMethod == "middleman" && input.MiddlemanID == "" {
		return nil, errors.BadRequest("Middleman ID is required for middleman delivery", nil)
	}

	// 4. Create transaction with security fields
	transactionID := uc.generateID()
	midtransOrderID := fmt.Sprintf("PGX-%s-%d", transactionID, time.Now().Unix())

	transaction := &entity.Transaction{
		ID:             transactionID,
		ProductID:      input.ProductID,
		SellerID:       product.SellerID,
		BuyerID:        buyerID,
		Status:         "payment_pending",
		DeliveryMethod: input.DeliveryMethod,
		Amount:         product.Price,
		Fee:            fee,
		TotalAmount:    totalAmount,
		PaymentMethod:  input.PaymentMethod,
		PaymentStatus:  "pending",
		AdminID:        input.MiddlemanID,
		// Add fraud analysis results if available
		FraudScore:     0.0,
		SecurityFlags:  []string{},
		
		// Midtrans fields
		MidtransOrderID: midtransOrderID,
		
		// Security fields
		SecurityLevel:  uc.calculateSecurityLevel(totalAmount, input.DeliveryMethod),
		EscrowStatus:   "pending",
		RequiredApprovals: uc.getRequiredApprovals(input.DeliveryMethod),
		
		Notes:     input.Notes,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// Update fraud analysis results before saving
	if fraudResult != nil {
		transaction.FraudScore = fraudResult.Score
		transaction.SecurityFlags = fraudResult.Flags
	}

	// 5. Save transaction
	if err := uc.transactionRepo.Create(ctx, transaction); err != nil {
		return nil, errors.Internal("Failed to create transaction", err)
	}

	// 6. Create payment via Midtrans (if not wallet payment)
	response := &SecureTransactionResponse{
		Transaction: transaction,
	}

	if input.PaymentMethod != "wallet" {
		paymentResp, err := uc.createMidtransPayment(ctx, transaction, product, input.CustomerDetails, input.Embed)
		if err != nil {
			log.Printf("Failed to create Midtrans payment: %v", err)
			// Update transaction status to failed
			transaction.Status = "payment_failed"
			transaction.PaymentStatus = "failed"
			uc.transactionRepo.Update(ctx, transaction)
			return nil, errors.Internal("Failed to create payment", err)
		}

		// Update transaction with payment details
		transaction.MidtransToken = paymentResp.Token
		transaction.MidtransRedirectURL = paymentResp.RedirectURL
		transaction.PaymentDetails = map[string]interface{}{
			"payment_type": paymentResp.PaymentType,
			"va_numbers":   paymentResp.VaNumbers,
		}

		if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
			log.Printf("Failed to update transaction with payment details: %v", err)
		}

		response.PaymentToken = paymentResp.Token
		response.PaymentURL = paymentResp.RedirectURL
		response.VirtualAccounts = paymentResp.VaNumbers
	}

	// 7. Create transaction chat if middleman delivery
	if input.DeliveryMethod == "middleman" && input.MiddlemanID != "" {
		chatInput := CreateTransactionChatInput{
			BuyerID:        buyerID,
			SellerID:       product.SellerID,
			ProductID:      input.ProductID,
			MiddlemanID:    input.MiddlemanID,
			InitialMessage: fmt.Sprintf("Transaction chat created for %s", product.Title),
		}

		chat, err := uc.chatUseCase.CreateTransactionChat(ctx, buyerID, chatInput)
		if err != nil {
			log.Printf("Failed to create transaction chat: %v", err)
		} else {
			transaction.MiddlemanChatID = chat.ID
			uc.transactionRepo.Update(ctx, transaction)
		}
	}

	log.Printf("Secure transaction created successfully: %s", transactionID)
	return response, nil
}

func (uc *EnhancedTransactionUseCase) createMidtransPayment(ctx context.Context, transaction *entity.Transaction, product *entity.Product, customerDetails service.CustomerDetails, embed bool) (*service.PaymentGatewayResponse, error) {
	// Create payment request
	paymentReq := service.PaymentGatewayRequest{
		OrderID:     transaction.MidtransOrderID,
		Amount:      transaction.TotalAmount,
		PaymentType: transaction.PaymentMethod,
		Embed:       embed,
		CustomerDetails: customerDetails,
		ItemDetails: []service.ItemDetail{
			{
				ID:       product.ID,
				Price:    product.Price,
				Quantity: 1,
				Name:     product.Title,
				Category: "Gaming Product",
			},
			{
				ID:       "platform_fee",
				Price:    transaction.Fee,
				Quantity: 1,
				Name:     "Platform Fee",
				Category: "Service Fee",
			},
		},
	}

	return uc.paymentGateway.CreatePayment(ctx, paymentReq)
}

func (uc *EnhancedTransactionUseCase) calculateSecurityLevel(amount float64, deliveryMethod string) string {
	if deliveryMethod == "instant" {
		return "low"
	}

	if amount >= 10000000 { // >= 10 juta
		return "high"
	} else if amount >= 1000000 { // >= 1 juta
		return "medium"
	}

	return "low"
}

func (uc *EnhancedTransactionUseCase) getRequiredApprovals(deliveryMethod string) []string {
	if deliveryMethod == "instant" {
		return []string{"system"} // Automatic
	}

	return []string{"seller", "middleman", "buyer"}
}




// GetPaymentStatus gets the current payment status for a transaction
func (uc *EnhancedTransactionUseCase) GetPaymentStatus(ctx context.Context, transactionID, userID string) (map[string]interface{}, error) {
	log.Printf("GetPaymentStatus called for transaction: %s, user: %s", transactionID, userID)
	
	// Get transaction
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("transaction not found: %v", err)
	}

	log.Printf("Transaction found: PaymentStatus='%s', MidtransOrderID='%s', Status='%s'", 
		transaction.PaymentStatus, transaction.MidtransOrderID, transaction.Status)

	// Verify user has access to this transaction
	if transaction.BuyerID != userID && transaction.SellerID != userID {
		return nil, fmt.Errorf("unauthorized access to transaction")
	}

	// Get latest status from Midtrans if needed
	if transaction.PaymentStatus == "pending" && transaction.MidtransOrderID != "" {
		log.Printf("Checking payment status with Midtrans for order: %s", transaction.MidtransOrderID)
		result, err := uc.paymentGateway.GetPaymentStatus(ctx, transaction.MidtransOrderID)
		if err == nil && result.Status != transaction.PaymentStatus {
			log.Printf("Payment status changed: %s -> %s", transaction.PaymentStatus, result.Status)
			// Update transaction with latest status from Midtrans
			transaction.PaymentStatus = result.Status
			if result.Status == "success" {
				transaction.Status = "paid"
				transaction.PaymentAt = &[]time.Time{time.Now()}[0]
				
				// Trigger instant delivery if applicable
				if transaction.DeliveryMethod == "instant" {
					// Use background context for webhook operations to avoid auth issues
					go uc.processInstantDeliveryFromWebhook(context.Background(), transaction)
				}
			}
			uc.transactionRepo.Update(ctx, transaction)
		} else if err != nil {
			log.Printf("Error checking payment status: %v", err)
		}
	}

	return map[string]interface{}{
		"transaction_id":       transaction.ID,
		"payment_status":       transaction.PaymentStatus,
		"status":               transaction.Status,
		"total_amount":         transaction.TotalAmount,
		"midtrans_order_id":    transaction.MidtransOrderID,
		"midtrans_redirect_url": transaction.MidtransRedirectURL,
		"created_at":           transaction.CreatedAt,
		"payment_at":           transaction.PaymentAt,
	}, nil
}

// HandlePaymentCallback processes Midtrans webhook notifications
func (uc *EnhancedTransactionUseCase) HandlePaymentCallback(ctx context.Context, notification map[string]interface{}) error {
	// Extract essential fields
	orderID, ok := notification["order_id"].(string)
	if !ok || orderID == "" {
		return fmt.Errorf("missing or invalid order_id in webhook")
	}

	transactionStatus, _ := notification["transaction_status"].(string)
	fraudStatus, _ := notification["fraud_status"].(string)
	statusCode, _ := notification["status_code"].(string)

	log.Printf("Processing webhook for order: %s, status: %s, fraud: %s", orderID, transactionStatus, fraudStatus)

	// Find transaction by Midtrans order ID
	transaction, err := uc.transactionRepo.GetByMidtransOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("transaction not found for order %s: %v", orderID, err)
	}

	// Map Midtrans status to our internal status
	newStatus := uc.mapMidtransStatus(transactionStatus, fraudStatus, statusCode)
	oldStatus := transaction.PaymentStatus

	// Only process if status changed
	if oldStatus == newStatus {
		log.Printf("Status unchanged for order %s: %s", orderID, newStatus)
		return nil
	}

	log.Printf("Payment status changing: %s -> %s for order %s", oldStatus, newStatus, orderID)

	// Update transaction status
	transaction.PaymentStatus = newStatus
	
	// Handle successful payment
	if newStatus == "success" {
		transaction.Status = "paid"
		now := time.Now()
		transaction.PaymentAt = &now
		
		log.Printf("Payment successful for order %s, triggering delivery", orderID)
		
		// Trigger instant delivery for instant transactions
		if transaction.DeliveryMethod == "instant" {
			// Use background context for webhook operations to avoid auth issues
			go uc.processInstantDeliveryFromWebhook(context.Background(), transaction)
		}

		// Send WebSocket notification to buyer/seller
		uc.notifyPaymentSuccess(ctx, transaction)
	}

	// Handle failed payment
	if newStatus == "failed" || newStatus == "expired" {
		transaction.Status = "payment_failed"
		log.Printf("Payment failed for order %s: %s", orderID, newStatus)
		
		// Send WebSocket notification
		uc.notifyPaymentFailure(ctx, transaction)
	}

	// Update transaction in database
	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return fmt.Errorf("failed to update transaction %s: %v", orderID, err)
	}

	log.Printf("Successfully processed webhook for order %s: %s -> %s", orderID, oldStatus, newStatus)
	return nil
}

// mapMidtransStatus maps Midtrans status to our internal status
func (uc *EnhancedTransactionUseCase) mapMidtransStatus(transactionStatus, fraudStatus, statusCode string) string {
	// Handle fraud status first
	if fraudStatus == "deny" {
		return "failed"
	}

	// Map transaction status
	switch transactionStatus {
	case "capture", "settlement":
		if fraudStatus == "accept" || fraudStatus == "" {
			return "success"
		}
		return "pending" // Wait for fraud review
	case "pending":
		return "pending"
	case "cancel", "deny", "expire":
		return "failed"
	case "refund", "partial_refund":
		return "refunded"
	default:
		log.Printf("Unknown transaction status: %s, defaulting to pending", transactionStatus)
		return "pending"
	}
}

// processInstantDeliveryFromWebhook processes instant delivery after successful payment
func (uc *EnhancedTransactionUseCase) processInstantDeliveryFromWebhook(ctx context.Context, transaction *entity.Transaction) {
	log.Printf("Processing instant delivery for transaction: %s, product: %s", transaction.ID, transaction.ProductID)
	
	// Get product to deliver credentials
	product, err := uc.productRepo.GetByID(ctx, transaction.ProductID)
	if err != nil {
		log.Printf("Failed to get product for delivery - ProductID: %s, Error: %v", transaction.ProductID, err)
		return
	}
	
	log.Printf("Product retrieved successfully: %s, hasCredentials: %t", product.Title, len(product.Credentials) > 0)

	// Update transaction status
	transaction.Status = "credentials_delivered"
	transaction.CredentialsDelivered = true
	now := time.Now()
	transaction.CredentialsDeliveredAt = &now

	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		log.Printf("Failed to update transaction after delivery: %v", err)
		return
	}
	
	// Update product sold count and status
	product.SoldCount++
	
	// Update product status based on stock
	if product.Stock > 0 && product.SoldCount >= product.Stock {
		product.Status = "sold_out"
		log.Printf("Product %s is now sold out (%d/%d)", product.ID, product.SoldCount, product.Stock)
	} else if product.Stock == 0 && len(product.Credentials) > 0 {
		// Single-use product with credentials - mark as sold
		product.Status = "sold"
		log.Printf("Single-use product %s is now sold", product.ID)
	}
	
	if err := uc.productRepo.Update(ctx, product); err != nil {
		log.Printf("Failed to update product status and sold count: %v", err)
		// Don't return error, transaction is still successful
	}

	// Send credentials via chat
	uc.sendCredentialsMessage(ctx, transaction, product)
	
	log.Printf("Instant delivery completed for transaction: %s", transaction.ID)
}

// notifyPaymentSuccess sends WebSocket notification for successful payment
func (uc *EnhancedTransactionUseCase) notifyPaymentSuccess(ctx context.Context, transaction *entity.Transaction) {
	if uc.wsManager == nil {
		log.Printf("WebSocket manager not available for payment notification")
		return
	}

	// Create payment success notification
	notification := map[string]interface{}{
		"type": "payment_status_update",
		"transaction": map[string]interface{}{
			"id":             transaction.ID,
			"status":         transaction.Status,
			"payment_status": transaction.PaymentStatus,
			"product_id":     transaction.ProductID,
			"buyer_id":       transaction.BuyerID,
			"seller_id":      transaction.SellerID,
			"total_amount":   transaction.TotalAmount,
			"payment_at":     transaction.PaymentAt,
		},
		"message": "🎉 Payment successful! Your order is being processed.",
		"timestamp": time.Now(),
	}

	// Send to buyer
	if notificationJSON, err := json.Marshal(notification); err == nil {
		log.Printf("Sending payment success notification to buyer: %s", transaction.BuyerID)
		uc.wsManager.SendToUser(transaction.BuyerID, notificationJSON)
	}

	// Send notification to seller as well  
	sellerNotification := make(map[string]interface{})
	for k, v := range notification {
		sellerNotification[k] = v
	}
	sellerNotification["message"] = "💰 Payment received! Order ready for delivery."
	if sellerNotificationJSON, err := json.Marshal(sellerNotification); err == nil {
		log.Printf("Sending payment notification to seller: %s", transaction.SellerID)
		uc.wsManager.SendToUser(transaction.SellerID, sellerNotificationJSON)
	}

	log.Printf("Payment success notification sent for transaction: %s", transaction.ID)
}

// notifyPaymentFailure sends WebSocket notification for failed payment
func (uc *EnhancedTransactionUseCase) notifyPaymentFailure(ctx context.Context, transaction *entity.Transaction) {
	// TODO: Implement WebSocket notification  
	log.Printf("Payment failure notification for transaction: %s", transaction.ID)
}

// sendCredentialsMessage sends credentials via chat system
func (uc *EnhancedTransactionUseCase) sendCredentialsMessage(ctx context.Context, transaction *entity.Transaction, product *entity.Product) {
	message := fmt.Sprintf("🎉 Payment confirmed! Here are your credentials:\n\n" +
		"**Product:** %s\n" +
		"**Credentials:** %s\n\n" +
		"Transaction ID: %s\n" +
		"Thank you for your purchase!", 
		product.Title, product.Credentials, transaction.ID)

	// For instant transactions, try to find or create a direct chat
	if uc.chatUseCase != nil {
		// Try to find existing chat between buyer and seller for this product
		chatID, err := uc.findOrCreateDirectChat(ctx, transaction.BuyerID, transaction.SellerID, product.ID)
		if err != nil {
			log.Printf("Failed to find/create chat for credentials delivery: %v", err)
			return
		}
		
		// Send system message with credentials
		_, err = uc.chatUseCase.SendSystemMessage(ctx, chatID, message, "credential_delivery", map[string]interface{}{
			"transaction_id": transaction.ID,
			"product_id": product.ID,
			"delivery_method": "instant",
		})
		if err != nil {
			log.Printf("Failed to send credentials message: %v", err)
		} else {
			log.Printf("Credentials delivered via chat %s for transaction %s", chatID, transaction.ID)
		}
	}
}

// findOrCreateDirectChat finds existing chat or creates a new direct chat between buyer and seller
func (uc *EnhancedTransactionUseCase) findOrCreateDirectChat(ctx context.Context, buyerID, sellerID, productID string) (string, error) {
	// Try to find existing direct chat between buyer and seller for this product
	// For simplicity, create a chat using the ChatUseCase CreateChatByProduct method
	input := CreateChatByProductInput{
		ProductID: productID,
		InitialMessage: "Direct chat created for credential delivery",
	}
	
	chatResp, err := uc.chatUseCase.CreateChatByProduct(ctx, buyerID, input)
	if err != nil {
		return "", fmt.Errorf("failed to create chat: %v", err)
	}
	
	return chatResp.Chat.ID, nil
}

// generateID creates a random ID
func (uc *EnhancedTransactionUseCase) generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}