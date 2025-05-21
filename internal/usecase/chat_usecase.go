package usecase

import (
	"context"
	"encoding/json"
	"time"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/internal/infrastructure/websocket"
	"pasargamex/pkg/errors"
)

type ChatUseCase struct {
	chatRepo    repository.ChatRepository
	userRepo    repository.UserRepository
	productRepo repository.ProductRepository
	wsManager   *websocket.Manager
}

func NewChatUseCase(
	chatRepo repository.ChatRepository,
	userRepo repository.UserRepository,
	productRepo repository.ProductRepository,
	wsManager *websocket.Manager,
) *ChatUseCase {
	return &ChatUseCase{
		chatRepo:    chatRepo,
		userRepo:    userRepo,
		productRepo: productRepo,
		wsManager:   wsManager,
	}
}

type CreateChatInput struct {
	RecipientID    string
	ProductID      string
	InitialMessage string
}

func (uc *ChatUseCase) CreateChat(ctx context.Context, userID string, input CreateChatInput) (*entity.Chat, error) {
	_, err := uc.userRepo.GetByID(ctx, input.RecipientID)
	if err != nil {
		return nil, errors.NotFound("Recipient not found", err)
	}

	// Check if chat already exists between these users for this product
	// This would require an additional method in the repository

	// Create chat
	chat := &entity.Chat{
		Participants: []string{userID, input.RecipientID},
		ProductID:    input.ProductID,
		Type:         "direct",
		UnreadCount:  map[string]int{input.RecipientID: 1},
	}

	if err := uc.chatRepo.Create(ctx, chat); err != nil {
		return nil, err
	}

	// If there's an initial message, create it
	if input.InitialMessage != "" {
		message := &entity.Message{
			ChatID:    chat.ID,
			SenderID:  userID,
			Content:   input.InitialMessage,
			Type:      "text",
			ReadBy:    []string{userID},
			CreatedAt: time.Now(),
		}

		if err := uc.chatRepo.CreateMessage(ctx, message); err != nil {
			return nil, err
		}

		// Update chat with last message
		chat.LastMessage = input.InitialMessage
		chat.LastMessageAt = message.CreatedAt

		if err := uc.chatRepo.Update(ctx, chat); err != nil {
			return nil, err
		}

		// Notify recipient via WebSocket
		notification := map[string]interface{}{
			"type":    "new_message",
			"chat_id": chat.ID,
			"message": message,
		}

		notificationJSON, _ := json.Marshal(notification)
		uc.wsManager.SendToUser(input.RecipientID, notificationJSON)
	}

	return chat, nil
}
