package service

import (
	"context"
	"fmt"
	"log"
	"time"
)

// PaymentGatewayRequest represents a payment request
type PaymentGatewayRequest struct {
	OrderID       string
	Amount        float64
	PaymentType   string // "bank_transfer", "credit_card", "gopay", etc
	Embed         bool   // true = embedded/popup, false = redirect (Midtrans only)
	CustomerDetails CustomerDetails
	ItemDetails   []ItemDetail
}

// CustomerDetails represents customer information
type CustomerDetails struct {
	FirstName string
	LastName  string
	Email     string
	Phone     string
}

// ItemDetail represents an item in the transaction
type ItemDetail struct {
	ID       string
	Price    float64
	Quantity int32
	Name     string
	Category string
}

// PaymentGatewayResponse represents a payment response
type PaymentGatewayResponse struct {
	Token       string
	RedirectURL string
	OrderID     string
	Status      string
	PaymentType string
	VaNumbers   []VaNumber
}

// VaNumber represents virtual account number
type VaNumber struct {
	Bank     string
	VaNumber string
}

// PaymentGatewayService interface for payment operations
type PaymentGatewayService interface {
	CreatePayment(ctx context.Context, req PaymentGatewayRequest) (*PaymentGatewayResponse, error)
	GetPaymentStatus(ctx context.Context, orderID string) (*PaymentGatewayResponse, error)
	HandleCallback(ctx context.Context, notification map[string]interface{}) (*PaymentGatewayResponse, error)
}

// SimplifiedPaymentService - Basic implementation for testing
type SimplifiedPaymentService struct {
	serverKey    string
	clientKey    string
	isProduction bool
}

func NewSimplifiedPaymentService(serverKey, clientKey string, isProduction bool) *SimplifiedPaymentService {
	return &SimplifiedPaymentService{
		serverKey:    serverKey,
		clientKey:    clientKey,
		isProduction: isProduction,
	}
}

func (sps *SimplifiedPaymentService) CreatePayment(ctx context.Context, req PaymentGatewayRequest) (*PaymentGatewayResponse, error) {
	log.Printf("Creating simplified payment for order: %s, amount: %.0f", req.OrderID, req.Amount)

	// For testing - simulate payment creation
	response := &PaymentGatewayResponse{
		Token:       fmt.Sprintf("test-token-%s-%d", req.OrderID, time.Now().Unix()),
		RedirectURL: fmt.Sprintf("https://app.sandbox.midtrans.com/snap/v1/transactions/%s/pay", req.OrderID),
		OrderID:     req.OrderID,
		Status:      "pending",
		PaymentType: req.PaymentType,
	}

	// Simulate VA numbers for bank transfer
	if req.PaymentType == "bank_transfer" || req.PaymentType == "midtrans_bank_transfer" {
		response.VaNumbers = []VaNumber{
			{
				Bank:     "bca",
				VaNumber: fmt.Sprintf("12345%d", time.Now().Unix()%100000),
			},
			{
				Bank:     "bni",
				VaNumber: fmt.Sprintf("98765%d", time.Now().Unix()%100000),
			},
		}
	}

	log.Printf("Simplified payment created successfully: %s", response.Token)
	return response, nil
}

func (sps *SimplifiedPaymentService) GetPaymentStatus(ctx context.Context, orderID string) (*PaymentGatewayResponse, error) {
	log.Printf("Getting payment status for order: %s", orderID)

	// For testing - simulate status check
	response := &PaymentGatewayResponse{
		OrderID:     orderID,
		Status:      "pending", // In real implementation, this would come from Midtrans
		PaymentType: "bank_transfer",
	}

	log.Printf("Payment status retrieved: %s", response.Status)
	return response, nil
}

func (sps *SimplifiedPaymentService) HandleCallback(ctx context.Context, notification map[string]interface{}) (*PaymentGatewayResponse, error) {
	log.Printf("Handling simplified callback: %+v", notification)

	orderID, ok := notification["order_id"].(string)
	if !ok {
		return nil, fmt.Errorf("order_id not found in notification")
	}

	transactionStatus, ok := notification["transaction_status"].(string)
	if !ok {
		transactionStatus = "pending"
	}

	paymentType, _ := notification["payment_type"].(string)

	// Determine final status
	finalStatus := "pending"
	switch transactionStatus {
	case "settlement", "capture":
		finalStatus = "success"
	case "cancel", "deny", "expire":
		finalStatus = "failure"
	case "pending":
		finalStatus = "pending"
	}

	response := &PaymentGatewayResponse{
		OrderID:     orderID,
		Status:      finalStatus,
		PaymentType: paymentType,
	}

	log.Printf("Callback processed: %s -> %s", orderID, finalStatus)
	return response, nil
}