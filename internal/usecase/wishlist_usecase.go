package usecase

import (
	"context"
	"log"
	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
)

type WishlistUseCase struct {
	wishlistRepo repository.WishlistRepository
	productRepo  repository.ProductRepository
}

func NewWishlistUseCase(
	wishlistRepo repository.WishlistRepository,
	productRepo repository.ProductRepository,
) *WishlistUseCase {
	return &WishlistUseCase{
		wishlistRepo: wishlistRepo,
		productRepo:  productRepo,
	}
}

type WishlistResponse struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	ProductID string                 `json:"product_id"`
	Product   *WishlistProductInfo   `json:"product"`
	CreatedAt string                 `json:"created_at"`
}

type WishlistProductInfo struct {
	ID             string                 `json:"id"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Price          float64                `json:"price"`
	Type           string                 `json:"type"`
	Images         []entity.ProductImage  `json:"images"`
	Status         string                 `json:"status"`
	SellerID       string                 `json:"seller_id"`
	GameTitleID    string                 `json:"game_title_id"`
	DeliveryMethod string                 `json:"delivery_method"`
	Views          int                    `json:"views"`
	SoldCount      int                    `json:"sold_count"`
}

func (u *WishlistUseCase) AddToWishlist(ctx context.Context, userID, productID string) (*WishlistResponse, error) {
	log.Printf("Adding product %s to wishlist for user %s", productID, userID)
	
	// Validate that user can't add their own products
	product, err := u.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, errors.NotFound("Product not found", err)
	}
	
	if product.SellerID == userID {
		return nil, errors.BadRequest("Cannot add your own product to wishlist", nil)
	}
	
	item, err := u.wishlistRepo.AddToWishlist(ctx, userID, productID)
	if err != nil {
		return nil, err
	}
	
	return &WishlistResponse{
		ID:        item.ID,
		UserID:    item.UserID,
		ProductID: item.ProductID,
		Product: &WishlistProductInfo{
			ID:             product.ID,
			Title:          product.Title,
			Description:    product.Description,
			Price:          product.Price,
			Type:           product.Type,
			Images:         product.Images,
			Status:         product.Status,
			SellerID:       product.SellerID,
			GameTitleID:    product.GameTitleID,
			DeliveryMethod: product.DeliveryMethod,
			Views:          product.Views,
			SoldCount:      product.SoldCount,
		},
		CreatedAt: item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (u *WishlistUseCase) RemoveFromWishlist(ctx context.Context, userID, productID string) error {
	log.Printf("Removing product %s from wishlist for user %s", productID, userID)
	
	return u.wishlistRepo.RemoveFromWishlist(ctx, userID, productID)
}

func (u *WishlistUseCase) GetUserWishlist(ctx context.Context, userID string, page, pageSize int) ([]WishlistResponse, int64, error) {
	log.Printf("Getting wishlist for user %s, page %d, pageSize %d", userID, page, pageSize)
	
	offset := (page - 1) * pageSize
	
	items, total, err := u.wishlistRepo.GetUserWishlist(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	
	var response []WishlistResponse
	for _, item := range items {
		response = append(response, WishlistResponse{
			ID:        item.ID,
			UserID:    item.UserID,
			ProductID: item.ProductID,
			Product: &WishlistProductInfo{
				ID:             item.Product.ID,
				Title:          item.Product.Title,
				Description:    item.Product.Description,
				Price:          item.Product.Price,
				Type:           item.Product.Type,
				Images:         item.Product.Images,
				Status:         item.Product.Status,
				SellerID:       item.Product.SellerID,
				GameTitleID:    item.Product.GameTitleID,
				DeliveryMethod: item.Product.DeliveryMethod,
				Views:          item.Product.Views,
				SoldCount:      item.Product.SoldCount,
			},
			CreatedAt: item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	
	return response, total, nil
}

func (u *WishlistUseCase) IsInWishlist(ctx context.Context, userID, productID string) (bool, error) {
	return u.wishlistRepo.IsInWishlist(ctx, userID, productID)
}

func (u *WishlistUseCase) GetWishlistCount(ctx context.Context, userID string) (int64, error) {
	return u.wishlistRepo.GetWishlistCount(ctx, userID)
}