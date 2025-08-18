package router

import (
	"pasargamex/internal/adapter/api/handler"
	"pasargamex/internal/adapter/api/middleware"

	"github.com/labstack/echo/v4"
)

func SetupWishlistRouter(e *echo.Echo, wishlistHandler *handler.WishlistHandler, authMiddleware *middleware.AuthMiddleware) {
	// All wishlist endpoints require authentication
	wishlistGroup := e.Group("/v1/wishlist")
	wishlistGroup.Use(authMiddleware.Authenticate)

	// Wishlist management
	wishlistGroup.POST("/:productId", wishlistHandler.AddToWishlist)       // POST /v1/wishlist/:productId - Add to wishlist
	wishlistGroup.DELETE("/:productId", wishlistHandler.RemoveFromWishlist) // DELETE /v1/wishlist/:productId - Remove from wishlist
	wishlistGroup.GET("", wishlistHandler.GetUserWishlist)                 // GET /v1/wishlist - Get user's wishlist
	wishlistGroup.GET("/:productId/status", wishlistHandler.CheckWishlistStatus) // GET /v1/wishlist/:productId/status - Check if in wishlist
	wishlistGroup.GET("/count", wishlistHandler.GetWishlistCount)          // GET /v1/wishlist/count - Get wishlist count
}