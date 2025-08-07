package handler

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"

	"pasargamex/internal/domain/service"
	"pasargamex/internal/usecase"
	"pasargamex/pkg/errors"
	"pasargamex/pkg/response"
)

type PaymentHandler struct {
	enhancedTransactionUC *usecase.EnhancedTransactionUseCase
}

func NewPaymentHandler(enhancedTransactionUC *usecase.EnhancedTransactionUseCase) *PaymentHandler {
	return &PaymentHandler{
		enhancedTransactionUC: enhancedTransactionUC,
	}
}

type CreateSecureTransactionRequest struct {
	ProductID      string `json:"product_id" validate:"required"`
	DeliveryMethod string `json:"delivery_method" validate:"required,oneof=instant middleman"`
	PaymentMethod  string `json:"payment_method" validate:"required,oneof=midtrans_snap midtrans_bank_transfer wallet"`
	MiddlemanID    string `json:"middleman_id,omitempty"`
	Notes          string `json:"notes,omitempty"`
	
	// Customer details for payment
	CustomerFirstName string `json:"customer_first_name" validate:"required"`
	CustomerLastName  string `json:"customer_last_name,omitempty"`
	CustomerEmail     string `json:"customer_email" validate:"required,email"`
	CustomerPhone     string `json:"customer_phone" validate:"required"`
}

func (h *PaymentHandler) CreateSecureTransaction(c echo.Context) error {
	var req CreateSecureTransactionRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, errors.BadRequest("Invalid request body", err))
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, errors.BadRequest("Validation failed", err))
	}

	// Get user ID from JWT token
	userID, ok := c.Get("uid").(string)
	if !ok {
		return response.Error(c, errors.Unauthorized("User not authenticated", nil))
	}

	// Validate middleman requirement
	if req.DeliveryMethod == "middleman" && req.MiddlemanID == "" {
		return response.Error(c, errors.BadRequest("Middleman ID is required for middleman delivery", nil))
	}

	// Create transaction input
	input := usecase.CreateSecureTransactionInput{
		ProductID:      req.ProductID,
		DeliveryMethod: req.DeliveryMethod,
		PaymentMethod:  req.PaymentMethod,
		MiddlemanID:    req.MiddlemanID,
		Notes:          req.Notes,
		CustomerDetails: service.CustomerDetails{
			FirstName: req.CustomerFirstName,
			LastName:  req.CustomerLastName,
			Email:     req.CustomerEmail,
			Phone:     req.CustomerPhone,
		},
	}

	result, err := h.enhancedTransactionUC.CreateSecureTransaction(c.Request().Context(), userID, input)
	if err != nil {
		log.Printf("Failed to create secure transaction: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, result)
}

// MidtransCallback handles payment callbacks from Midtrans with security verification
func (h *PaymentHandler) MidtransCallback(c echo.Context) error {
	log.Printf("Received Midtrans webhook callback from IP: %s", c.RealIP())

	// Parse callback payload
	var notification map[string]interface{}
	if err := c.Bind(&notification); err != nil {
		log.Printf("Failed to parse Midtrans callback: %v", err)
		return response.Error(c, errors.BadRequest("Invalid callback payload", err))
	}

	// Log essential fields only (avoid logging sensitive data)
	orderID, _ := notification["order_id"].(string)
	transactionStatus, _ := notification["transaction_status"].(string)
	paymentType, _ := notification["payment_type"].(string)
	fraudStatus, _ := notification["fraud_status"].(string)
	
	log.Printf("Midtrans webhook: OrderID=%s, Status=%s, PaymentType=%s, FraudStatus=%s", 
		orderID, transactionStatus, paymentType, fraudStatus)

	// Verify webhook authenticity (security best practice)
	if err := h.verifyMidtransSignature(c, notification); err != nil {
		log.Printf("Midtrans webhook signature verification failed: %v", err)
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"status": "UNAUTHORIZED",
		})
	}

	// Process callback with enhanced error handling
	err := h.enhancedTransactionUC.HandlePaymentCallback(c.Request().Context(), notification)
	if err != nil {
		log.Printf("Failed to process Midtrans callback for order %s: %v", orderID, err)
		// Still return 200 to Midtrans to avoid retries
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ERROR_PROCESSED",
		})
	}

	log.Printf("Successfully processed Midtrans webhook for order: %s", orderID)
	
	// Return OK to Midtrans (must be 200 to confirm receipt)
	return c.JSON(http.StatusOK, map[string]string{
		"status": "OK",
	})
}

