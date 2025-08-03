package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"pasargamex/internal/usecase"
	"pasargamex/pkg/errors"
	"pasargamex/pkg/response"
)

type ChatHandler struct {
	chatUseCase *usecase.ChatUseCase
}

func NewChatHandler(chatUseCase *usecase.ChatUseCase) *ChatHandler {
	return &ChatHandler{
		chatUseCase: chatUseCase,
	}
}

type createChatRequest struct {
	RecipientID    string `json:"recipient_id" validate:"required"`
	ProductID      string `json:"product_id"`
	InitialMessage string `json:"initial_message"`
}

// Updated: Added ProductID, AttachmentURL, Metadata
type sendMessageRequest struct {
	Content        string                 `json:"content" validate:"required"`
	Type           string                 `json:"type" validate:"required,oneof=text image system offer"`
	AttachmentURL  string                 `json:"attachment_url,omitempty" validate:"omitempty,url"`     // Deprecated: use AttachmentURLs
	AttachmentURLs []string               `json:"attachment_urls,omitempty"`                             // New: Multiple attachments
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	ProductID      string                 `json:"product_id,omitempty"` // ProductID for the message
}

// REMOVED: Request for creating a middleman chat
// type createMiddlemanChatRequest struct {
// 	BuyerID     string `json:"buyer_id" validate:"required"`
// 	SellerID    string `json:"seller_id" validate:"required"`
// 	MiddlemanID string `json:"middleman_id" validate:"required"`
// 	ProductID   string `json:"product_id" validate:"required"`
// 	TransactionID string `json:"transaction_id" validate:"required"`
// 	InitialMessage string `json:"initial_message"`
// }

// Request for accepting/rejecting an offer
type offerActionRequest struct {
	MessageID string `json:"message_id" validate:"required"`
}

// CreateChat creates a new chat between users
func (h *ChatHandler) CreateChat(c echo.Context) error {
	var req createChatRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	userID := c.Get("uid").(string)

	chat, err := h.chatUseCase.CreateChat(c.Request().Context(), userID, usecase.CreateChatInput{
		RecipientID:    req.RecipientID,
		ProductID:      req.ProductID,
		InitialMessage: req.InitialMessage,
	})

	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, chat)
}

// REMOVED: CreateMiddlemanChat handler (now triggered by TransactionUseCase)
// func (h *ChatHandler) CreateMiddlemanChat(c echo.Context) error {
// 	var req createMiddlemanChatRequest
// 	if err := c.Bind(&req); err != nil {
// 		return response.Error(c, err)
// 	}

// 	if err := c.Validate(&req); err != nil {
// 		return response.Error(c, err)
// 	}

// 	adminID := c.Get("uid").(string)
// 	if adminID != req.MiddlemanID {
// 		return response.Error(c, errors.Forbidden("Admin ID must match Middleman ID in request", nil))
// 	}

// 	chat, err := h.chatUseCase.CreateMiddlemanChat(c.Request().Context(), usecase.CreateMiddlemanChatInput{
// 		BuyerID:     req.BuyerID,
// 		SellerID:    req.SellerID,
// 		MiddlemanID: req.MiddlemanID,
// 		ProductID:   req.ProductID,
// 		TransactionID: req.TransactionID,
// 		InitialMessage: req.InitialMessage,
// 	})

// 	if err != nil {
// 		return response.Error(c, err)
// 	}

// 	return response.Created(c, chat)
// }

// GetUserChats gets all chats for the authenticated user
func (h *ChatHandler) GetUserChats(c echo.Context) error {
	userID := c.Get("uid").(string)

	limit := 20
	offset := 0

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	chats, total, err := h.chatUseCase.GetUserChats(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessPaginated(c, chats, total, limit, offset)
}

// GetChatByID gets a specific chat by ID
func (h *ChatHandler) GetChatByID(c echo.Context) error {
	chatID := c.Param("id")
	userID := c.Get("uid").(string)

	chat, err := h.chatUseCase.GetChatByID(c.Request().Context(), userID, chatID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, chat)
}

// SendMessage sends a message to a chat
func (h *ChatHandler) SendMessage(c echo.Context) error {
	chatID := c.Param("id")
	userID := c.Get("uid").(string)

	var req sendMessageRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	if req.Type == "" {
		req.Type = "text"
	}

	message, err := h.chatUseCase.SendMessage(c.Request().Context(), userID, usecase.SendMessageInput{
		ChatID:         chatID,
		Content:        req.Content,
		Type:          req.Type,
		AttachmentURL:  req.AttachmentURL,  // Backward compatibility
		AttachmentURLs: req.AttachmentURLs, // New: Multiple attachments
		Metadata:       req.Metadata,       // Pass metadata
		ProductID:      req.ProductID,      // Pass ProductID for the message
	})

	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, message)
}

