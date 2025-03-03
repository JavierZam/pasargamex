package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

// SetupProductRouter initializes product routes
func SetupProductRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware) {
	// Get handlers from DI
	productHandler := handler.GetProductHandler()
	admin := e.Group("/v1/admin/products")

	// Public product routes
	products := e.Group("/v1/products")
	products.GET("", productHandler.ListProducts)
	products.GET("/:id", productHandler.GetProduct)
	products.GET("/search", productHandler.SearchProducts)
	admin.POST("/migrate-bumped-at", productHandler.MigrateProductsBumpedAt)
	admin.Use(authMiddleware.Authenticate)
	admin.Use(adminMiddleware.AdminOnly)

	// Protected product routes (require authentication)
	myProducts := e.Group("/v1/my-products")
	myProducts.Use(authMiddleware.Authenticate)
	myProducts.GET("", productHandler.ListMyProducts)
	myProducts.POST("", productHandler.CreateProduct)
	myProducts.PUT("/:id", productHandler.UpdateProduct)
	myProducts.DELETE("/:id", productHandler.DeleteProduct)
	myProducts.POST("/:id/bump", productHandler.BumpProduct)
}