// GetPaymentStatus gets current payment status for a transaction
func (h *PaymentHandler) GetPaymentStatus(c echo.Context) error {
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}

	// Get user ID from JWT token
	userID, ok := c.Get("uid").(string)
	if !ok {
		return response.Error(c, errors.Unauthorized("User not authenticated", nil))
	}

	log.Printf("HANDLER: GetPaymentStatus called for transaction: %s, user: %s", transactionID, userID)

	// Get payment status from enhanced transaction use case
	status, err := h.enhancedTransactionUC.GetPaymentStatus(c.Request().Context(), transactionID, userID)
	if err != nil {
		log.Printf("Failed to get payment status: %v", err)
		return response.Error(c, errors.Internal("Failed to get payment status", err))
	}

	return response.Success(c, status)
}

// CreateInstantTransaction creates an instant delivery transaction (simplified endpoint for chat UI)
func (h *PaymentHandler) CreateInstantTransaction(c echo.Context) error {
	var req struct {
		ProductID      string `json:"product_id" validate:"required"`
		DeliveryMethod string `json:"delivery_method" validate:"required"`
		PaymentMethod  string `json:"payment_method" validate:"required"`
		Embed          bool   `json:"embed" default:"false"`
	}
	
	if err := c.Bind(&req); err != nil {
		return response.Error(c, errors.BadRequest("Invalid request body", err))
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, errors.BadRequest("Validation failed", err))
	}

	// Get user ID from JWT token
	userID, ok := c.Get("uid").(string)
	if !ok {
		return response.Error(c, errors.Unauthorized("User not authenticated", nil))
	}

	// Create simplified transaction for instant delivery
	input := usecase.CreateSecureTransactionInput{
		ProductID:      req.ProductID,
		DeliveryMethod: req.DeliveryMethod,
		PaymentMethod:  req.PaymentMethod,
		Embed:          req.Embed,
		CustomerDetails: service.CustomerDetails{
			FirstName: "Buyer", // Simplified for chat UI
			Email:     "buyer@example.com",
			Phone:     "08123456789",
		},
	}

	result, err := h.enhancedTransactionUC.CreateSecureTransaction(c.Request().Context(), userID, input)
	if err != nil {
		log.Printf("Failed to create instant transaction: %v", err)
		log.Printf("Error type: %T", err)
		return response.Error(c, err)
	}

	return response.Success(c, result)
}

// verifyMidtransSignature verifies the authenticity of Midtrans webhook
func (h *PaymentHandler) verifyMidtransSignature(c echo.Context, notification map[string]interface{}) error {
	// Get signature from header or notification
	var signatureKey string
	if sig := c.Request().Header.Get("X-Midtrans-Signature"); sig != "" {
		signatureKey = sig
	} else if sig, ok := notification["signature_key"].(string); ok {
		signatureKey = sig
	} else {
		// For sandbox environment, signature might not be present
		environment := os.Getenv("MIDTRANS_ENVIRONMENT")
		if environment == "sandbox" {
			log.Printf("Signature not found but running in sandbox mode, allowing webhook")
			return nil
		}
		return fmt.Errorf("no signature found in webhook")
	}

	// Extract required fields for signature verification
	orderID, _ := notification["order_id"].(string)
	statusCode, _ := notification["status_code"].(string)
	grossAmount, _ := notification["gross_amount"].(string)
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	
	if orderID == "" || statusCode == "" || grossAmount == "" {
		return fmt.Errorf("missing required fields for signature verification")
	}

	log.Printf("Verifying webhook signature for order: %s", orderID)
	
	// Create signature hash: SHA512(order_id+status_code+gross_amount+server_key)
	data := orderID + statusCode + grossAmount + serverKey
	hash := sha512.Sum512([]byte(data))
	expectedSignature := hex.EncodeToString(hash[:])
	
	if signatureKey != expectedSignature {
		environment := os.Getenv("MIDTRANS_ENVIRONMENT")
		if environment == "sandbox" {
			log.Printf("Signature mismatch in sandbox mode, but allowing webhook to proceed")
			log.Printf("Expected: %s, Got: %s", expectedSignature, signatureKey)
			return nil
		}
		return fmt.Errorf("signature verification failed")
	}
	
	log.Printf("Webhook signature verified successfully for order: %s", orderID)
	return nil
}
