package usecase

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	ws "pasargamex/internal/infrastructure/websocket"
	"pasargamex/pkg/errors"
)

type ChatUseCase struct {
	chatRepo    repository.ChatRepository
	userRepo    repository.UserRepository
	productRepo repository.ProductRepository
	wsManager   *ws.Manager
}

func NewChatUseCase(
	chatRepo repository.ChatRepository,
	userRepo repository.UserRepository,
	productRepo repository.ProductRepository,
	wsManager *ws.Manager,
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

type SendMessageInput struct {
	ChatID  string
	Content string
	Type    string // "text", "image", "system", "offer"
}

type ChatResponse struct {
	*entity.Chat
	Product   *entity.Product `json:"product,omitempty"`
	OtherUser *entity.User    `json:"other_user,omitempty"`
}

type MessageResponse struct {
	*entity.Message
	Sender *entity.User `json:"sender,omitempty"`
}

func (uc *ChatUseCase) CreateChat(ctx context.Context, userID string, input CreateChatInput) (*ChatResponse, error) {
	// Validate recipient exists
	recipient, err := uc.userRepo.GetByID(ctx, input.RecipientID)
	if err != nil {
		return nil, errors.NotFound("Recipient not found", err)
	}

	// Validate product if provided
	var product *entity.Product
	if input.ProductID != "" {
		product, err = uc.productRepo.GetByID(ctx, input.ProductID)
		if err != nil {
			return nil, errors.NotFound("Product not found", err)
		}
	}

	// Check if chat already exists between these users for this product
	existingChat, err := uc.findExistingChat(ctx, userID, input.RecipientID, input.ProductID)
	if err == nil && existingChat != nil {
		// Chat already exists, return it
		log.Printf("Found existing chat: %s between users %s and %s for product %s",
			existingChat.ID, userID, input.RecipientID, input.ProductID)

		return &ChatResponse{
			Chat:      existingChat,
			Product:   product,
			OtherUser: recipient,
		}, nil
	}

	log.Printf("Creating new chat between users %s and %s for product %s",
		userID, input.RecipientID, input.ProductID)

	// Create new chat
	chat := &entity.Chat{
		Participants:  []string{userID, input.RecipientID},
		ProductID:     input.ProductID,
		Type:          "direct",
		UnreadCount:   make(map[string]int),
		LastMessageAt: time.Now(),
	}

	if err := uc.chatRepo.Create(ctx, chat); err != nil {
		return nil, err
	}

	// If there's an initial message, create it
	if input.InitialMessage != "" {
		messageResp, err := uc.SendMessage(ctx, userID, SendMessageInput{
			ChatID:  chat.ID,
			Content: input.InitialMessage,
			Type:    "text",
		})
		if err != nil {
			return nil, err
		}

		// Update chat with last message
		chat.LastMessage = input.InitialMessage
		chat.LastMessageAt = messageResp.CreatedAt
		chat.UnreadCount[input.RecipientID] = 1

		if err := uc.chatRepo.Update(ctx, chat); err != nil {
			return nil, err
		}
	}

	return &ChatResponse{
		Chat:      chat,
		Product:   product,
		OtherUser: recipient,
	}, nil
}

// Add new helper function to find existing chat
func (uc *ChatUseCase) findExistingChat(ctx context.Context, userID1, userID2, productID string) (*entity.Chat, error) {
	log.Printf("Looking for existing chat between %s and %s for product %s", userID1, userID2, productID)

	// Get all chats for user1
	chats1, _, err := uc.chatRepo.ListByUserID(ctx, userID1, 100, 0)
	if err != nil {
		log.Printf("Error getting chats for user %s: %v", userID1, err)
		return nil, err
	}

	log.Printf("Found %d chats for user %s", len(chats1), userID1)

	// Check each chat
	for _, chat := range chats1 {
		log.Printf("Checking chat %s: participants=%v, productID=%s",
			chat.ID, chat.Participants, chat.ProductID)

		// Check if it's a direct chat between the two users
		if len(chat.Participants) == 2 &&
			containsString(chat.Participants, userID1) &&
			containsString(chat.Participants, userID2) {

			// Check product ID match
			// Both should be empty OR both should be the same
			if (productID == "" && chat.ProductID == "") ||
				(productID != "" && chat.ProductID == productID) {
				log.Printf("Found existing chat: %s", chat.ID)
				return chat, nil
			} else {
				log.Printf("Product ID mismatch: wanted=%s, found=%s", productID, chat.ProductID)
			}
		} else {
			log.Printf("Participants mismatch: wanted=[%s,%s], found=%v",
				userID1, userID2, chat.Participants)
		}
	}

	log.Printf("No existing chat found")
	return nil, errors.NotFound("No existing chat found", nil)
}

// Update helper function name to be more specific
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (uc *ChatUseCase) SendMessage(ctx context.Context, userID string, input SendMessageInput) (*MessageResponse, error) {
	// Validate chat exists and user is participant
	chat, err := uc.chatRepo.GetByID(ctx, input.ChatID)
	if err != nil {
		return nil, err
	}

	if !contains(chat.Participants, userID) {
		return nil, errors.Forbidden("User is not a participant in this chat", nil)
	}

	// Get sender info
	sender, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.NotFound("Sender not found", err)
	}

	// Create message
	message := &entity.Message{
		ChatID:    input.ChatID,
		SenderID:  userID,
		Content:   input.Content,
		Type:      input.Type,
		ReadBy:    []string{userID}, // Sender automatically reads their own message
		CreatedAt: time.Now(),
	}

	if err := uc.chatRepo.CreateMessage(ctx, message); err != nil {
		return nil, err
	}

	// Update chat last message info
	chat.LastMessage = input.Content
	chat.LastMessageAt = message.CreatedAt

	// Update unread count for other participants
	if chat.UnreadCount == nil {
		chat.UnreadCount = make(map[string]int)
	}

	for _, participantID := range chat.Participants {
		if participantID != userID {
			chat.UnreadCount[participantID]++
		}
	}

	if err := uc.chatRepo.Update(ctx, chat); err != nil {
		return nil, err
	}

	// Send real-time notification to other participants
	notification := map[string]interface{}{
		"type":    "new_message",
		"chat_id": input.ChatID,
		"message": message,
		"sender":  sender,
	}

	notificationJSON, _ := json.Marshal(notification)
	for _, participantID := range chat.Participants {
		if participantID != userID {
			uc.wsManager.SendToUser(participantID, notificationJSON)
		}
	}

	return &MessageResponse{
		Message: message,
		Sender:  sender,
	}, nil
}

