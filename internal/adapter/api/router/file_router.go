package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

func SetupFileRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware) {
	fileHandler := handler.GetFileHandler()

	files := e.Group("/v1/files")
	files.Use(authMiddleware.Authenticate)

	files.POST("/upload", fileHandler.UploadFile)

	files.POST("/upload/product-image", fileHandler.UploadProductImage)
	files.POST("/upload/profile-photo", fileHandler.UploadProfilePhoto)
	files.POST("/upload/verification-document", fileHandler.UploadVerificationDocument)

	files.POST("/delete", fileHandler.DeleteFile)
	files.GET("/list", fileHandler.ListUserFiles)

	files.GET("/view/:id", fileHandler.ViewFile)

	products := e.Group("/v1/products")
	products.Use(authMiddleware.Authenticate)
	products.POST("/:id/images", fileHandler.UploadAndLinkProductImage)

	admin := e.Group("/v1/admin/files")
	admin.Use(authMiddleware.Authenticate)
	admin.Use(adminMiddleware.AdminOnly)
	admin.GET("/view/:id", fileHandler.AdminViewFile)
}
