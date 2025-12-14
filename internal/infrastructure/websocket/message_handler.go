package websocket

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

// WebSocket Message Types
const (
	MessageTypePing            = "ping"
	MessageTypePong            = "pong"
	MessageTypeSendMessage     = "send_message"
	MessageTypeMessage         = "message"
	MessageTypeTyping          = "typing"
	MessageTypeJoinRoom        = "join_room"
	MessageTypeJoinChatRoom    = "join_chat_room" // Frontend format
	MessageTypeLeaveRoom       = "leave_room"
	MessageTypeLeaveChatRoom   = "leave_chat_room" // Frontend format
	MessageTypeMarkRead        = "mark_read"
	MessageTypeMarkMessageRead = "mark_message_read" // Frontend format
	MessageTypeReadReceipt     = "read_receipt"
	MessageTypeDeliveryReceipt = "delivery_receipt"
	MessageTypePresence        = "presence"
	MessageTypeUserPresence    = "user_presence"
)

// WebSocket Message Structure
type WSMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	ChatID    string      `json:"chat_id,omitempty"` // For frontend messages
	Timestamp string      `json:"timestamp"`
}

// Message Data Types
type SendMessageData struct {
	TempID    string `json:"temp_id"`
	ChatID    string `json:"chat_id"`
	Content   string `json:"content"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
}

type MessageData struct {
	ID             string                 `json:"id"`
	TempID         string                 `json:"temp_id,omitempty"`
	ChatID         string                 `json:"chat_id"`
	SenderID       string                 `json:"sender_id"`
	SenderName     string                 `json:"sender_name"`
	Content        string                 `json:"content"`
	Type           string                 `json:"type"`
	Status         string                 `json:"status"`
	Timestamp      string                 `json:"timestamp"`
	AttachmentURLs []string               `json:"attachment_urls,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type TypingData struct {
	ChatID    string `json:"chat_id"`
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	Typing    bool   `json:"typing"`
	ExpiresAt string `json:"expires_at"`
}

type JoinRoomData struct {
	ChatID string `json:"chat_id"`
}

type LeaveRoomData struct {
	ChatID string `json:"chat_id"`
}

type MarkReadData struct {
	ChatID    string `json:"chat_id"`
	MessageID string `json:"message_id"`
}

type ReadReceiptData struct {
	ChatID     string `json:"chat_id"`
	MessageID  string `json:"message_id"`
	ReaderID   string `json:"reader_id"`
	ReaderName string `json:"reader_name"`
}

type DeliveryReceiptData struct {
	ChatID      string `json:"chat_id"`
	MessageID   string `json:"message_id"`
	DeliveredTo string `json:"delivered_to"`
}

type PresenceData struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	IsOnline     bool   `json:"is_online"`
	LastSeen     string `json:"last_seen"`
	LastActivity string `json:"last_activity"`
}

// HandleClientMessage processes incoming WebSocket messages
func (m *Manager) HandleClientMessage(client *Client, messageBytes []byte) {
	var wsMessage WSMessage

	if err := json.Unmarshal(messageBytes, &wsMessage); err != nil {
		log.Printf("WebSocket: Failed to unmarshal message from client %s: %v", client.UserID, err)
		m.sendErrorToClient(client, "Invalid message format")
		return
	}

	log.Printf("WebSocket: Received message type '%s' from client %s", wsMessage.Type, client.UserID)

	switch wsMessage.Type {
	case MessageTypePing:
		m.handlePing(client)

	case MessageTypeSendMessage:
		m.handleSendMessage(client, wsMessage.Data)

	case MessageTypeTyping:
		m.handleTyping(client, wsMessage.Data)

	case MessageTypeJoinRoom:
		m.handleJoinRoom(client, wsMessage.Data)

	case MessageTypeJoinChatRoom:
		// Frontend format: { type: "join_chat_room", chat_id: "id" }
		m.handleJoinChatRoom(client, wsMessage)

	case MessageTypeLeaveRoom:
		m.handleLeaveRoom(client, wsMessage.Data)

	case MessageTypeLeaveChatRoom:
		// Frontend format: { type: "leave_chat_room", chat_id: "id" }
		m.handleLeaveChatRoom(client, wsMessage)

	case MessageTypeMarkRead:
		m.handleMarkRead(client, wsMessage.Data)

	case MessageTypeMarkMessageRead:
		// Frontend format: { type: "mark_message_read", chat_id: "id", data: { message_id: "id" } }
		m.handleMarkMessageRead(client, wsMessage)

	case MessageTypeDeliveryReceipt:
		m.handleDeliveryReceipt(client, wsMessage.Data)

	case "typing_start":
		m.handleTypingStart(client, wsMessage)

	case "typing_stop":
		m.handleTypingStop(client, wsMessage)

	default:
		log.Printf("WebSocket: Unknown message type '%s' from client %s", wsMessage.Type, client.UserID)
		m.sendErrorToClient(client, "Unknown message type")
	}
}

