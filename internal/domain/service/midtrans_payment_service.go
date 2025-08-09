package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// MidtransPaymentService - Real Midtrans implementation using HTTP API
type MidtransPaymentService struct {
	serverKey    string
	clientKey    string
	isProduction bool
	baseURL      string
}

func NewMidtransPaymentService(serverKey, clientKey string, isProduction bool) *MidtransPaymentService {
	baseURL := "https://app.sandbox.midtrans.com/snap/v1"
	if isProduction {
		baseURL = "https://app.midtrans.com/snap/v1"
	}

	return &MidtransPaymentService{
		serverKey:    serverKey,
		clientKey:    clientKey,
		isProduction: isProduction,
		baseURL:      baseURL,
	}
}

// MidtransSnapRequest represents Midtrans Snap API request
type MidtransSnapRequest struct {
	TransactionDetails TransactionDetails   `json:"transaction_details"`
	CustomerDetails    CustomerDetails      `json:"customer_details"`
	ItemDetails        []MidtransItemDetail `json:"item_details"`
	Callbacks          *Callbacks           `json:"callbacks,omitempty"`
}

// MidtransItemDetail represents item detail for Midtrans API
type MidtransItemDetail struct {
	ID       string  `json:"id"`
	Price    float64 `json:"price"`
	Quantity int32   `json:"quantity"`
	Name     string  `json:"name"`
	Category string  `json:"category,omitempty"`
}

// TransactionDetails for Midtrans
type TransactionDetails struct {
	OrderID     string  `json:"order_id"`
	GrossAmount float64 `json:"gross_amount"`
}

// Callbacks for Midtrans
type Callbacks struct {
	Finish  string `json:"finish"`
	Error   string `json:"error"`
	Pending string `json:"pending"`
}

// MidtransSnapResponse represents Midtrans Snap API response
type MidtransSnapResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

func (mps *MidtransPaymentService) CreatePayment(ctx context.Context, req PaymentGatewayRequest) (*PaymentGatewayResponse, error) {
	log.Printf("Creating Midtrans payment for order: %s, amount: %.0f", req.OrderID, req.Amount)

	// Debug: Log item details
	log.Printf("Item details count: %d", len(req.ItemDetails))
	totalItemsAmount := 0.0
	for i, item := range req.ItemDetails {
		log.Printf("Item %d: ID=%s, Name=%s, Price=%.2f, Quantity=%d", i, item.ID, item.Name, item.Price, item.Quantity)
		totalItemsAmount += item.Price * float64(item.Quantity)
	}
	log.Printf("Total items amount: %.2f, Gross amount: %.2f", totalItemsAmount, req.Amount)

	// Convert ItemDetails to MidtransItemDetail
	var midtransItems []MidtransItemDetail
	for _, item := range req.ItemDetails {
		midtransItems = append(midtransItems, MidtransItemDetail{
			ID:       item.ID,
			Price:    item.Price,
			Quantity: item.Quantity,
			Name:     item.Name,
			Category: item.Category,
		})
	}

	// Create Midtrans Snap request
	snapReq := MidtransSnapRequest{
		TransactionDetails: TransactionDetails{
			OrderID:     req.OrderID,
			GrossAmount: req.Amount,
		},
		CustomerDetails: req.CustomerDetails,
		ItemDetails:     midtransItems,
	}

	// Only add callbacks for redirect mode, not for embed mode
	if !req.Embed {
		// Use environment-specific callback URLs - these are called AFTER payment completion
		baseURL := os.Getenv("BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8080" // fallback for local development
		}

		// For production, use local development URL for frontend since HTML files are not deployed
		frontendBaseURL := baseURL
		if os.Getenv("ENVIRONMENT") == "production" {
			frontendBaseURL = "http://127.0.0.1:5500" // Point back to local for frontend files
		}

		snapReq.Callbacks = &Callbacks{
			Finish:  frontendBaseURL + "/websocket-chat-pgx/payment-success.html",
			Error:   frontendBaseURL + "/websocket-chat-pgx/payment-error.html",
			Pending: frontendBaseURL + "/websocket-chat-pgx/payment-pending.html",
		}
	}

	// Convert to JSON
	jsonData, err := json.Marshal(snapReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", mps.baseURL+"/transactions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	authHeader := base64.StdEncoding.EncodeToString([]byte(mps.serverKey + ":"))
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Basic "+authHeader)

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		log.Printf("Midtrans API error: %s", string(body))
		return nil, fmt.Errorf("midtrans API error: %s", string(body))
	}

	// Parse response
	var snapResp MidtransSnapResponse
	if err := json.Unmarshal(body, &snapResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	response := &PaymentGatewayResponse{
		Token:       snapResp.Token,
		RedirectURL: snapResp.RedirectURL,
		OrderID:     req.OrderID,
		Status:      "pending",
		PaymentType: req.PaymentType,
	}

	log.Printf("Midtrans payment created successfully: %s", response.Token)
	return response, nil
}

func (mps *MidtransPaymentService) GetPaymentStatus(ctx context.Context, orderID string) (*PaymentGatewayResponse, error) {
	log.Printf("Getting Midtrans payment status for order: %s", orderID)

	// Debug environment
	log.Printf("Midtrans environment: sandbox=%t", !mps.isProduction)

	// Create HTTP request to check transaction status
	statusURL := fmt.Sprintf("https://api.sandbox.midtrans.com/v2/%s/status", orderID)
	if mps.isProduction {
		statusURL = fmt.Sprintf("https://api.midtrans.com/v2/%s/status", orderID)
	}

	log.Printf("Status URL: %s", statusURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", statusURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	authHeader := base64.StdEncoding.EncodeToString([]byte(mps.serverKey + ":"))
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Basic "+authHeader)

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	log.Printf("Midtrans status API response: Status=%d, Body=%s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		log.Printf("Midtrans status API error: %s", string(body))
		return nil, fmt.Errorf("midtrans status API error: %s", string(body))
	}

	// Parse response
	var statusResp map[string]interface{}
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Extract status
	transactionStatus, _ := statusResp["transaction_status"].(string)
	paymentType, _ := statusResp["payment_type"].(string)

	log.Printf("Parsed transaction_status: '%s', payment_type: '%s'", transactionStatus, paymentType)

	// Map Midtrans status to our internal status
	status := "pending"
	switch transactionStatus {
	case "settlement", "capture":
		status = "success"
		log.Printf("Status mapped to SUCCESS")
	case "cancel", "deny", "expire":
		status = "failure"
		log.Printf("Status mapped to FAILURE")
	case "pending":
		status = "pending"
		log.Printf("Status mapped to PENDING")
	default:
		log.Printf("Unknown transaction_status: '%s', defaulting to pending", transactionStatus)
	}

	response := &PaymentGatewayResponse{
		OrderID:     orderID,
		Status:      status,
		PaymentType: paymentType,
	}

	log.Printf("Payment status retrieved: %s -> %s", orderID, status)
	return response, nil
}

func (mps *MidtransPaymentService) HandleCallback(ctx context.Context, notification map[string]interface{}) (*PaymentGatewayResponse, error) {
	log.Printf("Handling Midtrans callback: %+v", notification)

	orderID, ok := notification["order_id"].(string)
	if !ok {
		return nil, fmt.Errorf("order_id not found in notification")
	}

	transactionStatus, ok := notification["transaction_status"].(string)
	if !ok {
		transactionStatus = "pending"
	}

	paymentType, _ := notification["payment_type"].(string)

	// Map Midtrans status to our internal status
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
