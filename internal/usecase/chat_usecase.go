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

// Updated: Added ProductID, AttachmentURL, Metadata to SendMessageInput
type SendMessageInput struct {
	ChatID        string
	Content       string
	Type          string                 // "text", "image", "system", "offer"
	AttachmentURL string                 // For image/file sharing
	Metadata      map[string]interface{} // For offer details, system message data etc.
	ProductID     string                 // ProductID associated with this specific message
}

// New: Input for creating a middleman chat
type CreateMiddlemanChatInput struct {
	BuyerID        string
	SellerID       string
	MiddlemanID    string
	ProductID      string
	TransactionID  string
	InitialMessage string
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
		chatToReturn = existingChat
	} else {
		if err != nil && !errors.Is(err, "NOT_FOUND") {
			log.Printf("CreateChat Error: Failed to search for existing chat: %v", err)
			return nil, err
		}

		newChat := &entity.Chat{
			Participants:  []string{userID, input.RecipientID},
			ProductID:     input.ProductID, // Initial product context for the chat
			Type:          "direct",
			UnreadCount:   make(map[string]int),
			LastMessageAt: time.Now(),
		}

		if err := uc.chatRepo.Create(ctx, newChat); err != nil {
			log.Printf("CreateChat Error: Failed to create new chat in repository: %v", err)
			return nil, err
		}
		chatToReturn = newChat
	}

	if input.InitialMessage != "" {
		messageResp, err := uc.SendMessage(ctx, userID, SendMessageInput{
			ChatID:    chatToReturn.ID,
			Content:   input.InitialMessage,
			Type:      "text",
			ProductID: input.ProductID, // Pass ProductID to the initial message
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
	}

	return &ChatResponse{
		Chat:      chatToReturn,
		Product:   product,
		OtherUser: recipient,
	}, nil
}

// New: CreateMiddlemanChat creates a group chat for a transaction involving a middleman
func (uc *ChatUseCase) CreateMiddlemanChat(ctx context.Context, input CreateMiddlemanChatInput) (*ChatResponse, error) {
	// Validate participants exist (removed unused variable assignments)
	if _, err := uc.userRepo.GetByID(ctx, input.BuyerID); err != nil {
		return nil, errors.NotFound("Buyer not found", err)
	}
	if _, err := uc.userRepo.GetByID(ctx, input.SellerID); err != nil {
		return nil, errors.NotFound("Seller not found", err)
	}
	if _, err := uc.userRepo.GetByID(ctx, input.MiddlemanID); err != nil {
		return nil, errors.NotFound("Middleman not found", err)
	}

	// Check if a middleman chat for this transaction already exists
	// For simplicity, we assume one middleman chat per transaction
	existingChat, err := uc.chatRepo.GetChatByTransactionID(ctx, input.TransactionID)
	if err == nil && existingChat != nil {
		log.Printf("CreateMiddlemanChat: Found existing middleman chat %s for transaction %s. Reusing it.", existingChat.ID, input.TransactionID)
		return &ChatResponse{Chat: existingChat}, nil
	}
	if err != nil && !errors.Is(err, "NOT_FOUND") {
		return nil, err // Propagate actual error
	}

	// Create new middleman chat
	chat := &entity.Chat{
		Participants:  []string{input.BuyerID, input.SellerID, input.MiddlemanID},
		ProductID:     input.ProductID,
		TransactionID: input.TransactionID,
		Type:          "middleman", // Set type to "middleman"
		UnreadCount:   make(map[string]int),
		LastMessageAt: time.Now(),
	}

	if err := uc.chatRepo.Create(ctx, chat); err != nil {
		return nil, err
	}

	// Send initial system message to the middleman chat
	if input.InitialMessage != "" {
		_, err := uc.SendSystemMessage(ctx, chat.ID, input.InitialMessage, "transaction_init", nil)
		if err != nil {
			log.Printf("CreateMiddlemanChat: Failed to send initial system message for chat %s: %v", chat.ID, err)
			// Don't return error, chat is already created
		}
		chat.LastMessage = input.InitialMessage
		chat.LastMessageAt = time.Now() // Update last message time for system message
		if err := uc.chatRepo.Update(ctx, chat); err != nil {
			log.Printf("CreateMiddlemanChat: Failed to update chat %s with last system message: %v", chat.ID, err)
		}
	}

	return &ChatResponse{
		Chat:      chat,
		Product:   nil, // Product info might be fetched separately if needed for response
		OtherUser: nil, // No single "other user" in group chat context
	}, nil
}

func (uc *ChatUseCase) findExistingChat(ctx context.Context, userID1, userID2 string) (*entity.Chat, error) {
	chats1, _, err := uc.chatRepo.ListByUserID(ctx, userID1, -1, 0)
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

// Updated: SendMessage now accepts ProductID, AttachmentURL, and Metadata
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
		ChatID:        input.ChatID,
		SenderID:      userID,
		Content:       input.Content,
		Type:          input.Type,
		AttachmentURL: input.AttachmentURL, // New: Save attachment URL
		Metadata:      input.Metadata,      // New: Save metadata
		ProductID:     input.ProductID,     // Save ProductID in the message
		ReadBy:        []string{userID},
		CreatedAt:     time.Now(),
	}

	if err := uc.chatRepo.CreateMessage(ctx, message); err != nil {
		log.Printf("SendMessage Error: Failed to create message for chat %s: %v", input.ChatID, err)
		return nil, err
	}

	// Update chat's last message info (only for text/offer/image messages, not system messages)
	if message.Type != "system" {
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

// New: SendSystemMessage sends a system-generated message to a chat
func (uc *ChatUseCase) SendSystemMessage(ctx context.Context, chatID, content, systemType string, metadata map[string]interface{}) (*MessageResponse, error) {
	// System messages are sent by "system" (no specific user ID)
	// You might want a dedicated "system" user ID or just leave SenderID empty/null
	// For now, we'll use a placeholder "system" ID or an empty string.
	systemUserID := "system" // Or a specific admin ID if system messages are sent by an admin

	message := &entity.Message{
		ChatID:    chatID,
		SenderID:  systemUserID, // Sender is "system"
		Content:   content,
		Type:      "system", // Type is "system"
		Metadata:  metadata,
		ReadBy:    []string{}, // System messages are not "read" by users in the same way
		CreatedAt: time.Now(),
	}

	if err := uc.chatRepo.CreateMessage(ctx, message); err != nil {
		log.Printf("SendSystemMessage Error: Failed to create system message for chat %s: %v", chatID, err)
		return nil, err
	}

	// Update chat's last message info for system messages
	chat, err := uc.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		log.Printf("SendSystemMessage Error: Chat %s not found for system message: %v", chatID, err)
		return nil, err
	}
	chat.LastMessage = content
	chat.LastMessageAt = message.CreatedAt
	if err := uc.chatRepo.Update(ctx, chat); err != nil {
		log.Printf("SendSystemMessage Error: Failed to update chat %s with last system message: %v", chat.ID, err)
	}

	// Notify all participants about the new system message
	notification := map[string]interface{}{
		"type":    "new_message",
		"chat_id": chatID,
		"message": message,
		"sender":  map[string]string{"id": systemUserID, "username": "System"}, // Placeholder sender info
	}
	notificationJSON, _ := json.Marshal(notification)
	for _, participantID := range chat.Participants {
		uc.wsManager.SendToUser(participantID, notificationJSON)
	}

	return &MessageResponse{Message: message}, nil
}

// New: AcceptOffer updates the status of an offer message
func (uc *ChatUseCase) AcceptOffer(ctx context.Context, chatID, messageID, userID string) error {
	// 1. Get the message
	message, err := uc.chatRepo.GetMessageByID(ctx, chatID, messageID) // Assuming GetMessageByID exists or will be added
	if err != nil {
		return errors.NotFound("Offer message not found", err)
	}

	// 2. Validate it's an offer message
	if message.Type != "offer" {
		return errors.BadRequest("Message is not an offer", nil)
	}

	// 3. Check current offer status
	if status, ok := message.Metadata["status"].(string); ok && status != "pending" {
		return errors.BadRequest("Offer is not pending", nil)
	}

	// 4. Check if the user is the recipient of the offer (not the sender)
	// This logic depends on how you define offer sender/recipient.
	// For now, let's assume the other participant in a direct chat can accept.
	chat, err := uc.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return err
	}
	isRecipient := false
	for _, p := range chat.Participants {
		if p != message.SenderID && p == userID { // If current user is a participant but not the sender
			isRecipient = true
			break
		}
	}
	if !isRecipient {
		return errors.Forbidden("Only the offer recipient can accept/reject", nil)
	}

	// 5. Update offer status to "accepted"
	if message.Metadata == nil {
		message.Metadata = make(map[string]interface{})
	}
	message.Metadata["status"] = "accepted"
	message.Metadata["accepted_by"] = userID
	message.Metadata["accepted_at"] = time.Now()

	if err := uc.chatRepo.UpdateMessage(ctx, chatID, message); err != nil { // Assuming UpdateMessage exists or will be added
		return errors.Internal("Failed to accept offer", err)
	}

	// 6. Send a system message to the chat about the accepted offer
	acceptedByUsername := ""
	if user, uErr := uc.userRepo.GetByID(ctx, userID); uErr == nil {
		acceptedByUsername = user.Username
	}
	uc.SendSystemMessage(ctx, chatID, acceptedByUsername+" accepted the offer.", "offer_accepted", message.Metadata)

	// 7. Notify participants via WebSocket
	notification := map[string]interface{}{
		"type":        "offer_update",
		"chat_id":     chatID,
		"message_id":  messageID,
		"status":      "accepted",
		"accepted_by": userID,
		"metadata":    message.Metadata,
	}
	notificationJSON, _ := json.Marshal(notification)
	uc.wsManager.SendToChatRoom(chatID, notificationJSON, "") // Send to all in room

	return nil
}

