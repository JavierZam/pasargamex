package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"pasargamex/internal/usecase"
	"pasargamex/pkg/response"
	"pasargamex/pkg/utils"
)

type WalletHandler struct {
	walletUseCase *usecase.WalletUseCase
}

func NewWalletHandler(walletUseCase *usecase.WalletUseCase) *WalletHandler {
	return &WalletHandler{
		walletUseCase: walletUseCase,
	}
}

// Helper function for safe user ID extraction
func getUserID(c echo.Context) (string, error) {
	userID, ok := c.Get("uid").(string)
	if !ok || userID == "" {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "Invalid session")
	}
	return userID, nil
}

type createWalletRequest struct {
	Currency string `json:"currency" validate:"omitempty,oneof=IDR"`
}

type topupWalletRequest struct {
	Amount          float64 `json:"amount" validate:"required,min=10000,max=100000000"` // Min 10k IDR, Max 100M IDR
	PaymentMethodID string  `json:"payment_method_id" validate:"required"`
}

type withdrawWalletRequest struct {
	Amount          float64 `json:"amount" validate:"required,min=10000,max=50000000"` // Min 10k IDR, Max 50M IDR (consistent with topup)
	PaymentMethodID string  `json:"payment_method_id" validate:"required"`
}

type createPaymentMethodRequest struct {
	Type          string                 `json:"type" validate:"required,oneof=bank_transfer ewallet credit_card crypto"`
	Provider      string                 `json:"provider" validate:"required"`
	AccountNumber string                 `json:"account_number" validate:"required"`
	AccountName   string                 `json:"account_name" validate:"required"`
	IsDefault     bool                   `json:"is_default"`
	Details       map[string]interface{} `json:"details,omitempty"`
}

type updatePaymentMethodRequest struct {
	AccountNumber string                 `json:"account_number,omitempty"`
	AccountName   string                 `json:"account_name,omitempty"`
	IsDefault     bool                   `json:"is_default"`
	Details       map[string]interface{} `json:"details,omitempty"`
}

type processTopupRequest struct {
	Approve bool   `json:"approve"`
	Notes   string `json:"notes,omitempty"`
}

type processWithdrawRequest struct {
	Approve bool   `json:"approve"`
	Notes   string `json:"notes,omitempty"`
}

