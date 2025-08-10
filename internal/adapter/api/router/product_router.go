package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func SetupProductRouter(e *echo.Echo, authMiddleware *middleware.AuthMiddleware, adminMiddleware *middleware.AdminMiddleware, authClient *auth.Client) {

	productHandler := handler.GetProductHandler()

	products := e.Group("/v1/products")
	products.GET("", productHandler.ListProducts)
	e.GET("/v1/products/search", productHandler.SearchProducts)
	products.GET("/seller/:sellerId", productHandler.GetProductsBySeller)
	e.GET("/v1/sellers/:sellerId/profile", productHandler.GetSellerProfile)

	productDetailGroup := e.Group("/v1/products")
	productDetailGroup.Use(VerifyToken(authClient))
	productDetailGroup.GET("/:id", productHandler.GetProduct)
	
	// Product reviews - public endpoint
	reviewHandler := handler.GetReviewHandler()
	products.GET("/:id/reviews", reviewHandler.GetProductReviews)

	myProducts := e.Group("/v1/my-products")
	myProducts.Use(authMiddleware.Authenticate)
	myProducts.GET("", productHandler.ListMyProducts)
	myProducts.POST("", productHandler.CreateProduct)
	myProducts.PUT("/:id", productHandler.UpdateProduct)
	myProducts.DELETE("/:id", productHandler.DeleteProduct)
	myProducts.POST("/:id/bump", productHandler.BumpProduct)
	myProducts.DELETE("/:id/images/:imageId", productHandler.DeleteProductImage)

	admin := e.Group("/v1/admin/products")
	admin.Use(authMiddleware.Authenticate)
	admin.Use(adminMiddleware.AdminOnly)
	admin.POST("/migrate-bumped-at", productHandler.MigrateProductsBumpedAt)

	admin.POST("/validate-credentials", productHandler.ValidateCredentials)
}
