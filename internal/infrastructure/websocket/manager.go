package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"pasargamex/internal/domain/repository"

	"github.com/gorilla/websocket"
)

// Client represents a WebSocket connection client
type Client struct {
	UserID string
	Conn   *websocket.Conn
	Send   chan []byte
	// New: Track which chat rooms the client is currently viewing
	ActiveChatRoom string
}

// Manager manages all active WebSocket connections
type Manager struct {
	clients    map[string]*Client
	Register   chan *Client
	Unregister chan *Client
	broadcast  chan []byte
	// New: Map to track clients by chat room ID
	chatRoomClients map[string]map[string]*Client // chatID -> userID -> *Client
	mutex           sync.RWMutex
	userRepo        repository.UserRepository
}

// NewManager creates a new WebSocket connection manager
func NewManager(userRepo repository.UserRepository) *Manager {
	return &Manager{
		clients:         make(map[string]*Client),
		Register:        make(chan *Client),
		Unregister:      make(chan *Client),
		broadcast:       make(chan []byte),
		chatRoomClients: make(map[string]map[string]*Client),
		userRepo:        userRepo,
	}
}

// Start runs the manager's main loop in a goroutine
func (m *Manager) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case client := <-m.Register:
				m.mutex.Lock()
				m.clients[client.UserID] = client
				m.mutex.Unlock()
				log.Printf("WebSocket: Client registered: %s", client.UserID)

				// Update user status to online
				go m.updateUserPresence(ctx, client.UserID, "online")

			case client := <-m.Unregister:
				m.mutex.Lock()
				if _, ok := m.clients[client.UserID]; ok {
					delete(m.clients, client.UserID)
					close(client.Send)
					// Also remove from any active chat rooms
					if client.ActiveChatRoom != "" {
						if roomClients, ok := m.chatRoomClients[client.ActiveChatRoom]; ok {
							delete(roomClients, client.UserID)
							if len(roomClients) == 0 {
								delete(m.chatRoomClients, client.ActiveChatRoom)
							}
						}
					}
				}
				m.mutex.Unlock()
				log.Printf("WebSocket: Client unregistered: %s", client.UserID)

				// Update user status to offline
				go m.updateUserPresence(ctx, client.UserID, "offline")

			case message := <-m.broadcast:
				// This is for general broadcast, not specific to chat rooms
				for _, client := range m.clients {
					select {
					case client.Send <- message:
					default:
						close(client.Send)
						m.mutex.Lock()
						delete(m.clients, client.UserID)
						m.mutex.Unlock()
					}
				}

			case <-ctx.Done():
				return
			}
		}
	}()
}

// SendToUser sends a message to a specific user
func (m *Manager) SendToUser(userID string, message []byte) {
	m.mutex.RLock()
	client, ok := m.clients[userID]
	m.mutex.RUnlock()

	if ok {
		select {
		case client.Send <- message:
		default:
			log.Printf("WebSocket: Failed to send message to user %s, send channel is full.", userID)
			// Optionally, unregister client if send channel is blocked
			// m.Unregister <- client
		}
	} else {
		log.Printf("WebSocket: User %s not found in active clients.", userID)
	}
}

// SendToChatRoom sends a message to all clients in a specific chat room
func (m *Manager) SendToChatRoom(chatID string, message []byte, excludeUserID string) {
	m.mutex.RLock()
	roomClients, ok := m.chatRoomClients[chatID]
	m.mutex.RUnlock()

	if !ok {
		log.Printf("WebSocket: SendToChatRoom - No clients in chat room %s (room not in map)", chatID)
		return
	}

	// Count clients excluding sender
	targetClients := 0
	for userID := range roomClients {
		if userID != excludeUserID {
			targetClients++
		}
	}

	if targetClients == 0 {
		log.Printf("WebSocket: SendToChatRoom - No other clients in chat room %s (only sender %s present)", chatID, excludeUserID)
		return
	}

	log.Printf("WebSocket: SendToChatRoom - Sending to %d client(s) in chat room %s (excluding %s)", targetClients, chatID, excludeUserID)

	for userID, client := range roomClients {
		if userID == excludeUserID {
			continue // Don't send back to the sender
		}
		select {
		case client.Send <- message:
			log.Printf("WebSocket: SendToChatRoom - Message sent successfully to %s", userID)
		default:
			log.Printf("WebSocket: Failed to send message to user %s in chat room %s, send channel is full.", userID, chatID)
			// Optionally, unregister client if send channel is blocked
			// m.Unregister <- client
		}
	}
}

