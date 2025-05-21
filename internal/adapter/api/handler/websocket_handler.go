package handler

import (
	"net/http"

	gorillaws "github.com/gorilla/websocket" // Rename the import to avoid conflict
	"github.com/labstack/echo/v4"

	"pasargamex/internal/adapter/api/middleware"
	ws "pasargamex/internal/infrastructure/websocket" // Use alias for our websocket package
	"pasargamex/pkg/errors"
)

type WebSocketHandler struct {
	wsManager      *ws.Manager
	authMiddleware *middleware.AuthMiddleware
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
		wsManager:      wsManager,
		authMiddleware: authMiddleware,
	}
}

func (h *WebSocketHandler) HandleWebSocket(c echo.Context) error {
	// Get user ID from context (this would be set by your auth middleware)
	userID, ok := c.Get("uid").(string)
	if !ok || userID == "" {
		return errors.Unauthorized("Authentication required", nil)
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return errors.Internal("Failed to upgrade connection", err)
	}

	// Create a new client
	client := &ws.Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	// Register client with manager
	h.wsManager.Register <- client

	// Start goroutines for reading and writing
	go client.ReadPump(h.wsManager)
	go client.WritePump()

	return nil
}
