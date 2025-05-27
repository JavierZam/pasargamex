package repository

import (
	"context"
	"pasargamex/internal/domain/entity"
)

type ChatRepository interface {
	Create(ctx context.Context, chat *entity.Chat) error
	GetByID(ctx context.Context, id string) (*entity.Chat, error)
	ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*entity.Chat, int64, error)
	Update(ctx context.Context, chat *entity.Chat) error
	Delete(ctx context.Context, id string) error

	// Message methods
	CreateMessage(ctx context.Context, message *entity.Message) error
	GetMessagesByChat(ctx context.Context, chatID string, limit, offset int) ([]*entity.Message, int64, error)
	UpdateMessageReadStatus(ctx context.Context, messageID string, userID string) error

	// New methods for advanced chat features
	GetChatByTransactionID(ctx context.Context, transactionID string) (*entity.Chat, error) // New
	GetMessageByID(ctx context.Context, chatID, messageID string) (*entity.Message, error)  // New
	UpdateMessage(ctx context.Context, chatID string, message *entity.Message) error        // New
}
