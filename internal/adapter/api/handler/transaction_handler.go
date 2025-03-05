package handler

import (
	"github.com/labstack/echo/v4"

	"pasargamex/internal/usecase"
	"pasargamex/pkg/errors"
	"pasargamex/pkg/response"
	"pasargamex/pkg/utils"
)

type TransactionHandler struct {
	transactionUseCase *usecase.TransactionUseCase
}

func NewTransactionHandler(transactionUseCase *usecase.TransactionUseCase) *TransactionHandler {
	return &TransactionHandler{
		transactionUseCase: transactionUseCase,
	}
}

func SetupTransactionHandler(transactionUseCase *usecase.TransactionUseCase) {
	transactionHandler = NewTransactionHandler(transactionUseCase)
}

type resolveDisputeRequest struct {
	Resolution string `json:"resolution" validate:"required"`
	Refund     bool   `json:"refund"`
}

type createTransactionRequest struct {
	ProductID      string `json:"product_id" validate:"required"`
	DeliveryMethod string `json:"delivery_method" validate:"required,oneof=instant middleman"`
	Notes          string `json:"notes,omitempty"`
}

func (h *TransactionHandler) CreateTransaction(c echo.Context) error {
	// Parse request body
	var req createTransactionRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}
	
	// Get user ID from context
	userID := c.Get("uid").(string)
	
	// Call use case
	transaction, err := h.transactionUseCase.CreateTransaction(c.Request().Context(), userID, usecase.CreateTransactionInput{
		ProductID:      req.ProductID,
		DeliveryMethod: req.DeliveryMethod,
		Notes:          req.Notes,
	})
	
	if err != nil {
		return response.Error(c, err)
	}
	
	return response.Created(c, transaction)
}

func (h *TransactionHandler) GetTransaction(c echo.Context) error {
	// Get transaction ID from path
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}
	
	// Get user ID from context
	userID := c.Get("uid").(string)
	
	// Call use case
	transaction, err := h.transactionUseCase.GetTransactionByID(c.Request().Context(), userID, transactionID)
	if err != nil {
		return response.Error(c, err)
	}
	
	return response.Success(c, transaction)
}

func (h *TransactionHandler) ListTransactions(c echo.Context) error {
	// Parse query parameters
	role := c.QueryParam("role")     // "buyer" atau "seller"
	status := c.QueryParam("status") // Status transaksi
	
	// Get pagination parameters using the utility
	pagination := utils.GetPaginationParams(c)
	
	// Get user ID from context
	userID := c.Get("uid").(string)
	
	// Call use case with pagination parameters
	transactions, total, err := h.transactionUseCase.ListTransactions(
		c.Request().Context(),
		userID,
		role,
		status,
		pagination.Page,
		pagination.PageSize,
	)
	
	if err != nil {
		return response.Error(c, err)
	}
	
	return response.Paginated(c, transactions, total, pagination.Page, pagination.PageSize)
}

type processPaymentRequest struct {
	PaymentMethod  string                 `json:"payment_method" validate:"required"`
	PaymentDetails map[string]interface{} `json:"payment_details,omitempty"`
}

func (h *TransactionHandler) ProcessPayment(c echo.Context) error {
	// Get transaction ID from path
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}
	
	// Parse request body
	var req processPaymentRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}
	
	// Get user ID from context
	userID := c.Get("uid").(string)
	
	// Call use case
	transaction, err := h.transactionUseCase.ProcessPayment(
		c.Request().Context(),
		userID,
		transactionID,
		req.PaymentMethod,
		req.PaymentDetails,
	)
	
	if err != nil {
		return response.Error(c, err)
	}
	
	return response.Success(c, transaction)
}

func (h *TransactionHandler) GetTransactionLogs(c echo.Context) error {
	// Get transaction ID from path
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}
	
	// Get user ID from context
	userID := c.Get("uid").(string)
	
	// Call use case
	logs, err := h.transactionUseCase.GetTransactionLogs(c.Request().Context(), userID, transactionID)
	if err != nil {
		return response.Error(c, err)
	}
	
	return response.Success(c, logs)
}

func (h *TransactionHandler) ListAdminTransactions(c echo.Context) error {
	// Parse query parameters
	status := c.QueryParam("status")
	
	// Get pagination parameters using the utility
	pagination := utils.GetPaginationParams(c)
	
	// Get admin ID from context
	adminID := c.Get("uid").(string)
	
	// Create filter
	filter := make(map[string]interface{})
	if status != "" {
		filter["status"] = status
	}
	
	// Call use case with pagination parameters
	transactions, total, err := h.transactionUseCase.ListAdminTransactions(
		c.Request().Context(),
		adminID,
		filter,
		pagination.Page,
		pagination.PageSize,
	)
	
	if err != nil {
		return response.Error(c, err)
	}
	
	return response.Paginated(c, transactions, total, pagination.Page, pagination.PageSize)
}

