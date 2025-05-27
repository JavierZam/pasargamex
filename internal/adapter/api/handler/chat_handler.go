package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"pasargamex/internal/usecase"
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
	Content       string                 `json:"content" validate:"required"`
	Type          string                 `json:"type" validate:"required,oneof=text image system offer"`
	AttachmentURL string                 `json:"attachment_url,omitempty" validate:"omitempty,url"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	ProductID     string                 `json:"product_id,omitempty"` // ProductID for the message
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
		ChatID:        chatID,
		Content:       req.Content,
		Type:          req.Type,
		AttachmentURL: req.AttachmentURL, // Pass attachment URL
		Metadata:      req.Metadata,      // Pass metadata
		ProductID:     req.ProductID,     // Pass ProductID for the message
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
