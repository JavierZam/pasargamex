package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
)

type firestoreWishlistRepository struct {
	client *firestore.Client
}

func NewFirestoreWishlistRepository(client *firestore.Client) repository.WishlistRepository {
	return &firestoreWishlistRepository{client: client}
}

func (r *firestoreWishlistRepository) AddToWishlist(ctx context.Context, userID, productID string) (*entity.WishlistItem, error) {
	// Check if already exists
	exists, err := r.IsInWishlist(ctx, userID, productID)
	if err != nil {
		return nil, err
	}
	
	if exists {
		return nil, errors.BadRequest("Product already in wishlist", nil)
	}
	
	// Check if product exists and is active
	productRef := r.client.Collection("products").Doc(productID)
	productSnap, err := productRef.Get(ctx)
	if err != nil {
		return nil, errors.NotFound("Product not found", nil)
	}
	
	var product entity.Product
	if err := productSnap.DataTo(&product); err != nil {
		return nil, errors.Internal("Failed to parse product data", err)
	}
	
	if product.Status != "active" {
		return nil, errors.BadRequest("Cannot add inactive product to wishlist", nil)
	}
	
	// Create wishlist item
	wishlistID := fmt.Sprintf("%s_%s", userID, productID)
	wishlistItem := entity.WishlistItem{
		ID:        wishlistID,
		UserID:    userID,
		ProductID: productID,
		CreatedAt: time.Now(),
	}
	
	// Save to Firestore
	_, err = r.client.Collection("wishlists").Doc(wishlistID).Set(ctx, wishlistItem)
	if err != nil {
		return nil, errors.Internal("Failed to add to wishlist", err)
	}
	
	log.Printf("Added product %s to wishlist for user %s", productID, userID)
	return &wishlistItem, nil
}

func (r *firestoreWishlistRepository) RemoveFromWishlist(ctx context.Context, userID, productID string) error {
	wishlistID := fmt.Sprintf("%s_%s", userID, productID)
	
	// Check if exists
	exists, err := r.IsInWishlist(ctx, userID, productID)
	if err != nil {
		return err
	}
	
	if !exists {
		return errors.NotFound("Item not found in wishlist", nil)
	}
	
	// Delete from Firestore
	_, err = r.client.Collection("wishlists").Doc(wishlistID).Delete(ctx)
	if err != nil {
		return errors.Internal("Failed to remove from wishlist", err)
	}
	
	log.Printf("Removed product %s from wishlist for user %s", productID, userID)
	return nil
}

func (r *firestoreWishlistRepository) IsInWishlist(ctx context.Context, userID, productID string) (bool, error) {
	wishlistID := fmt.Sprintf("%s_%s", userID, productID)
	
	doc, err := r.client.Collection("wishlists").Doc(wishlistID).Get(ctx)
	if err != nil {
		if IsNotFound(err) {
			return false, nil
		}
		return false, errors.Internal("Failed to check wishlist", err)
	}
	
	return doc.Exists(), nil
}

func (r *firestoreWishlistRepository) GetUserWishlist(ctx context.Context, userID string, limit, offset int) ([]entity.WishlistItemWithProduct, int64, error) {
	query := r.client.Collection("wishlists").
		Where("userId", "==", userID).
		OrderBy("createdAt", firestore.Desc)
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, 0, errors.Internal("Failed to get wishlist", err)
	}
	
	var wishlistItems []entity.WishlistItemWithProduct
	
	for _, doc := range docs {
		var item entity.WishlistItem
		if err := doc.DataTo(&item); err != nil {
			log.Printf("Error parsing wishlist item %s: %v", doc.Ref.ID, err)
			continue
		}
		
		// Get product details
		productDoc, err := r.client.Collection("products").Doc(item.ProductID).Get(ctx)
		if err != nil {
			log.Printf("Error getting product %s: %v", item.ProductID, err)
			// Skip this item if product not found or error
			continue
		}
		
		var product entity.Product
		if err := productDoc.DataTo(&product); err != nil {
			log.Printf("Error parsing product %s: %v", item.ProductID, err)
			continue
		}
		
		// Only include active products
		if product.Status == "active" {
			wishlistItems = append(wishlistItems, entity.WishlistItemWithProduct{
				ID:        item.ID,
				UserID:    item.UserID,
				ProductID: item.ProductID,
				Product:   &product,
				CreatedAt: item.CreatedAt,
			})
		}
	}
	
	// Get total count
	totalQuery := r.client.Collection("wishlists").Where("userId", "==", userID)
	totalDocs, err := totalQuery.Documents(ctx).GetAll()
	if err != nil {
		return wishlistItems, int64(len(wishlistItems)), nil // Return what we have if count fails
	}
	
	// Count only active products in wishlist
	var activeCount int64
	for _, doc := range totalDocs {
		var item entity.WishlistItem
		if err := doc.DataTo(&item); err != nil {
			continue
		}
		
		// Check if product is still active
		productDoc, err := r.client.Collection("products").Doc(item.ProductID).Get(ctx)
		if err != nil {
			continue
		}
		
		var product entity.Product
		if err := productDoc.DataTo(&product); err != nil {
			continue
		}
		
		if product.Status == "active" {
			activeCount++
		}
	}
	
	return wishlistItems, activeCount, nil
}

func (r *firestoreWishlistRepository) GetWishlistItem(ctx context.Context, userID, productID string) (*entity.WishlistItem, error) {
	wishlistID := fmt.Sprintf("%s_%s", userID, productID)
	
	doc, err := r.client.Collection("wishlists").Doc(wishlistID).Get(ctx)
	if err != nil {
		if IsNotFound(err) {
			return nil, errors.NotFound("Wishlist item not found", nil)
		}
		return nil, errors.Internal("Failed to get wishlist item", err)
	}
	
	var item entity.WishlistItem
	if err := doc.DataTo(&item); err != nil {
		return nil, errors.Internal("Failed to parse wishlist item", err)
	}
	
	return &item, nil
}

func (r *firestoreWishlistRepository) GetWishlistCount(ctx context.Context, userID string) (int64, error) {
	query := r.client.Collection("wishlists").Where("userId", "==", userID)
	
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return 0, errors.Internal("Failed to get wishlist count", err)
	}
	
	// Count only active products
	var activeCount int64
	for _, doc := range docs {
		var item entity.WishlistItem
		if err := doc.DataTo(&item); err != nil {
			continue
		}
		
		// Check if product is still active
		productDoc, err := r.client.Collection("products").Doc(item.ProductID).Get(ctx)
		if err != nil {
			continue
		}
		
		var product entity.Product
		if err := productDoc.DataTo(&product); err != nil {
			continue
		}
		
		if product.Status == "active" {
			activeCount++
		}
	}
	
	return activeCount, nil
}

func IsNotFound(err error) bool {
	// Check if this is a Firestore "not found" error
	return err != nil && err.Error() == "rpc error: code = NotFound desc = no such entity"
}