// New: RejectOffer updates the status of an offer message to rejected
func (uc *ChatUseCase) RejectOffer(ctx context.Context, chatID, messageID, userID string) error {
	message, err := uc.chatRepo.GetMessageByID(ctx, chatID, messageID)
	if err != nil {
		return errors.NotFound("Offer message not found", err)
	}

	if message.Type != "offer" {
		return errors.BadRequest("Message is not an offer", nil)
	}

	if status, ok := message.Metadata["status"].(string); ok && status != "pending" {
		return errors.BadRequest("Offer is not pending", nil)
	}

	chat, err := uc.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return err
	}
	isRecipient := false
	for _, p := range chat.Participants {
		if p != message.SenderID && p == userID {
			isRecipient = true
			break
		}
	}
	if !isRecipient {
		return errors.Forbidden("Only the offer recipient can accept/reject", nil)
	}

	if message.Metadata == nil {
		message.Metadata = make(map[string]interface{})
	}
	message.Metadata["status"] = "rejected"
	message.Metadata["rejected_by"] = userID
	message.Metadata["rejected_at"] = time.Now()

	if err := uc.chatRepo.UpdateMessage(ctx, chatID, message); err != nil {
		return errors.Internal("Failed to reject offer", err)
	}

	rejectedByUsername := ""
	if user, uErr := uc.userRepo.GetByID(ctx, userID); uErr == nil {
		rejectedByUsername = user.Username
	}
	uc.SendSystemMessage(ctx, chatID, rejectedByUsername+" rejected the offer.", "offer_rejected", message.Metadata)

	notification := map[string]interface{}{
		"type":        "offer_update",
		"chat_id":     chatID,
		"message_id":  messageID,
		"status":      "rejected",
		"rejected_by": userID,
		"metadata":    message.Metadata,
	}
	notificationJSON, _ := json.Marshal(notification)
	uc.wsManager.SendToChatRoom(chatID, notificationJSON, "")

	return nil
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

