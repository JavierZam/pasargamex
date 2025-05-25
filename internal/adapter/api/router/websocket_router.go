package router

import (
	"github.com/labstack/echo/v4"

	"pasargamex/internal/adapter/api/handler"
)

// SetupWebSocketRouter sets up WebSocket routes
func SetupWebSocketRouter(e *echo.Echo, wsHandler *handler.WebSocketHandler) {
	// WebSocket endpoint for real-time communication
	// Remove auth middleware since we handle auth inside the handler
	e.GET("/ws", wsHandler.HandleWebSocket)
}
