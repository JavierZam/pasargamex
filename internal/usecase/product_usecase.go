package usecase

import (
	"context"
	"log"
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
	GameTitleID    string                 `json:"game_title_id"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Price          float64                `json:"price"`
	Type           string                 `json:"type"` // account, topup, boosting, item
	Attributes     map[string]interface{} `json:"attributes"`
	Status         string                 `json:"status"`
	DeliveryMethod string                 `json:"delivery_method"` // "instant", "middleman", "both"
	Credentials    map[string]interface{} `json:"credentials,omitempty"`
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
	seller, err := uc.userRepo.GetByID(ctx, sellerID)
	if err != nil {
		return nil, errors.BadRequest("Invalid seller", err)
	}

	// Validate seller is verified if using instant delivery
	if (input.DeliveryMethod == "instant" || input.DeliveryMethod == "both") && seller.VerificationStatus != "verified" {
		return nil, errors.BadRequest("Seller must be verified to use instant delivery", nil)
	}

	// Validate delivery method
	if input.DeliveryMethod != "instant" && input.DeliveryMethod != "middleman" && input.DeliveryMethod != "both" {
		return nil, errors.BadRequest("Invalid delivery method", nil)
	}

	// For instant delivery, credentials are required
	if (input.DeliveryMethod == "instant" || input.DeliveryMethod == "both") && len(input.Credentials) == 0 {
		return nil, errors.BadRequest("Credentials are required for instant delivery", nil)
	}

	// Convert images
	productImages := make([]entity.ProductImage, len(images))
	for i, img := range images {
		productImages[i] = entity.ProductImage{
			ID:           generateUUID(), // Implement this function
			URL:          img.URL,
			DisplayOrder: img.DisplayOrder,
		}
	}

	// Create product
	product := &entity.Product{
		GameTitleID:         gameTitle.ID,
		SellerID:            sellerID,
		Title:               input.Title,
		Description:         input.Description,
		Price:               input.Price,
		Type:                input.Type,
		Attributes:          input.Attributes,
		Status:              input.Status,
		Images:              productImages,
		Views:               0,
		Featured:            false,
		DeliveryMethod:      input.DeliveryMethod,
		Credentials:         input.Credentials,
		CredentialsValidated: false, // Credentials need to be validated by admin
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
		BumpedAt:            time.Now(),
	}

	// Save to repository
	if err := uc.productRepo.Create(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
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
	
	// Validate seller is verified if using instant delivery
	if (input.DeliveryMethod == "instant" || input.DeliveryMethod == "both") {
		seller, err := uc.userRepo.GetByID(ctx, sellerID)
		if err != nil {
			return nil, errors.BadRequest("Invalid seller", err)
		}
		
		if seller.VerificationStatus != "verified" {
			return nil, errors.BadRequest("Seller must be verified to use instant delivery", nil)
		}
	}

	// Validate delivery method
	if input.DeliveryMethod != "instant" && input.DeliveryMethod != "middleman" && input.DeliveryMethod != "both" {
		return nil, errors.BadRequest("Invalid delivery method", nil)
	}

	// For instant delivery, credentials are required
	if (input.DeliveryMethod == "instant" || input.DeliveryMethod == "both") && 
	   len(input.Credentials) == 0 && len(product.Credentials) == 0 {
		return nil, errors.BadRequest("Credentials are required for instant delivery", nil)
	}

	// Update fields
	product.Title = input.Title
	product.Description = input.Description
	product.Price = input.Price
	product.Type = input.Type
	product.Attributes = input.Attributes
	product.Status = input.Status
	product.DeliveryMethod = input.DeliveryMethod
	product.UpdatedAt = time.Now()
	
	// Update credentials if provided
	if len(input.Credentials) > 0 {
		product.Credentials = input.Credentials
		product.CredentialsValidated = false // Reset validation when credentials are updated
	}

	// Update images if provided
	if len(images) > 0 {
		productImages := make([]entity.ProductImage, len(images))
		for i, img := range images {
			productImages[i] = entity.ProductImage{
				ID:           generateUUID(), // Implement this function
				URL:          img.URL,
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
    
    // Add price filters
    if minPrice > 0 {
        filter["min_price"] = minPrice
    }
    
    if maxPrice > 0 {
        filter["max_price"] = maxPrice
    }

    // Calculate offset
    offset := (page - 1) * limit
    if offset < 0 {
        offset = 0
    }

    return uc.productRepo.List(ctx, filter, sort, limit, offset)
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

func (uc *ProductUseCase) MigrateProductsBumpedAt(ctx context.Context) error {
    // Ambil semua produk
    products, _, err := uc.productRepo.List(ctx, nil, "", 1000, 0)
    if err != nil {
        return err
    }
    
    // Update setiap produk jika bumpedAt tidak ada
    for _, product := range products {
        if product.BumpedAt.IsZero() {
            product.BumpedAt = product.CreatedAt
            if err := uc.productRepo.Update(ctx, product); err != nil {
                return err
            }
        }
    }
    
    return nil
}

func (uc *ProductUseCase) SearchProducts(ctx context.Context, query string, gameTitleID, productType, status string, minPrice, maxPrice float64, page, limit int) ([]*entity.Product, int64, error) {
    log.Printf("SearchProducts usecase called with query: '%s'", query)
    
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
    }
    
    // Add price filters
    if minPrice > 0 {
        filter["min_price"] = minPrice
    }
    
    if maxPrice > 0 {
        filter["max_price"] = maxPrice
    }
    
    // Calculate offset
    offset := (page - 1) * limit
    if offset < 0 {
        offset = 0
    }
    
    // Call repository.Search, not repository.List
    return uc.productRepo.Search(ctx, query, filter, limit, offset)
}

func (uc *ProductUseCase) ValidateCredentials(ctx context.Context, adminID string, productID string, credentials map[string]interface{}) (bool, error) {
	// TODO: Validate that adminID is actually an admin
	
	// Get the product
	product, err := uc.productRepo.GetByID(ctx, productID)
	if err != nil {
		return false, err
	}
	
	// Check if product uses instant delivery
	if product.DeliveryMethod != "instant" && product.DeliveryMethod != "both" {
		return false, errors.BadRequest("Product does not use instant delivery", nil)
	}
	
	// Compare credentials provided with product credentials
	// In a real-world scenario, you'd have more sophisticated validation
	// This is a simplified example
	if len(credentials) == 0 {
		return false, errors.BadRequest("No credentials provided", nil)
	}
	
	// Update product to mark credentials as validated
	product.CredentialsValidated = true
	product.UpdatedAt = time.Now()
	
	if err := uc.productRepo.Update(ctx, product); err != nil {
		return false, errors.Internal("Failed to update product validation status", err)
	}
	
	return true, nil
}