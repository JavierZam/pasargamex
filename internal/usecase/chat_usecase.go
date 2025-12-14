package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/internal/infrastructure/ratelimit"
	ws "pasargamex/internal/infrastructure/websocket"
	"pasargamex/pkg/errors"
)

type ChatUseCase struct {
	chatRepo    repository.ChatRepository
	userRepo    repository.UserRepository
	productRepo repository.ProductRepository
	wsManager   *ws.Manager
	rateLimiter *ratelimit.RateLimiter
}

func NewChatUseCase(
	chatRepo repository.ChatRepository,
	userRepo repository.UserRepository,
	productRepo repository.ProductRepository,
	wsManager *ws.Manager,
) *ChatUseCase {
	rateLimiter := ratelimit.NewRateLimiter()
	rateLimiter.StartCleanupRoutine() // Start cleanup routine

	return &ChatUseCase{
		chatRepo:    chatRepo,
		userRepo:    userRepo,
		productRepo: productRepo,
		wsManager:   wsManager,
		rateLimiter: rateLimiter,
	}
}

type CreateChatInput struct {
	RecipientID    string
	ProductID      string
	InitialMessage string
}

// New: CreateChatByProductInput for creating chat via product ID (auto-detect seller)
type CreateChatByProductInput struct {
	ProductID      string
	InitialMessage string
}

// New: CreateGroupChatInput for buyer-initiated group chat with auto middleman invite
type CreateGroupChatInput struct {
	ProductID      string
	MiddlemanID    string // Buyer selects preferred middleman
	InitialMessage string
}

// New: CreateSellerGroupChatInput for seller-initiated group chat
type CreateSellerGroupChatInput struct {
	ProductID      string
	BuyerID        string // Seller invites specific buyer
	MiddlemanID    string // Seller selects preferred middleman
	InitialMessage string
}

// New: CreateTransactionChatInput for universal transaction chat creation
type CreateTransactionChatInput struct {
	BuyerID        string
	SellerID       string
	ProductID      string
	MiddlemanID    string
	InitialMessage string
}