// SetClientActiveChatRoom updates the active chat room for a client
func (m *Manager) SetClientActiveChatRoom(userID, chatID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	client, ok := m.clients[userID]
	if !ok {
		log.Printf("WebSocket: Cannot set active chat room for unknown client %s.", userID)
		return
	}

	// Remove from old chat room if any
	if client.ActiveChatRoom != "" && client.ActiveChatRoom != chatID {
		if oldRoomClients, ok := m.chatRoomClients[client.ActiveChatRoom]; ok {
			delete(oldRoomClients, userID)
			if len(oldRoomClients) == 0 {
				delete(m.chatRoomClients, client.ActiveChatRoom)
			}
			log.Printf("WebSocket: User %s left chat room %s.", userID, client.ActiveChatRoom)
		}
	}

	// Add to new chat room
	if chatID != "" {
		if _, ok := m.chatRoomClients[chatID]; !ok {
			m.chatRoomClients[chatID] = make(map[string]*Client)
		}
		m.chatRoomClients[chatID][userID] = client
		client.ActiveChatRoom = chatID
		log.Printf("WebSocket: User %s joined chat room %s.", userID, chatID)
	} else {
		client.ActiveChatRoom = "" // Clear active chat room
		log.Printf("WebSocket: User %s cleared active chat room.", userID)
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump(m *Manager) {
	defer func() {
		m.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket ReadPump Error for %s: %v", c.UserID, err)
			}
			break
		}

		// Process incoming message using message handler
		log.Printf("Received message from %s: %s", c.UserID, string(message))
		m.HandleClientMessage(c, message)
	}
}

// WritePump sends messages to the WebSocket connection
func (c *Client) WritePump() {
	defer c.Conn.Close()

	for {
		message, ok := <-c.Send
		if !ok {
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("WebSocket WritePump Error for %s: %v", c.UserID, err)
			return
		}
	}
}

// updateUserPresence updates user's last_seen and online status
func (m *Manager) updateUserPresence(ctx context.Context, userID string, status string) {
	if m.userRepo == nil {
		log.Printf("WebSocket: UserRepo is nil, cannot update presence for user %s", userID)
		return
	}

	user, err := m.userRepo.GetByID(ctx, userID)
	if err != nil {
		log.Printf("WebSocket: Failed to get user %s for presence update: %v", userID, err)
		return
	}

	now := time.Now()
	user.LastSeen = now
	user.OnlineStatus = status
	user.UpdatedAt = now

	if err := m.userRepo.Update(ctx, user); err != nil {
		log.Printf("WebSocket: Failed to update presence for user %s: %v", userID, err)
		return
	}

	log.Printf("WebSocket: Updated presence for user %s to %s at %v", userID, status, now)

	// Broadcast presence update to other users in active chats
	m.BroadcastPresenceUpdate(userID, status)
}

// BroadcastPresenceUpdate broadcasts user presence changes to relevant chat participants
func (m *Manager) BroadcastPresenceUpdate(userID, status string) {
	ctx := context.Background()

	// Get user details for the presence update
	user, err := m.userRepo.GetByID(ctx, userID)
	if err != nil {
		log.Printf("WebSocket: Failed to get user %s for presence broadcast: %v", userID, err)
		return
	}

	// Get all chat rooms where this user is active
	m.mutex.RLock()
	userChatRooms := make([]string, 0)
	for chatID, clients := range m.chatRoomClients {
		if _, exists := clients[userID]; exists {
			userChatRooms = append(userChatRooms, chatID)
		}
	}
	m.mutex.RUnlock()

	// Create presence update message following HTML test pattern
	presenceMessage := map[string]interface{}{
		"type":      "user_presence",
		"user_id":   userID,
		"username":  user.Username,
		"is_online": status == "online",
		"last_seen": user.LastSeen.Format(time.RFC3339),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Broadcast to all chat rooms where this user is active
	for _, chatID := range userChatRooms {
		// Add chat_id to the message for this specific broadcast
		chatPresenceMessage := make(map[string]interface{})
		for k, v := range presenceMessage {
			chatPresenceMessage[k] = v
		}
		chatPresenceMessage["chat_id"] = chatID

		chatMessageBytes, err := json.Marshal(chatPresenceMessage)
		if err != nil {
			continue
		}

		// Send to all other users in this chat room (exclude the user whose presence changed)
		m.SendToChatRoom(chatID, chatMessageBytes, userID)
	}

	log.Printf("WebSocket: Broadcasting presence update for user %s (%s) to %d chat rooms: %s",
		userID, user.Username, len(userChatRooms), status)

	// Also broadcast to ALL online clients for global presence (untuk ChatList)
	m.mutex.RLock()
	onlineClients := make(map[string]*Client)
	for clientID, client := range m.clients {
		if clientID != userID { // Don't send to self
			onlineClients[clientID] = client
		}
	}
	m.mutex.RUnlock()

	// Send to all online clients
	globalPresenceBytes, err := json.Marshal(presenceMessage)
	if err == nil && len(onlineClients) > 0 {
		log.Printf("WebSocket: Broadcasting presence to %d online clients", len(onlineClients))
		for clientID, client := range onlineClients {
			select {
			case client.Send <- globalPresenceBytes:
				// Success
			default:
				log.Printf("WebSocket: Failed to send presence to client %s (channel full)", clientID)
			}
		}
	}
}

// BroadcastToChatRoom sends message to all clients in a specific chat room
func (m *Manager) BroadcastToChatRoom(chatID string, message WSMessage) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal message for chat room %s: %v", chatID, err)
		return
	}

	m.mutex.RLock()
	chatClients, exists := m.chatRoomClients[chatID]
	if !exists {
		m.mutex.RUnlock()
		return
	}

	// Copy clients to avoid holding lock too long
	clients := make([]*Client, 0, len(chatClients))
	for _, client := range chatClients {
		clients = append(clients, client)
	}
	m.mutex.RUnlock()

	log.Printf("WebSocket: Broadcasting message to %d clients in chat room %s", len(clients), chatID)

	for _, client := range clients {
		select {
		case client.Send <- messageBytes:
			// Success
		default:
			log.Printf("WebSocket: Failed to send message to client %s (channel full)", client.UserID)
			close(client.Send)
			m.RemoveClient(client.UserID)
		}
	}
}

