package router

import (
	"github.com/labstack/echo/v4"

	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"
)

// SetupChatRouter sets up all chat-related routes (excluding WebSocket)
func SetupChatRouter(e *echo.Echo, chatHandler *handler.ChatHandler, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware) {
	// Chat API endpoints
	chatGroup := e.Group("/v1/chats")
	chatGroup.Use(authMiddleware.Authenticate) // All chat endpoints require authentication

	// Chat management
	chatGroup.POST("", chatHandler.CreateChat)             // POST /v1/chats - Create new direct chat
	chatGroup.GET("", chatHandler.GetUserChats)            // GET /v1/chats - Get user's chats
	chatGroup.GET("/:id", chatHandler.GetChatByID)         // GET /v1/chats/:id - Get specific chat
	chatGroup.PUT("/:id/read", chatHandler.MarkChatAsRead) // PUT /v1/chats/:id/read - Mark chat as read

	// Message management
	chatGroup.POST("/:id/messages", chatHandler.SendMessage)    // POST /v1/chats/:id/messages - Send message
	chatGroup.GET("/:id/messages", chatHandler.GetChatMessages) // GET /v1/chats/:id/messages - Get chat messages

	// Offer system
	chatGroup.POST("/:id/messages/accept-offer", chatHandler.AcceptOffer) // POST /v1/chats/:id/messages/accept-offer
	chatGroup.POST("/:id/messages/reject-offer", chatHandler.RejectOffer) // POST /v1/chats/:id/messages/reject-offer

	// REMOVED: Middleman Chat creation endpoint (now triggered by TransactionUseCase)
	// adminChatGroup := e.Group("/v1/admin/chats")
	// adminChatGroup.Use(authMiddleware.Authenticate)
	// adminChatGroup.Use(adminMiddleware.AdminOnly)
	// adminChatGroup.POST("/middleman", chatHandler.CreateMiddlemanChat)
}
