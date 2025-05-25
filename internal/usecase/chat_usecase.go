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
	recipient, err := uc.userRepo.GetByID(ctx, input.RecipientID)
	if err != nil {
		log.Printf("CreateChat Error: Recipient %s not found: %v", input.RecipientID, err)
		return nil, errors.NotFound("Recipient not found", err)
	}

	var product *entity.Product
	if input.ProductID != "" {
		product, err = uc.productRepo.GetByID(ctx, input.ProductID)
		if err != nil {
			log.Printf("CreateChat Error: Product %s not found: %v", input.ProductID, err)
			return nil, errors.NotFound("Product not found", err)
		}
	}

	var chatToReturn *entity.Chat

	existingChat, err := uc.findExistingChat(ctx, userID, input.RecipientID)
	if err == nil && existingChat != nil {
		log.Printf("CreateChat: Found existing chat %s between users %s and %s. Reusing this chat.",
			existingChat.ID, userID, input.RecipientID)
		chatToReturn = existingChat
	} else {
		if err != nil && !errors.Is(err, "NOT_FOUND") {
			log.Printf("CreateChat Error: Failed to search for existing chat: %v", err)
			return nil, err
		}

		newChat := &entity.Chat{
			Participants:  []string{userID, input.RecipientID},
			ProductID:     input.ProductID,
			Type:          "direct",
			UnreadCount:   make(map[string]int),
			LastMessageAt: time.Now(),
		}

		if err := uc.chatRepo.Create(ctx, newChat); err != nil {
			log.Printf("CreateChat Error: Failed to create new chat in repository: %v", err)
			return nil, err
		}
		chatToReturn = newChat
		log.Printf("CreateChat: Successfully created new chat %s", chatToReturn.ID)
	}

	if input.InitialMessage != "" {
		messageResp, err := uc.SendMessage(ctx, userID, SendMessageInput{
			ChatID:  chatToReturn.ID,
			Content: input.InitialMessage,
			Type:    "text",
		})
		if err != nil {
			log.Printf("CreateChat Error: Failed to send initial message for chat %s: %v", chatToReturn.ID, err)
			return nil, err
		}

		chatToReturn.LastMessage = input.InitialMessage
		chatToReturn.LastMessageAt = messageResp.CreatedAt
		if chatToReturn.UnreadCount == nil {
			chatToReturn.UnreadCount = make(map[string]int)
		}
		for _, participantID := range chatToReturn.Participants {
			if participantID != userID {
				chatToReturn.UnreadCount[participantID]++
			}
		}

		if err := uc.chatRepo.Update(ctx, chatToReturn); err != nil {
			log.Printf("CreateChat Error: Failed to update chat %s with last message after initial message: %v", chatToReturn.ID, err)
		}
		log.Printf("CreateChat: Successfully sent initial message to chat %s", chatToReturn.ID)
	}

	return &ChatResponse{
		Chat:      chatToReturn,
		Product:   product,
		OtherUser: recipient,
	}, nil
}

func (uc *ChatUseCase) findExistingChat(ctx context.Context, userID1, userID2 string) (*entity.Chat, error) {
	chats1, _, err := uc.chatRepo.ListByUserID(ctx, userID1, -1, 0) // Use -1 limit to fetch all
	if err != nil {
		log.Printf("findExistingChat Error: Failed to list chats for user %s: %v", userID1, err)
		return nil, errors.Internal("Failed to list chats for user", err)
	}

	for _, chat := range chats1 {
		if chat.Type == "direct" && len(chat.Participants) == 2 {
			if containsString(chat.Participants, userID1) && containsString(chat.Participants, userID2) {
				return chat, nil
			}
		}
	}

	return nil, errors.NotFound("No existing chat found", nil)
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (uc *ChatUseCase) SendMessage(ctx context.Context, userID string, input SendMessageInput) (*MessageResponse, error) {
	chat, err := uc.chatRepo.GetByID(ctx, input.ChatID)
	if err != nil {
		log.Printf("SendMessage Error: Chat %s not found: %v", input.ChatID, err)
		return nil, err
	}

	if !containsString(chat.Participants, userID) {
		log.Printf("SendMessage Error: User %s is not a participant in chat %s", userID, input.ChatID)
		return nil, errors.Forbidden("User is not a participant in this chat", nil)
	}

	sender, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		log.Printf("SendMessage Error: Sender %s not found: %v", userID, err)
		return nil, errors.NotFound("Sender not found", err)
	}

	message := &entity.Message{
		ChatID:    input.ChatID,
		SenderID:  userID,
		Content:   input.Content,
		Type:      input.Type,
		ReadBy:    []string{userID},
		CreatedAt: time.Now(),
	}

	if err := uc.chatRepo.CreateMessage(ctx, message); err != nil {
		log.Printf("SendMessage Error: Failed to create message for chat %s: %v", input.ChatID, err)
		return nil, err
	}

	chat.LastMessage = input.Content
	chat.LastMessageAt = message.CreatedAt

	if chat.UnreadCount == nil {
		chat.UnreadCount = make(map[string]int)
	}

	for _, participantID := range chat.Participants {
		if participantID != userID {
			chat.UnreadCount[participantID]++
		}
	}

	if err := uc.chatRepo.Update(ctx, chat); err != nil {
		log.Printf("SendMessage Error: Failed to update chat %s with last message: %v", chat.ID, err)
		return nil, err
	}

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
		log.Printf("GetUserChats Error: Failed to list chats for user %s: %v", userID, err)
		return nil, 0, err
	}

	var chatResponses []*ChatResponse

	for _, chat := range chats {
		chatResp := &ChatResponse{Chat: chat}

		if chat.ProductID != "" {
			product, err := uc.productRepo.GetByID(ctx, chat.ProductID)
			if err == nil {
				chatResp.Product = product
			} else {
				log.Printf("GetUserChats Warning: Product %s not found for chat %s: %v", chat.ProductID, chat.ID, err)
			}
		}

		for _, participantID := range chat.Participants {
			if participantID != userID {
				otherUser, err := uc.userRepo.GetByID(ctx, participantID)
				if err == nil {
					chatResp.OtherUser = otherUser
				} else {
					log.Printf("GetUserChats Warning: Other user %s not found for chat %s: %v", participantID, chat.ID, err)
				}
				break
			}
		}

		chatResponses = append(chatResponses, chatResp)
	}

	return chatResponses, total, nil
}

