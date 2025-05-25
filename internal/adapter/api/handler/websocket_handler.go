package handler

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	gorillaws "github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	ws "pasargamex/internal/infrastructure/websocket"
	"pasargamex/internal/usecase" // Import usecase
	"pasargamex/pkg/errors"

	"firebase.google.com/go/v4/auth"
)

type WebSocketHandler struct {
	wsManager   *ws.Manager
	authClient  *auth.Client
	chatUseCase *usecase.ChatUseCase // New: Inject ChatUseCase
}

var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // You should restrict this in production
	},
}

// Update constructor to include ChatUseCase
func NewWebSocketHandlerWithAuth(wsManager *ws.Manager, authClient *auth.Client, chatUseCase *usecase.ChatUseCase) *WebSocketHandler {
	return &WebSocketHandler{
		wsManager:   wsManager,
		authClient:  authClient,
		chatUseCase: chatUseCase,
	}
}

type WSMessage struct {
	Type    string                 `json:"type"`
	Token   string                 `json:"token,omitempty"`
	ChatID  string                 `json:"chat_id,omitempty"`
	Content string                 `json:"content,omitempty"`
	UserID  string                 `json:"user_id,omitempty"` // For typing/presence, sender's ID
	Data    map[string]interface{} `json:"data,omitempty"`    // Generic data for various events
}

func (h *WebSocketHandler) HandleWebSocket(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("WebSocket: Failed to upgrade connection: %v", err)
		return errors.Internal("Failed to upgrade connection", err)
	}
	// defer conn.Close() // Removed: defer in goroutines instead

	var userID string
	var authenticated bool

	if token != "" && h.authClient != nil {
		if uid, err := h.verifyToken(token); err == nil {
			userID = uid
			authenticated = true
			log.Printf("WebSocket: User %s authenticated via initial token.", userID)
		} else {
			log.Printf("WebSocket: Initial token verification failed for potential user: %v", err)
		}
	}

	// Send initial response
	if authenticated {
		response := WSMessage{
			Type: "auth_success",
			Data: map[string]interface{}{
				"message": "Authenticated successfully",
				"user_id": userID,
			},
		}
		conn.WriteJSON(response)
	} else {
		response := WSMessage{
			Type: "auth_required",
			Data: map[string]interface{}{
				"message": "Please send auth message with token",
			},
		}
		conn.WriteJSON(response)
	}

	client := &ws.Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256), // Buffered channel
	}

	if authenticated {
		h.wsManager.Register <- client
	}

	// ReadPump: Handles incoming messages from client
	go func() {
		defer func() {
			if authenticated { // Only unregister if client was successfully authenticated
				h.wsManager.Unregister <- client
			}
			conn.Close()
		}()

		for {
			var message WSMessage
			err := conn.ReadJSON(&message)
			if err != nil {
				if gorillaws.IsUnexpectedCloseError(err, gorillaws.CloseGoingAway, gorillaws.CloseAbnormalClosure) {
					log.Printf("WebSocket Read Error for %s: %v", client.UserID, err)
				}
				break
			}

			// Handle authentication message if not yet authenticated
			if message.Type == "auth" && !authenticated && h.authClient != nil {
				if uid, err := h.verifyToken(message.Token); err == nil {
					userID = uid
					client.UserID = userID
					authenticated = true

					h.wsManager.Register <- client // Register client after successful authentication

					response := WSMessage{
						Type: "auth_success",
						Data: map[string]interface{}{
							"message": "Authenticated successfully",
							"user_id": userID,
						},
					}
					conn.WriteJSON(response)
					log.Printf("WebSocket: User %s authenticated via message.", userID)
				} else {
					response := WSMessage{
						Type: "auth_error",
						Data: map[string]interface{}{
							"message": "Invalid token",
						},
					}
					conn.WriteJSON(response)
					log.Printf("WebSocket: Authentication failed via message for token: %v", err)
				}
				continue // Skip processing other message types until authenticated
			}

			// Process other messages only if authenticated
			if authenticated {
				// log.Printf("Received message from %s: %+v", userID, message) // Cleaned up debug log

				switch message.Type {
				case "test_message":
					// Echo back the message as confirmation
					response := WSMessage{
						Type: "message_received",
						Data: map[string]interface{}{
							"original_message": message,
							"timestamp":        time.Now().Format(time.RFC3339),
						},
					}
					conn.WriteJSON(response)
				case "typing_start":
					if message.ChatID != "" {
						h.chatUseCase.HandleTypingEvent(context.Background(), client.UserID, message.ChatID, true)
					}
				case "typing_stop":
					if message.ChatID != "" {
						h.chatUseCase.HandleTypingEvent(context.Background(), client.UserID, message.ChatID, false)
					}
				case "join_chat_room":
					if message.ChatID != "" {
						h.wsManager.SetClientActiveChatRoom(client.UserID, message.ChatID)
						// Optionally, notify others in the room about presence
						h.chatUseCase.HandleUserPresence(context.Background(), client.UserID, message.ChatID, true)
					}
				case "leave_chat_room":
					if message.ChatID != "" {
						h.wsManager.SetClientActiveChatRoom(client.UserID, "") // Clear active room
						// Optionally, notify others about absence
						h.chatUseCase.HandleUserPresence(context.Background(), client.UserID, message.ChatID, false)
					}
				case "mark_message_read":
					if message.ChatID != "" && message.Data != nil {
						if messageID, ok := message.Data["message_id"].(string); ok {
							h.chatUseCase.MarkMessageAsRead(context.Background(), message.ChatID, messageID, client.UserID)
						}
					}
				default:
					log.Printf("WebSocket: Unknown message type from %s: %s", client.UserID, message.Type)
					response := WSMessage{
						Type: "error",
						Data: map[string]interface{}{
							"message":       "Unknown message type",
							"type_received": message.Type,
						},
					}
					conn.WriteJSON(response)
				}
			} else {
				response := WSMessage{
					Type: "error",
					Data: map[string]interface{}{
						"message": "Authentication required",
					},
				}
				conn.WriteJSON(response)
			}
		}
	}()

	// WritePump: Handles outgoing messages to client
	go client.WritePump() // Corrected: Removed argument

	// Keep connection alive (this select {} is blocking, ensuring goroutines run)
	select {}
}

func (h *WebSocketHandler) verifyToken(token string) (string, error) {
	if h.authClient == nil {
		return "", errors.Internal("Auth client not available", nil)
	}

	firebaseToken, err := h.authClient.VerifyIDToken(context.Background(), token)
	if err != nil {
		return "", err
	}

	return firebaseToken.UID, nil
}
