package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func SetupProductRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware, authClient *auth.Client) {
    // Get handlers from DI
    productHandler := handler.GetProductHandler()
    
    // Public product routes
    products := e.Group("/v1/products")
    products.GET("", productHandler.ListProducts)
    e.GET("/v1/products/search", productHandler.SearchProducts)
    
    // Product detail route with optional authentication
    productDetailGroup := e.Group("/v1/products")
    productDetailGroup.Use(VerifyToken(authClient))
    productDetailGroup.GET("/:id", productHandler.GetProduct)

    // Protected product routes (require authentication)
    myProducts := e.Group("/v1/my-products")
    myProducts.Use(authMiddleware.Authenticate)
    myProducts.GET("", productHandler.ListMyProducts)
    myProducts.POST("", productHandler.CreateProduct)
    myProducts.PUT("/:id", productHandler.UpdateProduct)
    myProducts.DELETE("/:id", productHandler.DeleteProduct)
    myProducts.POST("/:id/bump", productHandler.BumpProduct)
    
    // Admin routes
    admin := e.Group("/v1/admin/products")
    admin.Use(authMiddleware.Authenticate)
    admin.Use(adminMiddleware.AdminOnly)
    admin.POST("/migrate-bumped-at", productHandler.MigrateProductsBumpedAt)
    
    // New endpoint for validating product credentials
    admin.POST("/validate-credentials", productHandler.ValidateCredentials)
}