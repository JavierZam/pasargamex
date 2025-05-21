package handler

import (
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
