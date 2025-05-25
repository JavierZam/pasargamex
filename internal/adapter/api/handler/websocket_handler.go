package handler

import (
	"context"
	"log"
	"net/http"
	"strings"

	gorillaws "github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"pasargamex/internal/adapter/api/middleware"
	ws "pasargamex/internal/infrastructure/websocket"
	"pasargamex/pkg/errors"

	"firebase.google.com/go/v4/auth"
)

type WebSocketHandler struct {
	wsManager  *ws.Manager
	authClient *auth.Client
}

var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // You should restrict this in production
	},
}

func NewWebSocketHandler(wsManager *ws.Manager, authMiddleware *middleware.AuthMiddleware) *WebSocketHandler {
	return &WebSocketHandler{
		wsManager: wsManager,
	}
}

// Add constructor with auth client
func NewWebSocketHandlerWithAuth(wsManager *ws.Manager, authClient *auth.Client) *WebSocketHandler {
	return &WebSocketHandler{
		wsManager:  wsManager,
		authClient: authClient,
	}
}

type WSMessage struct {
	Type    string      `json:"type"`
	Token   string      `json:"token,omitempty"`
	Content string      `json:"content,omitempty"`
	UserID  string      `json:"user_id,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func (h *WebSocketHandler) HandleWebSocket(c echo.Context) error {
	// Try to get token from query parameter first
	token := c.QueryParam("token")
	if token == "" {
		// Try to get from Authorization header
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return errors.Internal("Failed to upgrade connection", err)
	}
	defer conn.Close()

	var userID string
	var authenticated bool

	// If we have token, try to authenticate immediately
	if token != "" && h.authClient != nil {
		if uid, err := h.verifyToken(token); err == nil {
			userID = uid
			authenticated = true
			log.Printf("WebSocket user authenticated: %s", userID)
		}
	}

	// Send initial response
	if authenticated {
		response := WSMessage{
			Type: "auth_success",
			Data: map[string]string{
				"message": "Authenticated successfully",
				"user_id": userID,
			},
		}
		conn.WriteJSON(response)
	} else {
		response := WSMessage{
			Type: "auth_required",
			Data: map[string]string{
				"message": "Please send auth message with token",
			},
		}
		conn.WriteJSON(response)
	}

	// Create a new client
	client := &ws.Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	// Register client with manager if authenticated
	if authenticated {
		h.wsManager.Register <- client
	}

	// Handle incoming messages
	go func() {
		defer func() {
			if authenticated {
				h.wsManager.Unregister <- client
			}
			conn.Close()
		}()

		for {
			var message WSMessage
			err := conn.ReadJSON(&message)
			if err != nil {
				if gorillaws.IsUnexpectedCloseError(err, gorillaws.CloseGoingAway, gorillaws.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				break
			}

			// Handle authentication message
			if message.Type == "auth" && !authenticated && h.authClient != nil {
				if uid, err := h.verifyToken(message.Token); err == nil {
					userID = uid
					client.UserID = userID
					authenticated = true

					// Register client after authentication
					h.wsManager.Register <- client

					response := WSMessage{
						Type: "auth_success",
						Data: map[string]string{
							"message": "Authenticated successfully",
							"user_id": userID,
						},
					}
					conn.WriteJSON(response)
					log.Printf("WebSocket user authenticated via message: %s", userID)
				} else {
					response := WSMessage{
						Type: "auth_error",
						Data: map[string]string{
							"message": "Invalid token",
						},
					}
					conn.WriteJSON(response)
				}
				continue
			}

			// Handle other messages only if authenticated
			if authenticated {
				log.Printf("Received message from %s: %+v", userID, message)

				// Echo back the message as confirmation
				response := WSMessage{
					Type: "message_received",
					Data: map[string]interface{}{
						"original_message": message,
						"timestamp":        "received",
					},
				}
				conn.WriteJSON(response)
			} else {
				response := WSMessage{
					Type: "error",
					Data: map[string]string{
						"message": "Authentication required",
					},
				}
				conn.WriteJSON(response)
			}
		}
	}()

	// Handle outgoing messages
	go func() {
		defer conn.Close()
		for {
			select {
			case message, ok := <-client.Send:
				if !ok {
					conn.WriteMessage(gorillaws.CloseMessage, []byte{})
					return
				}

				err := conn.WriteMessage(gorillaws.TextMessage, message)
				if err != nil {
					log.Printf("WebSocket write error: %v", err)
					return
				}
			}
		}
	}()

	// Keep connection alive
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