func (uc *ChatUseCase) HandleTypingEvent(ctx context.Context, userID, chatID string, isTyping bool) {
	chat, err := uc.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		log.Printf("HandleTypingEvent Error: Chat %s not found: %v", chatID, err)
		return
	}
	if !containsString(chat.Participants, userID) {
		log.Printf("HandleTypingEvent Error: User %s is not a participant in chat %s", userID, chatID)
		return
	}

	sender, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		log.Printf("HandleTypingEvent Error: Sender %s not found: %v", userID, err)
		return
	}

	notification := map[string]interface{}{
		"type":      "typing_indicator",
		"chat_id":   chatID,
		"user_id":   userID,
		"username":  sender.Username,
		"is_typing": isTyping,
	}

	notificationJSON, _ := json.Marshal(notification)
	uc.wsManager.SendToChatRoom(chatID, notificationJSON, userID)
}

func (uc *ChatUseCase) MarkMessageAsRead(ctx context.Context, chatID, messageID, userID string) {
	chat, err := uc.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		log.Printf("MarkMessageAsRead Error: Chat %s not found: %v", chatID, err)
		return
	}
	if !containsString(chat.Participants, userID) {
		log.Printf("MarkMessageAsRead Error: User %s is not a participant in chat %s", userID, chatID)
		return
	}

	err = uc.chatRepo.UpdateMessageReadStatus(ctx, messageID, userID)
	if err != nil {
		log.Printf("MarkMessageAsRead Error: Failed to update message %s read status for user %s: %v", messageID, userID, err)
		return
	}

	notification := map[string]interface{}{
		"type":       "message_read_receipt",
		"chat_id":    chatID,
		"message_id": messageID,
		"reader_id":  userID,
	}
	notificationJSON, _ := json.Marshal(notification)
	uc.wsManager.SendToChatRoom(chatID, notificationJSON, userID)
}

func (uc *ChatUseCase) HandleUserPresence(ctx context.Context, userID, chatID string, isOnline bool) {
	chat, err := uc.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		log.Printf("HandleUserPresence Error: Chat %s not found: %v", chatID, err)
		return
	}
	if !containsString(chat.Participants, userID) {
		log.Printf("HandleUserPresence Error: User %s is not a participant in chat %s", userID, chatID)
		return
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		log.Printf("HandleUserPresence Error: User %s not found: %v", userID, err)
		return
	}

	notification := map[string]interface{}{
		"type":      "user_presence",
		"chat_id":   chatID,
		"user_id":   userID,
		"username":  user.Username,
		"is_online": isOnline,
	}

	notificationJSON, _ := json.Marshal(notification)
	uc.wsManager.SendToChatRoom(chatID, notificationJSON, userID)
}
