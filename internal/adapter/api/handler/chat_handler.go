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
	Type           string `json:"type"`
}

type sendMessageRequest struct {
	Content string `json:"content" validate:"required"`
	Type    string `json:"type"`
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

	// Set default message type if not provided
	if req.Type == "" {
		req.Type = "text"
	}

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

// GetUserChats gets all chats for the authenticated user
func (h *ChatHandler) GetUserChats(c echo.Context) error {
	userID := c.Get("uid").(string)

	// Parse pagination parameters
	limit := 20 // Default limit
	offset := 0 // Default offset

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

	// Set default message type if not provided
	if req.Type == "" {
		req.Type = "text"
	}

	message, err := h.chatUseCase.SendMessage(c.Request().Context(), userID, usecase.SendMessageInput{
		ChatID:  chatID,
		Content: req.Content,
		Type:    req.Type,
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

	// Parse pagination parameters
	limit := 50 // Default limit for messages
	offset := 0 // Default offset

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
