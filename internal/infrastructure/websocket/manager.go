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
}

// Manager manages all active WebSocket connections
type Manager struct {
	clients    map[string]*Client
	Register   chan *Client // Make these public so they can be accessed
	Unregister chan *Client
	broadcast  chan []byte
	mutex      sync.RWMutex
}

// NewManager creates a new WebSocket connection manager
func NewManager() *Manager {
	return &Manager{
		clients:    make(map[string]*Client),
		Register:   make(chan *Client), // Capitalize for public access
		Unregister: make(chan *Client), // Capitalize for public access
		broadcast:  make(chan []byte),
	}
}

// Start runs the manager's main loop in a goroutine
func (m *Manager) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case client := <-m.Register: // Use the capitalized field names
				m.mutex.Lock()
				m.clients[client.UserID] = client
				m.mutex.Unlock()
				log.Printf("Client registered: %s", client.UserID)

			case client := <-m.Unregister: // Use the capitalized field names
				m.mutex.Lock()
				if _, ok := m.clients[client.UserID]; ok {
					delete(m.clients, client.UserID)
					close(client.Send)
				}
				m.mutex.Unlock()
				log.Printf("Client unregistered: %s", client.UserID)

			case message := <-m.broadcast:
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
		client.Send <- message
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump(m *Manager) {
	defer func() {
		m.Unregister <- c // Use the capitalized field name
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// Process incoming message
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
			log.Printf("error: %v", err)
			return
		}
	}
}