// Wallet Management
func (h *WalletHandler) CreateWallet(c echo.Context) error {
	var req createWalletRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error binding request: %v", err)
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		log.Printf("Validation error: %v", err)
		return response.Error(c, err)
	}

	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	input := usecase.CreateWalletInput{
		UserID:   userID,
		Currency: req.Currency,
	}

	wallet, err := h.walletUseCase.CreateWallet(c.Request().Context(), input)
	if err != nil {
		log.Printf("Error creating wallet: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, wallet)
}

func (h *WalletHandler) GetWallet(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	wallet, err := h.walletUseCase.GetWalletByUserID(c.Request().Context(), userID)
	if err != nil {
		log.Printf("Error getting wallet: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, wallet)
}

func (h *WalletHandler) GetWalletTransactions(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	// Parse pagination parameters
	pageStr := c.QueryParam("page")
	limitStr := c.QueryParam("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	pagination := &utils.Pagination{
		Page:  page,
		Limit: limit,
	}

	transactions, err := h.walletUseCase.GetWalletTransactions(c.Request().Context(), userID, pagination)
	if err != nil {
		log.Printf("Error getting wallet transactions: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, transactions)
}

// Payment Methods
func (h *WalletHandler) CreatePaymentMethod(c echo.Context) error {
	var req createPaymentMethodRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error binding request: %v", err)
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		log.Printf("Validation error: %v", err)
		return response.Error(c, err)
	}

	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	input := usecase.CreatePaymentMethodInput{
		Type:          req.Type,
		Provider:      req.Provider,
		AccountNumber: req.AccountNumber,
		AccountName:   req.AccountName,
		IsDefault:     req.IsDefault,
		Details:       req.Details,
	}

	paymentMethod, err := h.walletUseCase.CreatePaymentMethod(c.Request().Context(), userID, input)
	if err != nil {
		log.Printf("Error creating payment method: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, paymentMethod)
}

func (h *WalletHandler) GetPaymentMethods(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	paymentMethods, err := h.walletUseCase.GetPaymentMethods(c.Request().Context(), userID)
	if err != nil {
		log.Printf("Error getting payment methods: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, paymentMethods)
}

func (h *WalletHandler) UpdatePaymentMethod(c echo.Context) error {
	var req updatePaymentMethodRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error binding request: %v", err)
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		log.Printf("Validation error: %v", err)
		return response.Error(c, err)
	}

	userID, err := getUserID(c)
	if err != nil {
		return err
	}
	paymentMethodID := c.Param("id")

	input := usecase.UpdatePaymentMethodInput{
		AccountNumber: req.AccountNumber,
		AccountName:   req.AccountName,
		IsDefault:     req.IsDefault,
		Details:       req.Details,
	}

	paymentMethod, err := h.walletUseCase.UpdatePaymentMethod(c.Request().Context(), userID, paymentMethodID, input)
	if err != nil {
		log.Printf("Error updating payment method: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, paymentMethod)
}

func (h *WalletHandler) DeletePaymentMethod(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}
	paymentMethodID := c.Param("id")

	err = h.walletUseCase.DeletePaymentMethod(c.Request().Context(), userID, paymentMethodID)
	if err != nil {
		log.Printf("Error deleting payment method: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, nil)
}

// Topup
func (h *WalletHandler) CreateTopupRequest(c echo.Context) error {
	var req topupWalletRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error binding request: %v", err)
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		log.Printf("Validation error: %v", err)
		return response.Error(c, err)
	}

	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	input := usecase.TopupWalletInput{
		Amount:          req.Amount,
		PaymentMethodID: req.PaymentMethodID,
	}

	topupRequest, err := h.walletUseCase.CreateTopupRequest(c.Request().Context(), userID, input)
	if err != nil {
		log.Printf("Error creating topup request: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, topupRequest)
}

func (h *WalletHandler) GetTopupRequests(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	// Parse pagination parameters
	pageStr := c.QueryParam("page")
	limitStr := c.QueryParam("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	pagination := &utils.Pagination{
		Page:  page,
		Limit: limit,
	}

	topupRequests, err := h.walletUseCase.GetTopupRequests(c.Request().Context(), userID, pagination)
	if err != nil {
		log.Printf("Error getting topup requests: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, topupRequests)
}

func (h *WalletHandler) ProcessTopupRequest(c echo.Context) error {
	var req processTopupRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error binding request: %v", err)
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		log.Printf("Validation error: %v", err)
		return response.Error(c, err)
	}

	adminID, err := getUserID(c)
	if err != nil {
		return err
	}
	topupID := c.Param("id")

	topupRequest, err := h.walletUseCase.ProcessTopupRequest(c.Request().Context(), topupID, adminID, req.Approve, req.Notes)
	if err != nil {
		log.Printf("Error processing topup request: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, topupRequest)
}

// Withdraw
func (h *WalletHandler) CreateWithdrawRequest(c echo.Context) error {
	var req withdrawWalletRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error binding request: %v", err)
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		log.Printf("Validation error: %v", err)
		return response.Error(c, err)
	}

	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	input := usecase.WithdrawWalletInput{
		Amount:          req.Amount,
		PaymentMethodID: req.PaymentMethodID,
	}

	withdrawRequest, err := h.walletUseCase.CreateWithdrawRequest(c.Request().Context(), userID, input)
	if err != nil {
		log.Printf("Error creating withdraw request: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, withdrawRequest)
}

func (h *WalletHandler) GetWithdrawRequests(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	// Parse pagination parameters
	pageStr := c.QueryParam("page")
	limitStr := c.QueryParam("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	pagination := &utils.Pagination{
		Page:  page,
		Limit: limit,
	}

	withdrawRequests, err := h.walletUseCase.GetWithdrawRequests(c.Request().Context(), userID, pagination)
	if err != nil {
		log.Printf("Error getting withdraw requests: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, withdrawRequests)
}

func (h *WalletHandler) ProcessWithdrawRequest(c echo.Context) error {
	var req processWithdrawRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Error binding request: %v", err)
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		log.Printf("Validation error: %v", err)
		return response.Error(c, err)
	}

	adminID, err := getUserID(c)
	if err != nil {
		return err
	}
	withdrawID := c.Param("id")

	withdrawRequest, err := h.walletUseCase.ProcessWithdrawRequest(c.Request().Context(), withdrawID, adminID, req.Approve, req.Notes)
	if err != nil {
		log.Printf("Error processing withdraw request: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, withdrawRequest)
}

// Admin endpoints for monitoring
func (h *WalletHandler) GetPendingTopupRequests(c echo.Context) error {
	// Parse pagination parameters
	pageStr := c.QueryParam("page")
	limitStr := c.QueryParam("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	pagination := &utils.Pagination{
		Page:  page,
		Limit: limit,
	}

	pendingTopups, err := h.walletUseCase.GetPendingTopupRequests(c.Request().Context(), pagination)
	if err != nil {
		log.Printf("Error getting pending topup requests: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, pendingTopups)
}

func (h *WalletHandler) GetPendingWithdrawRequests(c echo.Context) error {
	// Parse pagination parameters
	pageStr := c.QueryParam("page")
	limitStr := c.QueryParam("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	pagination := &utils.Pagination{
		Page:  page,
		Limit: limit,
	}

	pendingWithdraws, err := h.walletUseCase.GetPendingWithdrawRequests(c.Request().Context(), pagination)
	if err != nil {
		log.Printf("Error getting pending withdraw requests: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, pendingWithdraws)
}

func (h *WalletHandler) GetWalletStatistics(c echo.Context) error {
	stats, err := h.walletUseCase.GetWalletStatistics(c.Request().Context())
	if err != nil {
		log.Printf("Error getting wallet statistics: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, stats)
}