func (h *TransactionHandler) ListPendingMiddlemanTransactions(c echo.Context) error {
	// Get pagination parameters using the utility
	pagination := utils.GetPaginationParams(c)
	
	// Get admin ID from context
	adminID := c.Get("uid").(string)
	
	// Call use case with pagination parameters
	transactions, total, err := h.transactionUseCase.ListPendingMiddlemanTransactions(
		c.Request().Context(),
		adminID,
		pagination.Page,
		pagination.PageSize,
	)
	
	if err != nil {
		return response.Error(c, err)
	}
	
	return response.Paginated(c, transactions, total, pagination.Page, pagination.PageSize)
}

// Untuk admin - assign middleman
func (h *TransactionHandler) AssignMiddleman(c echo.Context) error {
	// Get transaction ID from path
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}
	
	// Get admin ID from context
	adminID := c.Get("uid").(string)
	
	// Call use case
	transaction, err := h.transactionUseCase.AssignMiddleman(
		c.Request().Context(),
		adminID,
		transactionID,
	)
	
	if err != nil {
		return response.Error(c, err)
	}
	
	return response.Success(c, transaction)
}

type completeMiddlemanRequest struct {
	Credentials map[string]interface{} `json:"credentials" validate:"required"`
}

// Untuk admin - verifikasi dan selesaikan transaksi middleman
func (h *TransactionHandler) CompleteMiddleman(c echo.Context) error {
	// Get transaction ID from path
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}
	
	// Parse request body
	var req completeMiddlemanRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}
	
	// Get admin ID from context
	adminID := c.Get("uid").(string)
	
	// Call use case
	transaction, err := h.transactionUseCase.VerifyAndCompleteMiddleman(
		c.Request().Context(),
		adminID,
		transactionID,
		req.Credentials,
	)
	
	if err != nil {
		return response.Error(c, err)
	}
	
	return response.Success(c, transaction)
}

func (h *TransactionHandler) ConfirmDelivery(c echo.Context) error {
    // Get transaction ID from path
    transactionID := c.Param("id")
    if transactionID == "" {
        return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
    }
    
    // Get user ID from context
    userID := c.Get("uid").(string)
    
    // Call use case
    transaction, err := h.transactionUseCase.ConfirmDelivery(
        c.Request().Context(),
        userID,
        transactionID,
    )
    
    if err != nil {
        return response.Error(c, err)
    }
    
    return response.Success(c, transaction)
}

// CreateDispute handler for buyers/sellers to create a dispute
func (h *TransactionHandler) CreateDispute(c echo.Context) error {
    // Get transaction ID from path
    transactionID := c.Param("id")
    if transactionID == "" {
        return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
    }
    
    // Parse request body
    var req struct {
        Reason string `json:"reason" validate:"required"`
    }
    
    if err := c.Bind(&req); err != nil {
        return response.Error(c, err)
    }
    
    if err := c.Validate(&req); err != nil {
        return response.Error(c, err)
    }
    
    // Get user ID from context
    userID := c.Get("uid").(string)
    
    // Call use case
    transaction, err := h.transactionUseCase.CreateDispute(
        c.Request().Context(),
        userID,
        transactionID,
        req.Reason,
    )
    
    if err != nil {
        return response.Error(c, err)
    }
    
    return response.Success(c, transaction)
}

// ResolveDispute handler for admin to resolve a dispute
func (h *TransactionHandler) ResolveDispute(c echo.Context) error {
    // Get transaction ID from path
    transactionID := c.Param("id")
    if transactionID == "" {
        return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
    }
    
    // Parse request body
    var req struct {
        Resolution string `json:"resolution" validate:"required"`
        Refund     bool   `json:"refund"`
    }
    
    if err := c.Bind(&req); err != nil {
        return response.Error(c, err)
    }
    
    if err := c.Validate(&req); err != nil {
        return response.Error(c, err)
    }
    
    // Get admin ID from context
    adminID := c.Get("uid").(string)
    
    // Call use case
    transaction, err := h.transactionUseCase.ResolveDispute(
        c.Request().Context(),
        adminID,
        transactionID,
        req.Resolution,
        req.Refund,
    )
    
    if err != nil {
        return response.Error(c, err)
    }
    
    return response.Success(c, transaction)
}

// CancelTransaction handler for users/admin to cancel a transaction
func (h *TransactionHandler) CancelTransaction(c echo.Context) error {
    // Get transaction ID from path
    transactionID := c.Param("id")
    if transactionID == "" {
        return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
    }
    
    // Parse request body
    var req struct {
        Reason string `json:"reason" validate:"required"`
    }
    
    if err := c.Bind(&req); err != nil {
        return response.Error(c, err)
    }
    
    if err := c.Validate(&req); err != nil {
        return response.Error(c, err)
    }
    
    // Get user ID from context
    userID := c.Get("uid").(string)
    
    // Call use case
    transaction, err := h.transactionUseCase.CancelTransaction(
        c.Request().Context(),
        userID,
        transactionID,
        req.Reason,
    )
    
    if err != nil {
        return response.Error(c, err)
    }
    
    return response.Success(c, transaction)
}