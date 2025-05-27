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

type createTransactionRequest struct {
	ProductID      string `json:"product_id" validate:"required"`
	DeliveryMethod string `json:"delivery_method" validate:"required,oneof=instant middleman"`
	Notes          string `json:"notes,omitempty"`
}

func (h *TransactionHandler) CreateTransaction(c echo.Context) error {
	var req createTransactionRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	userID := c.Get("uid").(string)

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
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}

	userID := c.Get("uid").(string)

	transaction, err := h.transactionUseCase.GetTransactionByID(c.Request().Context(), userID, transactionID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, transaction)
}

func (h *TransactionHandler) ListTransactions(c echo.Context) error {
	role := c.QueryParam("role")
	status := c.QueryParam("status")

	pagination := utils.GetPaginationParams(c)

	userID := c.Get("uid").(string)

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

// Modified: ProcessPayment handler
func (h *TransactionHandler) ProcessPayment(c echo.Context) error {
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}

	var req processPaymentRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	userID := c.Get("uid").(string)

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

// New: ConfirmMiddlemanPayment handler
func (h *TransactionHandler) ConfirmMiddlemanPayment(c echo.Context) error {
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}

	adminID := c.Get("uid").(string)

	transaction, err := h.transactionUseCase.ConfirmMiddlemanPayment(
		c.Request().Context(),
		adminID,
		transactionID,
	)

	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, transaction)
}

func (h *TransactionHandler) GetTransactionLogs(c echo.Context) error {
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}

	userID := c.Get("uid").(string)

	logs, err := h.transactionUseCase.GetTransactionLogs(c.Request().Context(), userID, transactionID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, logs)
}

func (h *TransactionHandler) ListAdminTransactions(c echo.Context) error {
	status := c.QueryParam("status")

	pagination := utils.GetPaginationParams(c)

	adminID := c.Get("uid").(string)

	filter := make(map[string]interface{})
	if status != "" {
		filter["status"] = status
	}

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
	pagination := utils.GetPaginationParams(c)

	adminID := c.Get("uid").(string)

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

func (h *TransactionHandler) AssignMiddleman(c echo.Context) error {
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}

	adminID := c.Get("uid").(string)

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

func (h *TransactionHandler) CompleteMiddleman(c echo.Context) error {
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}

	var req completeMiddlemanRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	adminID := c.Get("uid").(string)

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
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}

	userID := c.Get("uid").(string)

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

func (h *TransactionHandler) CreateDispute(c echo.Context) error {
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}

	var req struct {
		Reason string `json:"reason" validate:"required"`
	}

	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	userID := c.Get("uid").(string)

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

func (h *TransactionHandler) ResolveDispute(c echo.Context) error {
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}

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

	adminID := c.Get("uid").(string)

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

func (h *TransactionHandler) CancelTransaction(c echo.Context) error {
	transactionID := c.Param("id")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}

	var req struct {
		Reason string `json:"reason" validate:"required"`
	}

	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	userID := c.Get("uid").(string)

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