func (m *Manager) handlePing(client *Client) {
	pongMessage := WSMessage{
		Type:      MessageTypePong,
		Data:      map[string]string{"status": "alive"},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	m.sendToClient(client, pongMessage)
}

func (m *Manager) handleSendMessage(client *Client, data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		m.sendErrorToClient(client, "Invalid send message data")
		return
	}

	var sendMsgData SendMessageData
	if err := json.Unmarshal(dataBytes, &sendMsgData); err != nil {
		m.sendErrorToClient(client, "Invalid send message format")
		return
	}

	// Validate required fields
	if sendMsgData.ChatID == "" || sendMsgData.Content == "" {
		m.sendErrorToClient(client, "Missing required fields")
		return
	}

	// Get user info from repository
	user, err := m.userRepo.GetByID(context.Background(), client.UserID)
	if err != nil {
		log.Printf("WebSocket: Failed to get user %s: %v", client.UserID, err)
		m.sendErrorToClient(client, "Failed to get user information")
		return
	}

	// Create message for database storage
	// Note: This should ideally call a message service/usecase
	messageID := generateMessageID() // You'll need to implement this

	messageData := MessageData{
		ID:             messageID,
		TempID:         sendMsgData.TempID,
		ChatID:         sendMsgData.ChatID,
		SenderID:       client.UserID,
		SenderName:     user.Username,
		Content:        sendMsgData.Content,
		Type:           sendMsgData.Type,
		Status:         "sent",
		Timestamp:      time.Now().Format(time.RFC3339),
		AttachmentURLs: []string{},
		Metadata:       map[string]interface{}{},
	}

	// TODO: Save message to database via message service
	// err = m.messageService.CreateMessage(ctx, messageData)
	// if err != nil {
	//     m.sendErrorToClient(client, "Failed to save message")
	//     return
	// }

	// Check if any recipients are online in this chat room
	m.mutex.RLock()
	chatClients, chatExists := m.chatRoomClients[sendMsgData.ChatID]
	hasOnlineRecipients := false
	if chatExists {
		for userID := range chatClients {
			if userID != client.UserID {
				hasOnlineRecipients = true
				break
			}
		}
	}
	m.mutex.RUnlock()

	// Update status based on recipient presence
	if hasOnlineRecipients {
		messageData.Status = "delivered" // At least one recipient is online
	} else {
		messageData.Status = "sent" // No recipients online, just sent
	}

	// Broadcast message to all clients in chat room
	message := WSMessage{
		Type:      MessageTypeMessage,
		Data:      messageData,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	m.BroadcastToChatRoom(sendMsgData.ChatID, message)

	// Send delivery receipt to sender if message was delivered
	if hasOnlineRecipients {
		deliveryReceipt := WSMessage{
			Type: MessageTypeDeliveryReceipt,
			Data: DeliveryReceiptData{
				ChatID:      sendMsgData.ChatID,
				MessageID:   messageID,
				DeliveredTo: "recipient_online",
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		m.sendToClient(client, deliveryReceipt)
	}

	log.Printf("WebSocket: Message %s sent from %s to chat %s (status: %s, recipients online: %v)",
		messageID, client.UserID, sendMsgData.ChatID, messageData.Status, hasOnlineRecipients)
}

func (m *Manager) handleTyping(client *Client, data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		m.sendErrorToClient(client, "Invalid typing data")
		return
	}

	var typingData TypingData
	if err := json.Unmarshal(dataBytes, &typingData); err != nil {
		m.sendErrorToClient(client, "Invalid typing format")
		return
	}

	// Get user info
	user, err := m.userRepo.GetByID(context.Background(), client.UserID)
	if err != nil {
		log.Printf("WebSocket: Failed to get user %s for typing: %v", client.UserID, err)
		return
	}

	typingData.UserID = client.UserID
	typingData.UserName = user.Username
	typingData.ExpiresAt = time.Now().Add(5 * time.Second).Format(time.RFC3339)

	message := WSMessage{
		Type:      MessageTypeTyping,
		Data:      typingData,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Broadcast typing to other clients in chat room (not to sender)
	m.BroadcastToChatRoomExcept(typingData.ChatID, client.UserID, message)
}

func (m *Manager) handleJoinRoom(client *Client, data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		m.sendErrorToClient(client, "Invalid join room data")
		return
	}

	var joinData JoinRoomData
	if err := json.Unmarshal(dataBytes, &joinData); err != nil {
		m.sendErrorToClient(client, "Invalid join room format")
		return
	}

	m.AddClientToChatRoom(joinData.ChatID, client.UserID)
	client.ActiveChatRoom = joinData.ChatID

	log.Printf("WebSocket: Client %s joined chat room %s", client.UserID, joinData.ChatID)
}

func (m *Manager) handleLeaveRoom(client *Client, data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		m.sendErrorToClient(client, "Invalid leave room data")
		return
	}

	var leaveData LeaveRoomData
	if err := json.Unmarshal(dataBytes, &leaveData); err != nil {
		m.sendErrorToClient(client, "Invalid leave room format")
		return
	}

	m.RemoveClientFromChatRoom(leaveData.ChatID, client.UserID)
	if client.ActiveChatRoom == leaveData.ChatID {
		client.ActiveChatRoom = ""
	}

	log.Printf("WebSocket: Client %s left chat room %s", client.UserID, leaveData.ChatID)
}

func (m *Manager) handleMarkRead(client *Client, data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		m.sendErrorToClient(client, "Invalid mark read data")
		return
	}

	var readData MarkReadData
	if err := json.Unmarshal(dataBytes, &readData); err != nil {
		m.sendErrorToClient(client, "Invalid mark read format")
		return
	}

	// Get user info
	user, err := m.userRepo.GetByID(context.Background(), client.UserID)
	if err != nil {
		log.Printf("WebSocket: Failed to get user %s for read receipt: %v", client.UserID, err)
		return
	}

	// TODO: Update message status in database
	// err = m.messageService.MarkMessageAsRead(ctx, readData.MessageID, client.UserID)

	// Send read receipt to other clients in chat room
	receiptData := ReadReceiptData{
		ChatID:     readData.ChatID,
		MessageID:  readData.MessageID,
		ReaderID:   client.UserID,
		ReaderName: user.Username,
	}

	message := WSMessage{
		Type:      MessageTypeReadReceipt,
		Data:      receiptData,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	m.BroadcastToChatRoomExcept(readData.ChatID, client.UserID, message)

	log.Printf("WebSocket: Message %s marked as read by %s", readData.MessageID, client.UserID)
}

func (m *Manager) handleDeliveryReceipt(client *Client, data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		m.sendErrorToClient(client, "Invalid delivery receipt data")
		return
	}

	var deliveryData DeliveryReceiptData
	if err := json.Unmarshal(dataBytes, &deliveryData); err != nil {
		m.sendErrorToClient(client, "Invalid delivery receipt format")
		return
	}

	// Send delivery receipt back to original sender
	message := WSMessage{
		Type:      MessageTypeDeliveryReceipt,
		Data:      deliveryData,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Find the message sender and send delivery confirmation
	// This would require looking up the message in database to find sender
	// For now, broadcast to chat room
	m.BroadcastToChatRoom(deliveryData.ChatID, message)

	log.Printf("WebSocket: Delivery receipt sent for message %s", deliveryData.MessageID)
}

func (m *Manager) sendToClient(client *Client, message WSMessage) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal message for client %s: %v", client.UserID, err)
		return
	}

	select {
	case client.Send <- messageBytes:
		// Success
	default:
		log.Printf("WebSocket: Client %s send channel full, closing connection", client.UserID)
		close(client.Send)
		m.RemoveClient(client.UserID)
	}
}

