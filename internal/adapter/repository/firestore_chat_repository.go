package repository

import (
	"context"
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

	// Use subcollection for messages
	_, err := r.client.Collection("chats").Doc(message.ChatID).Collection("messages").Doc(message.ID).Set(ctx, message)
	if err != nil {
		return errors.Internal("Failed to create message", err)
	}

	return nil
}

func (r *firestoreChatRepository) GetMessagesByChat(ctx context.Context, chatID string, limit, offset int) ([]*entity.Message, int64, error) {
	query := r.client.Collection("chats").Doc(chatID).Collection("messages").OrderBy("createdAt", firestore.Desc)

	// Get total count
	countDocs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, 0, errors.Internal("Failed to count messages", err)
	}
	total := int64(len(countDocs))

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// Execute query
	iter := query.Documents(ctx)
	var messages []*entity.Message

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, errors.Internal("Failed to iterate messages", err)
		}

		var message entity.Message
		if err := doc.DataTo(&message); err != nil {
			return nil, 0, errors.Internal("Failed to parse message data", err)
		}

		messages = append(messages, &message)
	}

	return messages, total, nil
}

func (r *firestoreChatRepository) UpdateMessageReadStatus(ctx context.Context, messageID string, userID string) error {
	// This assumes you know which chat the message belongs to
	// You might need to store the chatID with the messageID to make this more efficient

	// First get all chats to find the message
	iter := r.client.CollectionGroup("messages").Where("id", "==", messageID).Limit(1).Documents(ctx)
	doc, err := iter.Next()

	if err != nil {
		if err == iterator.Done {
			return errors.NotFound("Message not found", nil)
		}
		return errors.Internal("Failed to get message", err)
	}

	// Get the message and update read status
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
	_, err = doc.Ref.Set(ctx, message)
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

func (r *firestoreChatRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*entity.Chat, int64, error) {
	query := r.client.Collection("chats").Where("participants", "array-contains", userID).OrderBy("updatedAt", firestore.Desc)

	// Get total count
	countDocs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, 0, errors.Internal("Failed to count chats", err)
	}
	total := int64(len(countDocs))

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// Execute query
	iter := query.Documents(ctx)
	var chats []*entity.Chat

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, errors.Internal("Failed to iterate chats", err)
		}

		var chat entity.Chat
		if err := doc.DataTo(&chat); err != nil {
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
