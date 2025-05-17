package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

func SetupFileRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware) {
	fileHandler := handler.GetFileHandler()

	// Protected routes - require authentication
	files := e.Group("/v1/files")
	files.Use(authMiddleware.Authenticate)

	// Generic file upload endpoint
	files.POST("/upload", fileHandler.UploadFile)

	// Specialized endpoints for specific file types
	files.POST("/upload/product-image", fileHandler.UploadProductImage)
	files.POST("/upload/profile-photo", fileHandler.UploadProfilePhoto)
	files.POST("/upload/verification-document", fileHandler.UploadVerificationDocument)

	// Entity-specific upload endpoints with auto-linking
	files.POST("/upload/product/:productId/image", fileHandler.UploadAndLinkProductImage)

	// File deletion
	files.POST("/delete", fileHandler.DeleteFile)
}