// BroadcastToChatRoomExcept sends message to all clients in chat room except specified user
func (m *Manager) BroadcastToChatRoomExcept(chatID string, exceptUserID string, message WSMessage) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal message for chat room %s: %v", chatID, err)
		return
	}

	m.mutex.RLock()
	chatClients, exists := m.chatRoomClients[chatID]
	if !exists {
		m.mutex.RUnlock()
		log.Printf("WebSocket: No clients registered in chat room %s (room not found in chatRoomClients map)", chatID)
		return
	}

	// Copy clients excluding specified user
	clients := make([]*Client, 0, len(chatClients))
	for userID, client := range chatClients {
		if userID != exceptUserID {
			clients = append(clients, client)
		}
	}
	m.mutex.RUnlock()

	if len(clients) == 0 {
		log.Printf("WebSocket: No other clients in chat room %s (all clients except %s)", chatID, exceptUserID)
		return
	}

	log.Printf("WebSocket: Broadcasting message type '%s' to %d clients in chat room %s (except %s)", message.Type, len(clients), chatID, exceptUserID)

	for _, client := range clients {
		select {
		case client.Send <- messageBytes:
			// Success
		default:
			log.Printf("WebSocket: Failed to send message to client %s (channel full)", client.UserID)
			close(client.Send)
			m.RemoveClient(client.UserID)
		}
	}
}

// AddClientToChatRoom adds a client to a chat room
func (m *Manager) AddClientToChatRoom(chatID string, userID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	client, exists := m.clients[userID]
	if !exists {
		log.Printf("WebSocket: Client %s not found when adding to chat room %s", userID, chatID)
		return
	}

	if m.chatRoomClients[chatID] == nil {
		m.chatRoomClients[chatID] = make(map[string]*Client)
	}

	m.chatRoomClients[chatID][userID] = client
	log.Printf("WebSocket: Added client %s to chat room %s", userID, chatID)
}

// RemoveClientFromChatRoom removes a client from a chat room
func (m *Manager) RemoveClientFromChatRoom(chatID string, userID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if chatClients, exists := m.chatRoomClients[chatID]; exists {
		delete(chatClients, userID)

		// Clean up empty chat room
		if len(chatClients) == 0 {
			delete(m.chatRoomClients, chatID)
		}
	}

	log.Printf("WebSocket: Removed client %s from chat room %s", userID, chatID)
}

// RemoveClient removes a client from all rooms and connections
func (m *Manager) RemoveClient(userID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Remove from all chat rooms
	for chatID, chatClients := range m.chatRoomClients {
		delete(chatClients, userID)

		// Clean up empty chat room
		if len(chatClients) == 0 {
			delete(m.chatRoomClients, chatID)
		}
	}

	// Remove from main clients map
	if client, exists := m.clients[userID]; exists {
		close(client.Send)
		delete(m.clients, userID)
	}

	log.Printf("WebSocket: Removed client %s from all rooms", userID)
}
