package handler

import (
	"log"

	"github.com/labstack/echo/v4"

	"pasargamex/internal/usecase"
	"pasargamex/pkg/errors"
	"pasargamex/pkg/response"
)

type EscrowHandler struct {
	escrowManagerUC *usecase.EscrowManagerUseCase
}

func NewEscrowHandler(escrowManagerUC *usecase.EscrowManagerUseCase) *EscrowHandler {
	return &EscrowHandler{
		escrowManagerUC: escrowManagerUC,
	}
}

type DeliverCredentialsRequest struct {
	TransactionID string                 `json:"transaction_id" validate:"required"`
	Credentials   map[string]interface{} `json:"credentials" validate:"required"`
}

type ConfirmCredentialsRequest struct {
	TransactionID string `json:"transaction_id" validate:"required"`
	IsWorking     bool   `json:"is_working" validate:"required"`
	Notes         string `json:"notes,omitempty"`
}

// DeliverCredentials - Seller delivers account credentials
func (h *EscrowHandler) DeliverCredentials(c echo.Context) error {
	var req DeliverCredentialsRequest
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

	// Validate credentials content
	if len(req.Credentials) == 0 {
		return response.Error(c, errors.BadRequest("Credentials cannot be empty", nil))
	}

	// Common gaming credentials validation
	requiredFields := []string{"username", "password"}
	for _, field := range requiredFields {
		if _, exists := req.Credentials[field]; !exists {
			return response.Error(c, errors.BadRequest("Missing required credential: "+field, nil))
		}
		
		if str, ok := req.Credentials[field].(string); !ok || str == "" {
			return response.Error(c, errors.BadRequest("Invalid credential value: "+field, nil))
		}
	}

	err := h.escrowManagerUC.DeliverCredentials(c.Request().Context(), req.TransactionID, userID, req.Credentials)
	if err != nil {
		log.Printf("Failed to deliver credentials: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, map[string]interface{}{
		"message": "Credentials delivered successfully",
		"transaction_id": req.TransactionID,
		"auto_release_in": "24 hours",
	})
}

// ConfirmCredentials - Buyer confirms credentials are working
func (h *EscrowHandler) ConfirmCredentials(c echo.Context) error {
	var req ConfirmCredentialsRequest
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

	// If not working, notes are required
	if !req.IsWorking && req.Notes == "" {
		return response.Error(c, errors.BadRequest("Notes are required when reporting non-working credentials", nil))
	}

	err := h.escrowManagerUC.ConfirmCredentials(c.Request().Context(), req.TransactionID, userID, req.IsWorking, req.Notes)
	if err != nil {
		log.Printf("Failed to confirm credentials: %v", err)
		return response.Error(c, err)
	}

	var message string
	if req.IsWorking {
		message = "Credentials confirmed as working. Funds released to seller."
	} else {
		message = "Credentials reported as not working. Dispute created for admin review."
	}

	return response.Success(c, map[string]interface{}{
		"message": message,
		"transaction_id": req.TransactionID,
		"status": req.IsWorking,
	})
}

// GetTransactionCredentials - Get delivered credentials (buyer only)
func (h *EscrowHandler) GetTransactionCredentials(c echo.Context) error {
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}

	// Get user ID from JWT token
	userID, ok := c.Get("uid").(string)
	if !ok {
		return response.Error(c, errors.Unauthorized("User not authenticated", nil))
	}

	// TODO: Implement get credentials logic
	// This would involve:
	// 1. Get transaction from repository
	// 2. Verify user is the buyer
	// 3. Verify credentials have been delivered
	// 4. Return credentials (securely)

	return response.Success(c, map[string]interface{}{
		"message": "TODO: Implement get credentials",
		"transaction_id": transactionID,
		"user_id": userID,
	})
}