// GetChatMessages gets messages for a specific chat
func (h *ChatHandler) GetChatMessages(c echo.Context) error {
	chatID := c.Param("id")
	userID := c.Get("uid").(string)

	limit := 50
	offset := 0

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	messages, total, err := h.chatUseCase.GetChatMessages(c.Request().Context(), userID, chatID, limit, offset)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessPaginated(c, messages, total, limit, offset)
}

// MarkChatAsRead marks a chat as read for the authenticated user
func (h *ChatHandler) MarkChatAsRead(c echo.Context) error {
	chatID := c.Param("id")
	userID := c.Get("uid").(string)

	err := h.chatUseCase.MarkChatAsRead(c.Request().Context(), userID, chatID)
	if err != nil {
		return response.Error(c, err)
	}

	return c.NoContent(http.StatusOK)
}

// New: AcceptOffer handler
func (h *ChatHandler) AcceptOffer(c echo.Context) error {
	chatID := c.Param("id")
	var req offerActionRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}
	userID := c.Get("uid").(string)

	err := h.chatUseCase.AcceptOffer(c.Request().Context(), chatID, req.MessageID, userID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, map[string]string{"message": "Offer accepted"})
}

// New: RejectOffer handler
func (h *ChatHandler) RejectOffer(c echo.Context) error {
	chatID := c.Param("id")
	var req offerActionRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}
	userID := c.Get("uid").(string)

	err := h.chatUseCase.RejectOffer(c.Request().Context(), chatID, req.MessageID, userID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, map[string]string{"message": "Offer rejected"})
}

// New: CreateGroupChat - Buyer initiates group chat with auto-selected seller and chosen middleman
func (h *ChatHandler) CreateGroupChat(c echo.Context) error {
	userID := c.Get("uid").(string)

	type createGroupChatRequest struct {
		ProductID      string `json:"product_id" validate:"required"`
		MiddlemanID    string `json:"middleman_id" validate:"required"`
		InitialMessage string `json:"initial_message"`
	}

	var req createGroupChatRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	input := usecase.CreateGroupChatInput{
		ProductID:      req.ProductID,
		MiddlemanID:    req.MiddlemanID,
		InitialMessage: req.InitialMessage,
	}

	chat, err := h.chatUseCase.CreateBuyerGroupChat(c.Request().Context(), userID, input)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, chat)
}

// New: CreateSellerGroupChat - Seller initiates group chat with chosen buyer and middleman
func (h *ChatHandler) CreateSellerGroupChat(c echo.Context) error {
	userID := c.Get("uid").(string)

	type createSellerGroupChatRequest struct {
		ProductID      string `json:"product_id" validate:"required"`
		BuyerID        string `json:"buyer_id" validate:"required"`
		MiddlemanID    string `json:"middleman_id" validate:"required"`
		InitialMessage string `json:"initial_message"`
	}

	var req createSellerGroupChatRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	input := usecase.CreateSellerGroupChatInput{
		ProductID:      req.ProductID,
		BuyerID:        req.BuyerID,
		MiddlemanID:    req.MiddlemanID,
		InitialMessage: req.InitialMessage,
	}

	chat, err := h.chatUseCase.CreateSellerGroupChat(c.Request().Context(), userID, input)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, chat)
}

// New: ListAvailableMiddlemen - Get list of available admin users for middleman selection
func (h *ChatHandler) ListAvailableMiddlemen(c echo.Context) error {
	currentUserID := c.Get("uid").(string)
	
	admins, err := h.chatUseCase.ListAvailableMiddlemen(c.Request().Context())
	if err != nil {
		return response.Error(c, err)
	}

	// Return simplified admin info for middleman selection, excluding current user
	type middlemanOption struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	}

	var options []middlemanOption
	for _, admin := range admins {
		// Skip current user from middleman options
		if admin.ID != currentUserID {
			options = append(options, middlemanOption{
				ID:       admin.ID,
				Username: admin.Username,
				Email:    admin.Email,
			})
		}
	}

	return response.Success(c, map[string]interface{}{
		"middlemen": options,
	})
}

