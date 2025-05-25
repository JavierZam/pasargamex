package websocket

import (
	"context"
	"log"
	"sync"

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
}

// NewManager creates a new WebSocket connection manager
func NewManager() *Manager {
	return &Manager{
		clients:         make(map[string]*Client),
		Register:        make(chan *Client),
		Unregister:      make(chan *Client),
		broadcast:       make(chan []byte),
		chatRoomClients: make(map[string]map[string]*Client),
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
		log.Printf("WebSocket: No clients in chat room %s.", chatID)
		return
	}

	for userID, client := range roomClients {
		if userID == excludeUserID {
			continue // Don't send back to the sender
		}
		select {
		case client.Send <- message:
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

		// Process incoming message - this will be handled by WebSocketHandler
		log.Printf("Received message from %s: %s", c.UserID, string(message))
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
