package repository

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
)

type firestoreChatRepository struct {
	client *firestore.Client
}

func NewFirestoreChatRepository(client *firestore.Client) repository.ChatRepository {
	return &firestoreChatRepository{
		client: client,
	}
}

func (r *firestoreChatRepository) Create(ctx context.Context, chat *entity.Chat) error {
	if chat.ID == "" {
		chat.ID = uuid.New().String()
	}

	now := time.Now()
	chat.CreatedAt = now
	chat.UpdatedAt = now

	_, err := r.client.Collection("chats").Doc(chat.ID).Set(ctx, chat)
	if err != nil {
		return errors.Internal("Failed to create chat", err)
	}

	return nil
}

func (r *firestoreChatRepository) CreateMessage(ctx context.Context, message *entity.Message) error {
	if message.ID == "" {
		message.ID = uuid.New().String()
	}

	message.CreatedAt = time.Now()

	_, err := r.client.Collection("chats").Doc(message.ChatID).Collection("messages").Doc(message.ID).Set(ctx, message)
	if err != nil {
		return errors.Internal("Failed to create message", err)
	}

	return nil
}

// New: GetMessageByID retrieves a specific message by its ID within a chat
func (r *firestoreChatRepository) GetMessageByID(ctx context.Context, chatID, messageID string) (*entity.Message, error) {
	doc, err := r.client.Collection("chats").Doc(chatID).Collection("messages").Doc(messageID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, errors.NotFound("Message", err)
		}
		return nil, errors.Internal("Failed to get message", err)
	}

	var message entity.Message
	if err := doc.DataTo(&message); err != nil {
		return nil, errors.Internal("Failed to parse message data", err)
	}
	return &message, nil
}

// New: UpdateMessage updates an existing message (e.g., for offer status)
func (r *firestoreChatRepository) UpdateMessage(ctx context.Context, chatID string, message *entity.Message) error {
	_, err := r.client.Collection("chats").Doc(chatID).Collection("messages").Doc(message.ID).Set(ctx, message)
	if err != nil {
		return errors.Internal("Failed to update message", err)
	}
	return nil
}

func (r *firestoreChatRepository) GetMessagesByChat(ctx context.Context, chatID string, limit, offset int) ([]*entity.Message, int64, error) {
	query := r.client.Collection("chats").Doc(chatID).Collection("messages").OrderBy("createdAt", firestore.Desc)

	countDocs, err := query.Documents(ctx).GetAll()
	if err != nil {
		log.Printf("Firestore error while counting messages for chat %s: %v", chatID, err)
		return nil, 0, errors.Internal("Failed to count messages for chat", err)
	}
	total := int64(len(countDocs))

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	iter := query.Documents(ctx)
	var messages []*entity.Message

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Firestore error while iterating messages for chat %s: %v", chatID, err)
			return nil, 0, errors.Internal("Failed to iterate messages", err)
		}

		var message entity.Message
		if err := doc.DataTo(&message); err != nil {
			log.Printf("Error parsing message data for chat %s: %v", chatID, err)
			return nil, 0, errors.Internal("Failed to parse message data", err)
		}

		messages = append(messages, &message)
	}

	return messages, total, nil
}

func (r *firestoreChatRepository) UpdateMessageReadStatus(ctx context.Context, messageID string, userID string) error {
	// To update read status, we need to find the message first.
	// This approach uses CollectionGroup which requires an index for 'id' field.
	// Alternatively, if chatID is passed along with messageID, we can directly access the subcollection.
	// For now, let's assume messageID is unique enough to find the message across chats.
	// If performance is an issue, consider passing chatID to this method.
	iter := r.client.CollectionGroup("messages").Where("id", "==", messageID).Limit(1).Documents(ctx)
	doc, err := iter.Next()

	if err != nil {
		if err == iterator.Done {
			return errors.NotFound("Message not found", nil)
		}
		return errors.Internal("Failed to get message", err)
	}

	var message entity.Message
	if err := doc.DataTo(&message); err != nil {
		return errors.Internal("Failed to parse message data", err)
	}

	// Check if user already marked as read
	for _, reader := range message.ReadBy {
		if reader == userID {
			return nil // Already marked as read
		}
	}

	// Add user to read list
	message.ReadBy = append(message.ReadBy, userID)

	// Update the message
	_, err = doc.Ref.Set(ctx, message) // Use doc.Ref.Set to update the specific message
	if err != nil {
		return errors.Internal("Failed to update message read status", err)
	}

	return nil
}

func (r *firestoreChatRepository) GetByID(ctx context.Context, id string) (*entity.Chat, error) {
	doc, err := r.client.Collection("chats").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, errors.NotFound("Chat not found", nil)
		}
		return nil, errors.Internal("Failed to get chat", err)
	}

	var chat entity.Chat
	if err := doc.DataTo(&chat); err != nil {
		return nil, errors.Internal("Failed to parse chat data", err)
	}

	return &chat, nil
}

// New: GetChatByTransactionID retrieves a chat by its associated transaction ID
func (r *firestoreChatRepository) GetChatByTransactionID(ctx context.Context, transactionID string) (*entity.Chat, error) {
	query := r.client.Collection("chats").Where("transactionId", "==", transactionID).Limit(1)
	iter := query.Documents(ctx)
	doc, err := iter.Next()

	if err != nil {
		if err == iterator.Done {
			return nil, errors.NotFound("Chat for transaction not found", nil)
		}
		return nil, errors.Internal("Failed to query chat by transaction ID", err)
	}

	var chat entity.Chat
	if err := doc.DataTo(&chat); err != nil {
		return nil, errors.Internal("Failed to parse chat data", err)
	}

	return &chat, nil
}

func (r *firestoreChatRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*entity.Chat, int64, error) {
	query := r.client.Collection("chats").Where("participants", "array-contains", userID).OrderBy("updatedAt", firestore.Desc)

	countDocs, err := query.Documents(ctx).GetAll()
	if err != nil {
		log.Printf("Firestore error while counting chats for user %s: %v", userID, err)
		return nil, 0, errors.Internal("Failed to count chats", err)
	}
	total := int64(len(countDocs))

	if limit != -1 && limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	iter := query.Documents(ctx)
	var chats []*entity.Chat

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Firestore error while iterating chats for user %s: %v", userID, err)
			return nil, 0, errors.Internal("Failed to iterate chats", err)
		}

		var chat entity.Chat
		if err := doc.DataTo(&chat); err != nil {
			log.Printf("Error parsing chat data for user %s: %v", userID, err)
			return nil, 0, errors.Internal("Failed to parse chat data", err)
		}

		chats = append(chats, &chat)
	}

	return chats, total, nil
}
func (r *firestoreChatRepository) Update(ctx context.Context, chat *entity.Chat) error {
	chat.UpdatedAt = time.Now()

	_, err := r.client.Collection("chats").Doc(chat.ID).Set(ctx, chat)
	if err != nil {
		return errors.Internal("Failed to update chat", err)
	}

	return nil
}
func (r *firestoreChatRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.Collection("chats").Doc(id).Delete(ctx)
	if err != nil {
		return errors.Internal("Failed to delete chat", err)
	}

	return nil
}