// New: GetUserRoleInChat - Get current user's role in specific chat
func (h *ChatHandler) GetUserRoleInChat(c echo.Context) error {
	userID := c.Get("uid").(string)
	chatID := c.Param("id")

	role, err := h.chatUseCase.GetUserRoleInChat(c.Request().Context(), chatID, userID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, map[string]interface{}{
		"chat_id": chatID,
		"user_id": userID,
		"role":    role,
	})
}

// New: GetChatParticipantsWithRoles - Get all participants with their roles
func (h *ChatHandler) GetChatParticipantsWithRoles(c echo.Context) error {
	chatID := c.Param("id")

	participants, err := h.chatUseCase.GetChatParticipantsWithRoles(c.Request().Context(), chatID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, participants)
}

// New: ListUsers - Get list of users for buyer/seller selection
func (h *ChatHandler) ListUsers(c echo.Context) error {
	users, err := h.chatUseCase.ListUsers(c.Request().Context())
	if err != nil {
		return response.Error(c, err)
	}

	// Return simplified user info for selection
	type userOption struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}

	var options []userOption
	for _, user := range users {
		options = append(options, userOption{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
		})
	}

	return response.Success(c, map[string]interface{}{
		"users": options,
	})
}

// New: GetUserProducts - Get products owned by specific user
func (h *ChatHandler) GetUserProducts(c echo.Context) error {
	userID := c.Param("userId")
	if userID == "" {
		return response.Error(c, errors.BadRequest("User ID is required", nil))
	}

	products, err := h.chatUseCase.GetUserProducts(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, map[string]interface{}{
		"products": products,
	})
}

// New: CreateTransactionChat - Universal transaction chat creator
func (h *ChatHandler) CreateTransactionChat(c echo.Context) error {
	type createTransactionChatRequest struct {
		BuyerID        string `json:"buyer_id" validate:"required"`
		SellerID       string `json:"seller_id" validate:"required"`
		ProductID      string `json:"product_id" validate:"required"`
		MiddlemanID    string `json:"middleman_id" validate:"required"`
		InitialMessage string `json:"initial_message"`
	}

	var req createTransactionChatRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	creatorID := c.Get("uid").(string)

	input := usecase.CreateTransactionChatInput{
		BuyerID:        req.BuyerID,
		SellerID:       req.SellerID,
		ProductID:      req.ProductID,
		MiddlemanID:    req.MiddlemanID,
		InitialMessage: req.InitialMessage,
	}

	chat, err := h.chatUseCase.CreateTransactionChat(c.Request().Context(), creatorID, input)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, chat)
}

// New: CreateChatByProduct - Create direct chat with seller via product ID
func (h *ChatHandler) CreateChatByProduct(c echo.Context) error {
	type createChatByProductRequest struct {
		ProductID      string `json:"product_id" validate:"required"`
		InitialMessage string `json:"initial_message"`
	}

	var req createChatByProductRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	buyerID := c.Get("uid").(string)

	chat, err := h.chatUseCase.CreateChatByProduct(c.Request().Context(), buyerID, usecase.CreateChatByProductInput{
		ProductID:      req.ProductID,
		InitialMessage: req.InitialMessage,
	})

	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, chat)
}

// Helper function to parse pagination parameters
func parsePagination(c echo.Context) (int, int) {
	limit := 20
	offset := 0

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	return limit, offset
}

// New: GetIndividualChats - Get only direct/individual chats
func (h *ChatHandler) GetIndividualChats(c echo.Context) error {
	userID := c.Get("uid").(string)
	limit, offset := parsePagination(c)

	chats, total, err := h.chatUseCase.GetChatsByType(c.Request().Context(), userID, "direct", limit, offset)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, map[string]interface{}{
		"items": chats,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

// New: GetGroupChats - Get only group/transaction chats
func (h *ChatHandler) GetGroupChats(c echo.Context) error {
	userID := c.Get("uid").(string)
	limit, offset := parsePagination(c)

	chats, total, err := h.chatUseCase.GetChatsByType(c.Request().Context(), userID, "group", limit, offset)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, map[string]interface{}{
		"items": chats,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}