func (uc *ChatUseCase) GetUserChats(ctx context.Context, userID string, limit, offset int) ([]*ChatResponse, int64, error) {
	chats, total, err := uc.chatRepo.ListByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	var chatResponses []*ChatResponse

	for _, chat := range chats {
		chatResp := &ChatResponse{Chat: chat}

		// Get product info if exists
		if chat.ProductID != "" {
			product, err := uc.productRepo.GetByID(ctx, chat.ProductID)
			if err == nil { // Don't fail if product not found
				chatResp.Product = product
			}
		}

		// Get other user info
		for _, participantID := range chat.Participants {
			if participantID != userID {
				otherUser, err := uc.userRepo.GetByID(ctx, participantID)
				if err == nil { // Don't fail if user not found
					chatResp.OtherUser = otherUser
				}
				break
			}
		}

		chatResponses = append(chatResponses, chatResp)
	}

	return chatResponses, total, nil
}

func (uc *ChatUseCase) GetChatMessages(ctx context.Context, userID, chatID string, limit, offset int) ([]*MessageResponse, int64, error) {
	// Validate user can access this chat
	chat, err := uc.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return nil, 0, err
	}

	if !contains(chat.Participants, userID) {
		return nil, 0, errors.Forbidden("User is not a participant in this chat", nil)
	}

	messages, total, err := uc.chatRepo.GetMessagesByChat(ctx, chatID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	var messageResponses []*MessageResponse

	for _, message := range messages {
		messageResp := &MessageResponse{Message: message}

		// Get sender info
		sender, err := uc.userRepo.GetByID(ctx, message.SenderID)
		if err == nil { // Don't fail if sender not found
			messageResp.Sender = sender
		}

		messageResponses = append(messageResponses, messageResp)
	}

	return messageResponses, total, nil
}

func (uc *ChatUseCase) MarkChatAsRead(ctx context.Context, userID, chatID string) error {
	// Validate user can access this chat
	chat, err := uc.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return err
	}

	if !contains(chat.Participants, userID) {
		return errors.Forbidden("User is not a participant in this chat", nil)
	}

	// Reset unread count for this user
	if chat.UnreadCount == nil {
		chat.UnreadCount = make(map[string]int)
	}
	chat.UnreadCount[userID] = 0

	return uc.chatRepo.Update(ctx, chat)
}

func (uc *ChatUseCase) GetChatByID(ctx context.Context, userID, chatID string) (*ChatResponse, error) {
	chat, err := uc.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	if !contains(chat.Participants, userID) {
		return nil, errors.Forbidden("User is not a participant in this chat", nil)
	}

	chatResp := &ChatResponse{Chat: chat}

	// Get product info if exists
	if chat.ProductID != "" {
		product, err := uc.productRepo.GetByID(ctx, chat.ProductID)
		if err == nil {
			chatResp.Product = product
		}
	}

	// Get other user info
	for _, participantID := range chat.Participants {
		if participantID != userID {
			otherUser, err := uc.userRepo.GetByID(ctx, participantID)
			if err == nil {
				chatResp.OtherUser = otherUser
			}
			break
		}
	}

	return chatResp, nil
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