// Updated: Added ProductID, AttachmentURL, Metadata to SendMessageInput
type SendMessageInput struct {
	ChatID         string
	Content        string
	Type           string                 // "text", "image", "system", "offer"
	AttachmentURL  string                 // Deprecated: For single image/file sharing
	AttachmentURLs []string               // New: For multiple image/file sharing
	Metadata       map[string]interface{} // For offer details, system message data etc.
	ProductID      string                 // ProductID associated with this specific message
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
	// Rate limiting check
	allowed, waitTime := uc.rateLimiter.Allow(userID, "create_chat")
	if !allowed {
		log.Printf("CreateChat Rate Limited: User %s must wait %v", userID, waitTime)
		return nil, errors.TooManyRequests("Rate limit exceeded. Please wait before creating another chat", waitTime)
	}

	// Prevent self-chat
	if userID == input.RecipientID {
		log.Printf("CreateChat Error: User %s attempted to create chat with themselves", userID)
		return nil, errors.BadRequest("You cannot create a chat with yourself", nil)
	}

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
	// Validate that all participants are different users
	if input.BuyerID == input.SellerID || input.BuyerID == input.MiddlemanID || input.SellerID == input.MiddlemanID {
		log.Printf("CreateMiddlemanChat Error: Duplicate participants detected - Buyer: %s, Seller: %s, Middleman: %s",
			input.BuyerID, input.SellerID, input.MiddlemanID)
		return nil, errors.BadRequest("All participants must be different users", nil)
	}

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

	// Create new middleman chat with role assignments
	chat := &entity.Chat{
		Participants:  []string{input.BuyerID, input.SellerID, input.MiddlemanID},
		ProductID:     input.ProductID,
		TransactionID: input.TransactionID,
		Type:          "middleman", // Set type to "middleman"
		SellerID:      input.SellerID,
		BuyerID:       input.BuyerID,
		MiddlemanID:   input.MiddlemanID,
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

// New: CreateBuyerGroupChat - Buyer initiates group chat with product seller and chosen middleman
func (uc *ChatUseCase) CreateBuyerGroupChat(ctx context.Context, buyerID string, input CreateGroupChatInput) (*ChatResponse, error) {
	// Validate product exists and get seller ID
	product, err := uc.productRepo.GetByID(ctx, input.ProductID)
	if err != nil {
		return nil, errors.NotFound("Product not found", err)
	}

	// Validate seller exists
	seller, err := uc.userRepo.GetByID(ctx, product.SellerID)
	if err != nil {
		return nil, errors.NotFound("Product seller not found", err)
	}

	// Validate buyer exists
	buyer, err := uc.userRepo.GetByID(ctx, buyerID)
	if err != nil {
		return nil, errors.NotFound("Buyer not found", err)
	}

	// Validate middleman exists and is admin
	middleman, err := uc.userRepo.GetByID(ctx, input.MiddlemanID)
	if err != nil {
		return nil, errors.NotFound("Middleman not found", err)
	}
	if middleman.Role != "admin" {
		return nil, errors.BadRequest("Selected middleman must be an admin", nil)
	}

	// Check if group chat already exists for this combination
	existingChat, err := uc.chatRepo.GetGroupChatByProductAndParticipants(ctx, input.ProductID, []string{buyerID, product.SellerID, input.MiddlemanID})
	if err == nil && existingChat != nil {
		return &ChatResponse{Chat: existingChat}, nil
	}

	// Create new group chat
	chat := &entity.Chat{
		Participants:  []string{buyerID, product.SellerID, input.MiddlemanID},
		ProductID:     input.ProductID,
		Type:          "group_transaction", // New type for buyer-initiated group chats
		UnreadCount:   make(map[string]int),
		LastMessageAt: time.Now(),
	}

	if err := uc.chatRepo.Create(ctx, chat); err != nil {
		return nil, err
	}

	// Send initial system message
	systemMessage := fmt.Sprintf("🛍️ Group transaction chat created!\n\n👤 Buyer: %s\n🏪 Seller: %s\n⚖️ Middleman: %s\n🎮 Product: %s\n\n%s",
		buyer.Username, seller.Username, middleman.Username, product.Title, input.InitialMessage)

	_, err = uc.SendSystemMessage(ctx, chat.ID, systemMessage, "group_chat_init", map[string]interface{}{
		"buyer_id":     buyerID,
		"seller_id":    product.SellerID,
		"middleman_id": input.MiddlemanID,
		"product_id":   input.ProductID,
		"initiated_by": "buyer",
	})
	if err != nil {
		log.Printf("CreateBuyerGroupChat: Failed to send initial system message: %v", err)
	}

	// Update chat with last message
	chat.LastMessage = systemMessage
	chat.LastMessageAt = time.Now()
	if err := uc.chatRepo.Update(ctx, chat); err != nil {
		log.Printf("CreateBuyerGroupChat: Failed to update chat: %v", err)
	}

	// Send notifications to all participants
	uc.notifyGroupChatCreated(ctx, chat, "buyer", buyer.Username, product.Title)

	return &ChatResponse{Chat: chat}, nil
}

// New: CreateSellerGroupChat - Seller initiates group chat with chosen buyer and middleman
func (uc *ChatUseCase) CreateSellerGroupChat(ctx context.Context, sellerID string, input CreateSellerGroupChatInput) (*ChatResponse, error) {
	// Validate product exists and belongs to seller
	product, err := uc.productRepo.GetByID(ctx, input.ProductID)
	if err != nil {
		return nil, errors.NotFound("Product not found", err)
	}
	if product.SellerID != sellerID {
		return nil, errors.Forbidden("You can only create group chats for your own products", nil)
	}

	// Validate buyer exists
	buyer, err := uc.userRepo.GetByID(ctx, input.BuyerID)
	if err != nil {
		return nil, errors.NotFound("Buyer not found", err)
	}

	// Validate seller exists
	seller, err := uc.userRepo.GetByID(ctx, sellerID)
	if err != nil {
		return nil, errors.NotFound("Seller not found", err)
	}

	// Validate middleman exists and is admin
	middleman, err := uc.userRepo.GetByID(ctx, input.MiddlemanID)
	if err != nil {
		return nil, errors.NotFound("Middleman not found", err)
	}
	if middleman.Role != "admin" {
		return nil, errors.BadRequest("Selected middleman must be an admin", nil)
	}

	// Check if group chat already exists for this combination
	existingChat, err := uc.chatRepo.GetGroupChatByProductAndParticipants(ctx, input.ProductID, []string{input.BuyerID, sellerID, input.MiddlemanID})
	if err == nil && existingChat != nil {
		return &ChatResponse{Chat: existingChat}, nil
	}

	// Create new group chat
	chat := &entity.Chat{
		Participants:  []string{input.BuyerID, sellerID, input.MiddlemanID},
		ProductID:     input.ProductID,
		Type:          "group_transaction", // Same type as buyer-initiated
		UnreadCount:   make(map[string]int),
		LastMessageAt: time.Now(),
	}

	if err := uc.chatRepo.Create(ctx, chat); err != nil {
		return nil, err
	}

	// Send initial system message
	systemMessage := fmt.Sprintf("🏪 Seller invitation to group transaction!\n\n🏪 Seller: %s\n👤 Buyer: %s\n⚖️ Middleman: %s\n🎮 Product: %s\n\n%s",
		seller.Username, buyer.Username, middleman.Username, product.Title, input.InitialMessage)

	_, err = uc.SendSystemMessage(ctx, chat.ID, systemMessage, "group_chat_init", map[string]interface{}{
		"buyer_id":     input.BuyerID,
		"seller_id":    sellerID,
		"middleman_id": input.MiddlemanID,
		"product_id":   input.ProductID,
		"initiated_by": "seller",
	})
	if err != nil {
		log.Printf("CreateSellerGroupChat: Failed to send initial system message: %v", err)
	}

	// Update chat with last message
	chat.LastMessage = systemMessage
	chat.LastMessageAt = time.Now()
	if err := uc.chatRepo.Update(ctx, chat); err != nil {
		log.Printf("CreateSellerGroupChat: Failed to update chat: %v", err)
	}

	// Send notifications to all participants
	uc.notifyGroupChatCreated(ctx, chat, "seller", seller.Username, product.Title)

	return &ChatResponse{Chat: chat}, nil
}

// Helper function to notify participants about new group chat
func (uc *ChatUseCase) notifyGroupChatCreated(ctx context.Context, chat *entity.Chat, initiatedBy, initiatorName, productTitle string) {
	notification := map[string]interface{}{
		"type":          "group_chat_created",
		"chat_id":       chat.ID,
		"product_id":    chat.ProductID,
		"product_title": productTitle,
		"initiated_by":  initiatedBy,
		"initiator":     initiatorName,
		"participants":  chat.Participants,
	}

	// Send WebSocket notification to all participants
	notificationBytes, _ := json.Marshal(notification)
	for _, participantID := range chat.Participants {
		uc.wsManager.SendToUser(participantID, notificationBytes)
	}
}

// New: ListAvailableMiddlemen - Get list of available admin users for middleman selection
func (uc *ChatUseCase) ListAvailableMiddlemen(ctx context.Context) ([]*entity.User, error) {
	return uc.chatRepo.ListAdminUsers(ctx)
}

// New: GetUserRoleInChat - Determine user's role in a chat using stored role fields
func (uc *ChatUseCase) GetUserRoleInChat(ctx context.Context, chatID, userID string) (string, error) {
	chat, err := uc.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return "", err
	}

	// Check if user is participant
	if !containsString(chat.Participants, userID) {
		return "", errors.Forbidden("User is not a participant in this chat", nil)
	}

	// Use stored role fields for group/middleman chats
	if chat.Type == "group_transaction" || chat.Type == "middleman" {
		if userID == chat.SellerID {
			return "seller", nil
		}
		if userID == chat.BuyerID {
			return "buyer", nil
		}
		if userID == chat.MiddlemanID {
			return "middleman", nil
		}
	}

	// For direct chats, try to determine seller/buyer from product context
	if chat.Type == "direct" && chat.ProductID != "" {
		product, err := uc.productRepo.GetByID(ctx, chat.ProductID)
		if err == nil {
			if userID == product.SellerID {
				return "seller", nil
			} else {
				return "buyer", nil
			}
		}
	}

	// Default to participant for direct chats without product context
	return "participant", nil
}

// New: GetChatParticipantsWithRoles - Get chat participants with their roles
func (uc *ChatUseCase) GetChatParticipantsWithRoles(ctx context.Context, chatID string) (map[string]interface{}, error) {
	chat, err := uc.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"chat_id":      chatID,
		"chat_type":    chat.Type,
		"product_id":   chat.ProductID,
		"seller_id":    chat.SellerID,
		"buyer_id":     chat.BuyerID,
		"middleman_id": chat.MiddlemanID,
		"participants": make([]map[string]interface{}, 0),
	}

	var product *entity.Product
	if chat.ProductID != "" {
		product, _ = uc.productRepo.GetByID(ctx, chat.ProductID)
		if product != nil {
			result["product_title"] = product.Title
			result["product_price"] = product.Price
		}
	}

	for _, participantID := range chat.Participants {
		user, err := uc.userRepo.GetByID(ctx, participantID)
		if err != nil {
			continue // Skip if user not found
		}

		role, _ := uc.GetUserRoleInChat(ctx, chatID, participantID)

		participantInfo := map[string]interface{}{
			"user_id":  participantID,
			"username": user.Username,
			"email":    user.Email,
			"role":     role,
		}

		result["participants"] = append(result["participants"].([]map[string]interface{}), participantInfo)
	}

	return result, nil
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
	// Rate limiting check
	allowed, waitTime := uc.rateLimiter.Allow(userID, "send_message")
	if !allowed {
		log.Printf("SendMessage Rate Limited: User %s must wait %v", userID, waitTime)

		// Send rate limit notification via WebSocket
		notification := map[string]interface{}{
			"type":      "rate_limit_exceeded",
			"message":   "You are sending messages too quickly. Please slow down.",
			"wait_time": waitTime.Seconds(),
		}
		notificationJSON, _ := json.Marshal(notification)
		uc.wsManager.SendToUser(userID, notificationJSON)

		return nil, errors.TooManyRequests("Rate limit exceeded. Please wait before sending another message", waitTime)
	}

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

	// Determine attachment URLs (use new field if provided, fallback to old field)
	var attachmentURLs []string
	if len(input.AttachmentURLs) > 0 {
		attachmentURLs = input.AttachmentURLs
	} else if input.AttachmentURL != "" {
		attachmentURLs = []string{input.AttachmentURL}
	}

	message := &entity.Message{
		ChatID:         input.ChatID,
		SenderID:       userID,
		Content:        input.Content,
		Type:           input.Type,
		Status:         "sent",              // Initial status is 'sent'
		AttachmentURL:  input.AttachmentURL, // Backward compatibility
		AttachmentURLs: attachmentURLs,      // New: Multiple attachments
		Metadata:       input.Metadata,      // New: Save metadata
		ProductID:      input.ProductID,     // Save ProductID in the message
		ReadBy:         []string{userID},
		CreatedAt:      time.Now(),
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

	// Use SendToChatRoom to broadcast to all connected participants in the chat room
	// This is for users who have the chat room open
	log.Printf("SendMessage: Broadcasting new message to chat room %s (excluding sender %s)", input.ChatID, userID)
	uc.wsManager.SendToChatRoom(input.ChatID, notificationJSON, userID)

	// ALSO send via SendToUser for chat list updates
	// This ensures users who are on the chat list page (not in the room) also get the update
	chatListUpdate := map[string]interface{}{
		"type":            "chat_list_update",
		"chat_id":         input.ChatID,
		"last_message":    message.Content,
		"last_message_at": message.CreatedAt.Format(time.RFC3339),
		"sender_id":       userID,
		"sender_name":     sender.Username,
		"message_type":    message.Type,
	}
	chatListUpdateJSON, _ := json.Marshal(chatListUpdate)

	log.Printf("SendMessage: Sending chat_list_update to %d participants (excluding sender %s)", len(chat.Participants)-1, userID)
	for _, participantID := range chat.Participants {
		if participantID != userID {
			log.Printf("SendMessage: Sending chat_list_update to participant %s", participantID)
			uc.wsManager.SendToUser(participantID, chatListUpdateJSON)
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
		Type:      "system",    // Type is "system"
		Status:    "delivered", // System messages are automatically delivered
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

	// Use SendToChatRoom to broadcast to all connected participants
	log.Printf("SendSystemMessage: Broadcasting to chat room %s", chatID)
	uc.wsManager.SendToChatRoom(chatID, notificationJSON, "")

	return &MessageResponse{Message: message}, nil
}

// New: AcceptOffer updates the status of an offer message
func (uc *ChatUseCase) AcceptOffer(ctx context.Context, chatID, messageID, userID string) error {
	log.Printf("DEBUG: AcceptOffer called - chatID: %s, messageID: %s, userID: %s", chatID, messageID, userID)

	// 1. Get the message
	message, err := uc.chatRepo.GetMessageByID(ctx, chatID, messageID)
	if err != nil {
		log.Printf("ERROR: Failed to get message %s: %v", messageID, err)
		return errors.NotFound("Offer message not found", err)
	}

	log.Printf("DEBUG: Found message - Type: %s, Metadata: %+v", message.Type, message.Metadata)

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

	// 6. Update product price if offer contains a new price and product ID
	log.Printf("DEBUG: Checking for price update in metadata: %+v", message.Metadata)

	// Try both "offered_price" and "price" for backward compatibility
	var newPrice float64
	var priceExists bool

	if val, exists := message.Metadata["offered_price"].(float64); exists {
		newPrice, priceExists = val, true
	} else if val, exists := message.Metadata["price"].(float64); exists {
		newPrice, priceExists = val, true
	}

	if priceExists {
		log.Printf("DEBUG: Found offered_price: %.0f", newPrice)

		if productID, productIDExists := message.Metadata["product_id"].(string); productIDExists {
			log.Printf("DEBUG: Found product_id: %s", productID)

			// Get the product
			product, err := uc.productRepo.GetByID(ctx, productID)
			if err == nil {
				// Store original price for transaction history
				originalPrice := product.Price
				message.Metadata["original_price"] = originalPrice

				log.Printf("DEBUG: Updating product price from %.0f to %.0f IDR", originalPrice, newPrice)

				// Update product with negotiated price
				product.Price = newPrice
				product.UpdatedAt = time.Now()

				if updateErr := uc.productRepo.Update(ctx, product); updateErr != nil {
					log.Printf("WARNING: Failed to update product price after offer acceptance: %v", updateErr)
					// Don't fail the entire operation, just log the warning
				} else {
					log.Printf("SUCCESS: Product %s price updated from %.0f to %.0f IDR after offer acceptance", productID, originalPrice, newPrice)
				}
			} else {
				log.Printf("ERROR: Failed to get product %s: %v", productID, err)
			}
		} else {
			log.Printf("DEBUG: No product_id found in metadata")
		}
	} else {
		log.Printf("DEBUG: No offered_price found in metadata")
	}

	// 7. Send a system message to the chat about the accepted offer
	acceptedByUsername := ""
	if user, uErr := uc.userRepo.GetByID(ctx, userID); uErr == nil {
		acceptedByUsername = user.Username
	}

	// Enhanced system message with price information
	systemMessage := acceptedByUsername + " accepted the offer."
	if newPrice, exists := message.Metadata["offered_price"].(float64); exists {
		systemMessage = fmt.Sprintf("%s Negotiated price: Rp %.0f", systemMessage, newPrice)
	}
	uc.SendSystemMessage(ctx, chatID, systemMessage, "offer_accepted", message.Metadata)

	// 8. Notify participants via WebSocket
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

// New: GetChatsByType - Get chats filtered by type (direct or group)
func (uc *ChatUseCase) GetChatsByType(ctx context.Context, userID string, chatType string, limit, offset int) ([]*ChatResponse, int64, error) {
	chats, _, err := uc.chatRepo.ListByUserID(ctx, userID, limit, offset)
	if err != nil {
		log.Printf("GetChatsByType Error: Failed to list chats for user %s: %v", userID, err)
		return nil, 0, err
	}

	var filteredChats []*entity.Chat

	// Filter chats by type
	for _, chat := range chats {
		if chatType == "direct" && chat.Type == "direct" {
			filteredChats = append(filteredChats, chat)
		} else if chatType == "group" && (chat.Type == "group_transaction" || chat.Type == "middleman") {
			filteredChats = append(filteredChats, chat)
		}
	}

	var chatResponses []*ChatResponse

	for _, chat := range filteredChats {
		chatResp := &ChatResponse{Chat: chat}

		if chat.ProductID != "" {
			product, err := uc.productRepo.GetByID(ctx, chat.ProductID)
			if err == nil {
				chatResp.Product = product
			} else {
				log.Printf("GetChatsByType Warning: Product %s not found for chat %s: %v", chat.ProductID, chat.ID, err)
			}
		}

		// For direct chats, find the other user
		if chat.Type == "direct" {
			for _, participantID := range chat.Participants {
				if participantID != userID {
					otherUser, err := uc.userRepo.GetByID(ctx, participantID)
					if err == nil {
						chatResp.OtherUser = otherUser
					} else {
						log.Printf("GetChatsByType Warning: Other user %s not found for chat %s: %v", participantID, chat.ID, err)
					}
					break
				}
			}
		}

		chatResponses = append(chatResponses, chatResp)
	}

	return chatResponses, int64(len(filteredChats)), nil
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
	// Rate limiting for typing events
	allowed, _ := uc.rateLimiter.Allow(userID, "typing")
	if !allowed {
		log.Printf("HandleTypingEvent Rate Limited: User %s", userID)
		return // Silently ignore excessive typing events
	}

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

	err = uc.chatRepo.UpdateMessageReadStatus(ctx, chatID, messageID, userID)
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

// New: ListUsers - Get all users for buyer/seller selection
func (uc *ChatUseCase) ListUsers(ctx context.Context) ([]*entity.User, error) {
	// Get admin users first as a fallback (since we know this method works)
	users := uc.userRepo.GetUserByRole(ctx, "", 100)
	if len(users) == 0 {
		return nil, errors.Internal("No users found", nil)
	}

	// Convert []*entity.User to match expected return type
	result := make([]*entity.User, len(users))
	for i, user := range users {
		result[i] = user
	}

	return result, nil
}

// New: GetUserProducts - Get products owned by specific user
func (uc *ChatUseCase) GetUserProducts(ctx context.Context, userID string) ([]*entity.Product, error) {
	products, _, err := uc.productRepo.ListBySellerID(ctx, userID, "", 100, 0)
	if err != nil {
		return nil, errors.Internal("Failed to get user products", err)
	}

	return products, nil
}

// New: CreateTransactionChat - Universal transaction chat creator
func (uc *ChatUseCase) CreateTransactionChat(ctx context.Context, creatorID string, input CreateTransactionChatInput) (*entity.Chat, error) {
	// Validate that all participants are different
	if input.BuyerID == input.SellerID {
		return nil, errors.BadRequest("Buyer and seller cannot be the same person", nil)
	}
	if input.BuyerID == input.MiddlemanID {
		return nil, errors.BadRequest("Buyer and middleman cannot be the same person", nil)
	}
	if input.SellerID == input.MiddlemanID {
		return nil, errors.BadRequest("Seller and middleman cannot be the same person", nil)
	}

	// Validate all participants exist
	buyer, err := uc.userRepo.GetByID(ctx, input.BuyerID)
	if err != nil {
		return nil, errors.BadRequest("Buyer not found", err)
	}

	seller, err := uc.userRepo.GetByID(ctx, input.SellerID)
	if err != nil {
		return nil, errors.BadRequest("Seller not found", err)
	}

	middleman, err := uc.userRepo.GetByID(ctx, input.MiddlemanID)
	if err != nil {
		return nil, errors.BadRequest("Middleman not found", err)
	}

	// Validate role constraints
	if buyer.Role == "admin" {
		return nil, errors.BadRequest("Admin users cannot be buyers in transactions", nil)
	}

	// Validate middleman has admin role
	if middleman.Role != "admin" {
		return nil, errors.BadRequest("Selected middleman must have admin role", nil)
	}

	// Validate product exists and belongs to seller
	product, err := uc.productRepo.GetByID(ctx, input.ProductID)
	if err != nil {
		return nil, errors.BadRequest("Product not found", err)
	}

	if product.SellerID != input.SellerID {
		return nil, errors.BadRequest("Product does not belong to selected seller", nil)
	}

	// Check for existing group chat with same participants and product
	participants := []string{input.BuyerID, input.SellerID, input.MiddlemanID}
	if existingChat, err := uc.chatRepo.GetGroupChatByProductAndParticipants(ctx, input.ProductID, participants); err == nil {
		return existingChat, nil // Return existing chat if found
	}

	// Create new transaction chat with role assignments
	chat := &entity.Chat{
		Type:         "group_transaction",
		ProductID:    input.ProductID,
		Participants: participants,
		SellerID:     input.SellerID,
		BuyerID:      input.BuyerID,
		MiddlemanID:  input.MiddlemanID,
		UnreadCount:  make(map[string]int),
	}

	if err := uc.chatRepo.Create(ctx, chat); err != nil {
		return nil, errors.Internal("Failed to create transaction chat", err)
	}

	// Send initial system message
	initialMessage := input.InitialMessage
	if initialMessage == "" {
		initialMessage = fmt.Sprintf("Transaction chat created for product: %s\nBuyer: %s\nSeller: %s\nMiddleman: %s",
			product.Title, buyer.Username, seller.Username, middleman.Username)
	}

	systemMessage := &entity.Message{
		ChatID:    chat.ID,
		SenderID:  "system",
		Content:   initialMessage,
		Type:      "system",
		Status:    "delivered", // System messages are automatically delivered
		ProductID: input.ProductID,
		Metadata: map[string]interface{}{
			"type":         "transaction_chat_created",
			"product_id":   input.ProductID,
			"product_name": product.Title,
			"buyer_id":     input.BuyerID,
			"seller_id":    input.SellerID,
			"middleman_id": input.MiddlemanID,
		},
	}

	if err := uc.chatRepo.CreateMessage(ctx, systemMessage); err != nil {
		log.Printf("Failed to create system message: %v", err)
	}

	// Send real-time notifications to all participants
	notification := map[string]interface{}{
		"type":      "transaction_chat_created",
		"chat_id":   chat.ID,
		"product":   product.Title,
		"buyer":     buyer.Username,
		"seller":    seller.Username,
		"middleman": middleman.Username,
		"creator":   creatorID,
	}

	notificationJSON, _ := json.Marshal(notification)
	for _, participantID := range participants {
		if participantID != creatorID {
			uc.wsManager.SendToUser(participantID, notificationJSON)
		}
	}

	return chat, nil
}

// Helper function to format price with thousand separators
func formatPrice(price float64) string {
	str := strconv.FormatFloat(price, 'f', 0, 64)

	// Add thousand separators
	n := len(str)
	if n <= 3 {
		return str
	}

	var result strings.Builder
	for i, digit := range str {
		if i > 0 && (n-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteRune(digit)
	}

	return result.String()
}

// New: CreateChatByProduct - Create direct chat with seller via product ID
func (uc *ChatUseCase) CreateChatByProduct(ctx context.Context, buyerID string, input CreateChatByProductInput) (*ChatResponse, error) {
	// Rate limiting check
	allowed, waitTime := uc.rateLimiter.Allow(buyerID, "create_chat")
	if !allowed {
		log.Printf("CreateChatByProduct Rate Limited: User %s must wait %v", buyerID, waitTime)
		return nil, errors.TooManyRequests("Rate limit exceeded. Please wait before creating another chat", waitTime)
	}

	// Get product to extract seller ID
	product, err := uc.productRepo.GetByID(ctx, input.ProductID)
	if err != nil {
		log.Printf("CreateChatByProduct Error: Product %s not found: %v", input.ProductID, err)
		return nil, errors.NotFound("Product not found", err)
	}

	// Prevent chat with yourself
	if buyerID == product.SellerID {
		log.Printf("CreateChatByProduct Error: User %s attempted to create chat with themselves via product %s", buyerID, input.ProductID)
		return nil, errors.BadRequest("You cannot create a chat with yourself", nil)
	}

	// Get seller details
	seller, err := uc.userRepo.GetByID(ctx, product.SellerID)
	if err != nil {
		log.Printf("CreateChatByProduct Error: Seller %s not found: %v", product.SellerID, err)
		return nil, errors.NotFound("Product seller not found", err)
	}

	// Check for existing chat between buyer and seller
	var chatToReturn *entity.Chat
	existingChat, err := uc.findExistingChat(ctx, buyerID, product.SellerID)
	if err == nil && existingChat != nil {
		chatToReturn = existingChat

		// Update existing chat with new product ID (always set to latest product being discussed)
		chatToReturn.ProductID = input.ProductID
		chatToReturn.UpdatedAt = time.Now()

		if err := uc.chatRepo.Update(ctx, chatToReturn); err != nil {
			log.Printf("CreateChatByProduct Warning: Failed to update chat with new product ID: %v", err)
		} else {
			log.Printf("CreateChatByProduct: Updated existing chat %s with new product %s", existingChat.ID, input.ProductID)
		}
	} else {
		if err != nil && !errors.Is(err, "NOT_FOUND") {
			log.Printf("CreateChatByProduct Error: Failed to search for existing chat: %v", err)
			return nil, err
		}

		// Create new chat
		newChat := &entity.Chat{
			Participants:  []string{buyerID, product.SellerID},
			ProductID:     input.ProductID,
			Type:          "direct",
			UnreadCount:   make(map[string]int),
			LastMessageAt: time.Now(),
		}

		if err := uc.chatRepo.Create(ctx, newChat); err != nil {
			log.Printf("CreateChatByProduct Error: Failed to create new chat: %v", err)
			return nil, err
		}
		chatToReturn = newChat
		log.Printf("CreateChatByProduct: Created new chat %s for product %s", newChat.ID, input.ProductID)
	}

	// Get buyer details for user embed
	buyer, err := uc.userRepo.GetByID(ctx, buyerID)
	if err != nil {
		log.Printf("CreateChatByProduct Error: Buyer %s not found: %v", buyerID, err)
		return nil, errors.NotFound("Buyer not found", err)
	}

	// Send initial message - simple text only since product info is in embed
	initialMessage := input.InitialMessage
	if initialMessage == "" {
		initialMessage = "Hi! I'm interested in this product!"
	}

	// Create product embed metadata for better display
	productEmbedMetadata := map[string]interface{}{
		"product_embed": map[string]interface{}{
			"product_id":      input.ProductID,
			"product_title":   product.Title,
			"product_price":   product.Price,
			"delivery_method": product.DeliveryMethod,
			"seller_id":       product.SellerID,
		},
		"buyer_info": map[string]interface{}{
			"buyer_id": buyerID,
			"username": buyer.Username,
			"email":    buyer.Email,
		},
	}

	messageResp, err := uc.SendMessage(ctx, buyerID, SendMessageInput{
		ChatID:    chatToReturn.ID,
		Content:   initialMessage,
		Type:      "text",
		ProductID: input.ProductID,
		Metadata:  productEmbedMetadata,
	})
	if err != nil {
		log.Printf("CreateChatByProduct Error: Failed to send initial message: %v", err)
		return nil, err
	}

	// Update chat with last message info
	chatToReturn.LastMessage = initialMessage
	chatToReturn.LastMessageAt = messageResp.CreatedAt
	if chatToReturn.UnreadCount == nil {
		chatToReturn.UnreadCount = make(map[string]int)
	}
	for _, participantID := range chatToReturn.Participants {
		if participantID != buyerID {
			chatToReturn.UnreadCount[participantID]++
		}
	}

	if err := uc.chatRepo.Update(ctx, chatToReturn); err != nil {
		log.Printf("CreateChatByProduct Error: Failed to update chat with last message: %v", err)
	}

	return &ChatResponse{
		Chat:      chatToReturn,
		Product:   product,
		OtherUser: seller,
	}, nil
}