func (m *Manager) sendErrorToClient(client *Client, errorMsg string) {
	errorMessage := WSMessage{
		Type: "error",
		Data: map[string]string{
			"error":   errorMsg,
			"user_id": client.UserID,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	m.sendToClient(client, errorMessage)
}

func generateMessageID() string {
	// Simple ID generation - you might want to use UUID or database auto-increment
	return time.Now().Format("20060102150405") + "_" + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

// Frontend format handlers

func (m *Manager) handleJoinChatRoom(client *Client, wsMessage WSMessage) {
	if wsMessage.ChatID == "" {
		m.sendErrorToClient(client, "Missing chat_id")
		return
	}

	m.AddClientToChatRoom(wsMessage.ChatID, client.UserID)
	client.ActiveChatRoom = wsMessage.ChatID

	log.Printf("WebSocket: Client %s joined chat room %s (frontend format)", client.UserID, wsMessage.ChatID)

	// Broadcast user presence to other participants in the chat room
	user, err := m.userRepo.GetByID(context.Background(), client.UserID)
	if err != nil {
		log.Printf("WebSocket: Failed to get user %s for presence broadcast: %v", client.UserID, err)
		return
	}

	presenceMessage := WSMessage{
		Type: MessageTypeUserPresence,
		Data: map[string]interface{}{
			"user_id":   client.UserID,
			"username":  user.Username,
			"is_online": true,
			"last_seen": time.Now().Format(time.RFC3339),
			"chat_id":   wsMessage.ChatID,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	m.BroadcastToChatRoomExcept(wsMessage.ChatID, client.UserID, presenceMessage)
	log.Printf("WebSocket: Broadcasted online presence for %s to chat room %s", client.UserID, wsMessage.ChatID)
}

func (m *Manager) handleLeaveChatRoom(client *Client, wsMessage WSMessage) {
	if wsMessage.ChatID == "" {
		m.sendErrorToClient(client, "Missing chat_id")
		return
	}

	m.RemoveClientFromChatRoom(wsMessage.ChatID, client.UserID)
	if client.ActiveChatRoom == wsMessage.ChatID {
		client.ActiveChatRoom = ""
	}

	log.Printf("WebSocket: Client %s left chat room %s (frontend format)", client.UserID, wsMessage.ChatID)

	// Broadcast user offline presence to other participants in the chat room
	presenceMessage := WSMessage{
		Type: MessageTypeUserPresence,
		Data: map[string]interface{}{
			"user_id":   client.UserID,
			"is_online": false,
			"last_seen": time.Now().Format(time.RFC3339),
			"chat_id":   wsMessage.ChatID,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	m.BroadcastToChatRoomExcept(wsMessage.ChatID, client.UserID, presenceMessage)
}

func (m *Manager) handleMarkMessageRead(client *Client, wsMessage WSMessage) {
	if wsMessage.ChatID == "" {
		m.sendErrorToClient(client, "Missing chat_id")
		return
	}

	// Extract message_id from data
	data, ok := wsMessage.Data.(map[string]interface{})
	if !ok {
		m.sendErrorToClient(client, "Invalid data format")
		return
	}

	messageID, ok := data["message_id"].(string)
	if !ok {
		m.sendErrorToClient(client, "Missing message_id in data")
		return
	}

	// Get user info
	user, err := m.userRepo.GetByID(context.Background(), client.UserID)
	if err != nil {
		log.Printf("WebSocket: Failed to get user %s for read receipt: %v", client.UserID, err)
		return
	}

	// TODO: Update message status in database
	// err = m.messageService.MarkMessageAsRead(ctx, messageID, client.UserID)

	// Send read receipt to other clients in chat room
	receiptData := ReadReceiptData{
		ChatID:     wsMessage.ChatID,
		MessageID:  messageID,
		ReaderID:   client.UserID,
		ReaderName: user.Username,
	}

	message := WSMessage{
		Type:      MessageTypeReadReceipt,
		Data:      receiptData,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	m.BroadcastToChatRoomExcept(wsMessage.ChatID, client.UserID, message)

	log.Printf("WebSocket: Message %s marked as read by %s (frontend format)", messageID, client.UserID)
}

// handleTypingStart handles typing_start events from frontend
func (m *Manager) handleTypingStart(client *Client, wsMessage WSMessage) {
	chatID := wsMessage.ChatID
	if chatID == "" {
		m.sendErrorToClient(client, "Missing chat_id for typing_start")
		return
	}

	// Skip DB lookup for performance - frontend doesn't need username for typing indicator
	typingData := TypingData{
		ChatID:    chatID,
		UserID:    client.UserID,
		UserName:  "", // Frontend will look up name if needed
		Typing:    true,
		ExpiresAt: time.Now().Add(5 * time.Second).Format(time.RFC3339),
	}

	message := WSMessage{
		Type:      "typing_indicator",
		Data:      typingData,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	m.BroadcastToChatRoomExcept(chatID, client.UserID, message)
	log.Printf("WebSocket: Typing start from %s in chat %s", client.UserID, chatID)
}

// handleTypingStop handles typing_stop events from frontend
func (m *Manager) handleTypingStop(client *Client, wsMessage WSMessage) {
	chatID := wsMessage.ChatID
	if chatID == "" {
		m.sendErrorToClient(client, "Missing chat_id for typing_stop")
		return
	}

	// Skip DB lookup for performance - frontend doesn't need username for typing indicator
	typingData := TypingData{
		ChatID:    chatID,
		UserID:    client.UserID,
		UserName:  "", // Frontend will look up name if needed
		Typing:    false,
		ExpiresAt: time.Now().Format(time.RFC3339),
	}

	message := WSMessage{
		Type:      "typing_indicator",
		Data:      typingData,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	m.BroadcastToChatRoomExcept(chatID, client.UserID, message)
	log.Printf("WebSocket: Typing stop from %s in chat %s", client.UserID, chatID)
}
