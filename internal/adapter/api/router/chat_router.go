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
	chatGroup.POST("/product", chatHandler.CreateChatByProduct) // POST /v1/chats/product - Create chat with seller via product ID
	chatGroup.GET("", chatHandler.GetUserChats)            // GET /v1/chats - Get user's chats
	chatGroup.GET("/individual", chatHandler.GetIndividualChats) // GET /v1/chats/individual - List individual chats only
	chatGroup.GET("/group", chatHandler.GetGroupChats)        // GET /v1/chats/group - List group chats only
	chatGroup.GET("/:id", chatHandler.GetChatByID)         // GET /v1/chats/:id - Get specific chat
	chatGroup.PUT("/:id/read", chatHandler.MarkChatAsRead) // PUT /v1/chats/:id/read - Mark chat as read

	// Message management
	chatGroup.POST("/:id/messages", chatHandler.SendMessage)    // POST /v1/chats/:id/messages - Send message
	chatGroup.GET("/:id/messages", chatHandler.GetChatMessages) // GET /v1/chats/:id/messages - Get chat messages

	// Offer system
	chatGroup.POST("/:id/messages/accept-offer", chatHandler.AcceptOffer) // POST /v1/chats/:id/messages/accept-offer
	chatGroup.POST("/:id/messages/reject-offer", chatHandler.RejectOffer) // POST /v1/chats/:id/messages/reject-offer

	// Group chat system
	chatGroup.POST("/group", chatHandler.CreateGroupChat)           // POST /v1/chats/group - Buyer creates group chat with seller + middleman
	chatGroup.POST("/group/seller", chatHandler.CreateSellerGroupChat) // POST /v1/chats/group/seller - Seller creates group chat with buyer + middleman
	chatGroup.GET("/middlemen", chatHandler.ListAvailableMiddlemen) // GET /v1/chats/middlemen - List available middlemen

	// Role detection system
	chatGroup.GET("/:id/role", chatHandler.GetUserRoleInChat)           // GET /v1/chats/:id/role - Get current user's role in chat
	chatGroup.GET("/:id/participants", chatHandler.GetChatParticipantsWithRoles) // GET /v1/chats/:id/participants - Get all participants with roles

	// Universal transaction chat system
	chatGroup.POST("/transaction", chatHandler.CreateTransactionChat)   // POST /v1/chats/transaction - Create transaction chat (universal)
	chatGroup.GET("/users", chatHandler.ListUsers)                     // GET /v1/chats/users - Get users for buyer/seller selection
	chatGroup.GET("/users/:userId/products", chatHandler.GetUserProducts) // GET /v1/chats/users/:userId/products - Get user's products

	// REMOVED: Middleman Chat creation endpoint (now triggered by TransactionUseCase)
	// adminChatGroup := e.Group("/v1/admin/chats")
	// adminChatGroup.Use(authMiddleware.Authenticate)
	// adminChatGroup.Use(adminMiddleware.AdminOnly)
	// adminChatGroup.POST("/middleman", chatHandler.CreateMiddlemanChat)
}
