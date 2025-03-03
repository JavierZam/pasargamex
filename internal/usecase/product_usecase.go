package usecase

import (
	"context"
	"time"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
)

type ProductUseCase struct {
	productRepo   repository.ProductRepository
	gameTitleRepo repository.GameTitleRepository
	userRepo      repository.UserRepository
}

func NewProductUseCase(
	productRepo repository.ProductRepository,
	gameTitleRepo repository.GameTitleRepository,
	userRepo repository.UserRepository,
) *ProductUseCase {
	return &ProductUseCase{
		productRepo:   productRepo,
		gameTitleRepo: gameTitleRepo,
		userRepo:      userRepo,
	}
}

type CreateProductInput struct {
	GameTitleID string                 `json:"game_title_id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Price       float64                `json:"price"`
	Type        string                 `json:"type"` // account, topup, boosting, item
	Attributes  map[string]interface{} `json:"attributes"`
	Status      string                 `json:"status"`
}

type ProductImageInput struct {
	URL         string `json:"url"`
	DisplayOrder int    `json:"display_order"`
}

func (uc *ProductUseCase) CreateProduct(ctx context.Context, sellerID string, input CreateProductInput, images []ProductImageInput) (*entity.Product, error) {
	// Validate game title
	gameTitle, err := uc.gameTitleRepo.GetByID(ctx, input.GameTitleID)
	if err != nil {
		return nil, errors.BadRequest("Invalid game title", err)
	}

	// Validate user
	_, err = uc.userRepo.GetByID(ctx, sellerID)
	if err != nil {
		return nil, errors.BadRequest("Invalid seller", err)
	}

	// Convert images
	productImages := make([]entity.ProductImage, len(images))
	for i, img := range images {
		productImages[i] = entity.ProductImage{
			ID:          generateUUID(), // Implement this function
			URL:         img.URL,
			DisplayOrder: img.DisplayOrder,
		}
	}

	// Create product
	product := &entity.Product{
		GameTitleID: gameTitle.ID,
		SellerID:    sellerID,
		Title:       input.Title,
		Description: input.Description,
		Price:       input.Price,
		Type:        input.Type,
		Attributes:  input.Attributes,
		Status:      input.Status,
		Images:      productImages,
		Views:       0,
		Featured:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save to repository
	if err := uc.productRepo.Create(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (uc *ProductUseCase) GetProductByID(ctx context.Context, id string) (*entity.Product, error) {
	product, err := uc.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Increment view counter (async)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = uc.productRepo.IncrementViews(ctx, id)
	}()

	return product, nil
}

func (uc *ProductUseCase) ListProducts(ctx context.Context, gameTitleID, productType, status string, minPrice, maxPrice float64, sort string, page, limit int) ([]*entity.Product, int64, error) {
	// Build filter
	filter := make(map[string]interface{})
	
	if gameTitleID != "" {
		filter["gameTitleId"] = gameTitleID
	}
	
	if productType != "" {
		filter["type"] = productType
	}
	
	if status != "" {
		filter["status"] = status
	} else {
		// Default to active products
		filter["status"] = "active"
	}

	// Calculate offset
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	return uc.productRepo.List(ctx, filter, sort, limit, offset)
}

func (uc *ProductUseCase) UpdateProduct(ctx context.Context, id string, sellerID string, input CreateProductInput, images []ProductImageInput) (*entity.Product, error) {
	// Get existing product
	product, err := uc.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if product.SellerID != sellerID {
		return nil, errors.Forbidden("You don't have permission to update this product", nil)
	}

	// Validate game title if changing
	if input.GameTitleID != product.GameTitleID {
		_, err = uc.gameTitleRepo.GetByID(ctx, input.GameTitleID)
		if err != nil {
			return nil, errors.BadRequest("Invalid game title", err)
		}
		product.GameTitleID = input.GameTitleID
	}

	// Update fields
	product.Title = input.Title
	product.Description = input.Description
	product.Price = input.Price
	product.Type = input.Type
	product.Attributes = input.Attributes
	product.Status = input.Status
	product.UpdatedAt = time.Now()

	// Update images if provided
	if len(images) > 0 {
		productImages := make([]entity.ProductImage, len(images))
		for i, img := range images {
			productImages[i] = entity.ProductImage{
				ID:          generateUUID(), // Implement this function
				URL:         img.URL,
				DisplayOrder: img.DisplayOrder,
			}
		}
		product.Images = productImages
	}

	// Save to repository
	if err := uc.productRepo.Update(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (uc *ProductUseCase) DeleteProduct(ctx context.Context, id string, sellerID string) error {
	// Get existing product
	product, err := uc.productRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Check ownership
	if product.SellerID != sellerID {
		return errors.Forbidden("You don't have permission to delete this product", nil)
	}

	// Soft delete
	return uc.productRepo.SoftDelete(ctx, id)
}

// Helper function
func generateUUID() string {
	// Implement a UUID generator
	// For example, using google/uuid package:
	// return uuid.New().String()
	return "img-" + time.Now().Format("20060102150405-999999999")
}

func (uc *ProductUseCase) ListBySellerID(ctx context.Context, sellerID, status string, limit, offset int) ([]*entity.Product, int64, error) {
    // Validate user
    _, err := uc.userRepo.GetByID(ctx, sellerID)
    if err != nil {
        return nil, 0, errors.BadRequest("Invalid seller", err)
    }

    // Call repository
    return uc.productRepo.ListBySellerID(ctx, sellerID, status, limit, offset)
}

func (uc *ProductUseCase) BumpProduct(ctx context.Context, productID string, sellerID string) (*entity.Product, error) {
    // Get existing product
    product, err := uc.productRepo.GetByID(ctx, productID)
    if err != nil {
        return nil, err
    }

    // Check ownership
    if product.SellerID != sellerID {
        return nil, errors.Forbidden("You don't have permission to bump this product", nil)
    }

    // Check if product is active
    if product.Status != "active" {
        return nil, errors.BadRequest("Only active products can be bumped", nil)
    }

    // Update bumpedAt to current time
    product.BumpedAt = time.Now()
    product.UpdatedAt = time.Now()

    // Save to repository
    if err := uc.productRepo.Update(ctx, product); err != nil {
        return nil, err
    }

    return product, nil
}