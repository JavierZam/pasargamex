package repository

import (
	"context"
	"pasargamex/internal/domain/entity"
)

type WishlistRepository interface {
	// Add product to user's wishlist
	AddToWishlist(ctx context.Context, userID, productID string) (*entity.WishlistItem, error)
	
	// Remove product from user's wishlist
	RemoveFromWishlist(ctx context.Context, userID, productID string) error
	
	// Check if product is in user's wishlist
	IsInWishlist(ctx context.Context, userID, productID string) (bool, error)
	
	// Get user's wishlist with product details
	GetUserWishlist(ctx context.Context, userID string, limit, offset int) ([]entity.WishlistItemWithProduct, int64, error)
	
	// Get wishlist item by ID
	GetWishlistItem(ctx context.Context, userID, productID string) (*entity.WishlistItem, error)
	
	// Get wishlist count for user
	GetWishlistCount(ctx context.Context, userID string) (int64, error)
}