func (uc *ChatUseCase) GetChatMessages(ctx context.Context, userID, chatID string, limit, offset int) ([]*MessageResponse, int64, error) {
	chat, err := uc.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		log.Printf("GetChatMessages Error: Chat %s not found: %v", chatID, err)
		return nil, 0, err
	}

	if !containsString(chat.Participants, userID) {
		log.Printf("GetChatMessages Error: User %s is not a participant in chat %s", userID, chatID)
		return nil, 0, errors.Forbidden("User is not a participant in this chat", nil)
	}

	messages, total, err := uc.chatRepo.GetMessagesByChat(ctx, chatID, limit, offset)
	if err != nil {
		log.Printf("GetChatMessages Error: Failed to get messages for chat %s: %v", chatID, err)
		return nil, 0, err
	}

	var messageResponses []*MessageResponse

	for _, message := range messages {
		messageResp := &MessageResponse{Message: message}

		sender, err := uc.userRepo.GetByID(ctx, message.SenderID)
		if err == nil {
			messageResp.Sender = sender
		} else {
			log.Printf("GetChatMessages Warning: Sender %s not found for message %s in chat %s: %v", message.SenderID, message.ID, chatID, err)
		}

		messageResponses = append(messageResponses, messageResp)
	}

	return messageResponses, total, nil
}

func (uc *ChatUseCase) MarkChatAsRead(ctx context.Context, userID, chatID string) error {
	chat, err := uc.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		log.Printf("MarkChatAsRead Error: Chat %s not found: %v", chatID, err)
		return err
	}

	if !containsString(chat.Participants, userID) {
		log.Printf("MarkChatAsRead Error: User %s is not a participant in chat %s", userID, chatID)
		return errors.Forbidden("User is not a participant in this chat", nil)
	}

	if chat.UnreadCount == nil {
		chat.UnreadCount = make(map[string]int)
	}
	chat.UnreadCount[userID] = 0

	if err := uc.chatRepo.Update(ctx, chat); err != nil {
		log.Printf("MarkChatAsRead Error: Failed to update chat %s unread count for user %s: %v", chatID, userID, err)
		return err
	}

	return nil
}

func (uc *ChatUseCase) GetChatByID(ctx context.Context, userID, chatID string) (*ChatResponse, error) {
	chat, err := uc.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		log.Printf("GetChatByID Error: Chat %s not found: %v", chatID, err)
		return nil, err
	}

	if !containsString(chat.Participants, userID) {
		log.Printf("GetChatByID Error: User %s is not a participant in chat %s", userID, chatID)
		return nil, errors.Forbidden("User is not a participant in this chat", nil)
	}

	chatResp := &ChatResponse{Chat: chat}

	if chat.ProductID != "" {
		product, err := uc.productRepo.GetByID(ctx, chat.ProductID)
		if err == nil {
			chatResp.Product = product
		} else {
			log.Printf("GetChatByID Warning: Product %s not found for chat %s: %v", chat.ProductID, chat.ID, err)
		}
	}

	for _, participantID := range chat.Participants {
		if participantID != userID {
			otherUser, err := uc.userRepo.GetByID(ctx, participantID)
			if err == nil {
				chatResp.OtherUser = otherUser
			} else {
				log.Printf("GetChatByID Warning: Other user %s not found for chat %s: %v", participantID, chat.ID, err)
			}
			break
		}
	}

	return chatResp, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
