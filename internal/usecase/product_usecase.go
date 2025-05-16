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
	productRepo     repository.ProductRepository
	gameTitleRepo   repository.GameTitleRepository
	userRepo        repository.UserRepository
	transactionRepo repository.TransactionRepository
}

func NewProductUseCase(
	productRepo repository.ProductRepository,
	gameTitleRepo repository.GameTitleRepository,
	userRepo repository.UserRepository,
	transactionRepo repository.TransactionRepository,

) *ProductUseCase {
	return &ProductUseCase{
		productRepo:     productRepo,
		gameTitleRepo:   gameTitleRepo,
		userRepo:        userRepo,
		transactionRepo: transactionRepo,
	}
}

type CreateProductInput struct {
	GameTitleID    string                 `json:"game_title_id"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Price          float64                `json:"price"`
	Type           string                 `json:"type"`
	Attributes     map[string]interface{} `json:"attributes"`
	Status         string                 `json:"status"`
	DeliveryMethod string                 `json:"delivery_method"`
	Credentials    map[string]interface{} `json:"credentials,omitempty"`
}

type ProductImageInput struct {
	URL          string `json:"url"`
	DisplayOrder int    `json:"display_order"`
}

func (uc *ProductUseCase) CreateProduct(ctx context.Context, sellerID string, input CreateProductInput, images []ProductImageInput) (*entity.Product, error) {

	gameTitle, err := uc.gameTitleRepo.GetByID(ctx, input.GameTitleID)
	if err != nil {
		return nil, errors.BadRequest("Invalid game title", err)
	}

	seller, err := uc.userRepo.GetByID(ctx, sellerID)
	if err != nil {
		return nil, errors.BadRequest("Invalid seller", err)
	}

	if (input.DeliveryMethod == "instant" || input.DeliveryMethod == "both") && seller.VerificationStatus != "verified" {
		return nil, errors.BadRequest("Seller must be verified to use instant delivery", nil)
	}

	if input.DeliveryMethod != "instant" && input.DeliveryMethod != "middleman" && input.DeliveryMethod != "both" {
		return nil, errors.BadRequest("Invalid delivery method", nil)
	}

	if (input.DeliveryMethod == "instant" || input.DeliveryMethod == "both") && len(input.Credentials) == 0 {
		return nil, errors.BadRequest("Credentials are required for instant delivery", nil)
	}

	productImages := make([]entity.ProductImage, len(images))
	for i, img := range images {
		productImages[i] = entity.ProductImage{
			ID:           generateUUID(),
			URL:          img.URL,
			DisplayOrder: img.DisplayOrder,
		}
	}

	product := &entity.Product{
		GameTitleID:          gameTitle.ID,
		SellerID:             sellerID,
		Title:                input.Title,
		Description:          input.Description,
		Price:                input.Price,
		Type:                 input.Type,
		Attributes:           input.Attributes,
		Status:               input.Status,
		Images:               productImages,
		Views:                0,
		Featured:             false,
		DeliveryMethod:       input.DeliveryMethod,
		Credentials:          input.Credentials,
		CredentialsValidated: false,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
		BumpedAt:             time.Now(),
	}

	if err := uc.productRepo.Create(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (uc *ProductUseCase) UpdateProduct(ctx context.Context, id string, sellerID string, input CreateProductInput, images []ProductImageInput) (*entity.Product, error) {

	product, err := uc.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if product.SellerID != sellerID {
		return nil, errors.Forbidden("You don't have permission to update this product", nil)
	}

	if input.GameTitleID != product.GameTitleID {
		_, err = uc.gameTitleRepo.GetByID(ctx, input.GameTitleID)
		if err != nil {
			return nil, errors.BadRequest("Invalid game title", err)
		}
		product.GameTitleID = input.GameTitleID
	}

	if input.DeliveryMethod == "instant" || input.DeliveryMethod == "both" {
		seller, err := uc.userRepo.GetByID(ctx, sellerID)
		if err != nil {
			return nil, errors.BadRequest("Invalid seller", err)
		}

		if seller.VerificationStatus != "verified" {
			return nil, errors.BadRequest("Seller must be verified to use instant delivery", nil)
		}
	}

	if input.DeliveryMethod != "instant" && input.DeliveryMethod != "middleman" && input.DeliveryMethod != "both" {
		return nil, errors.BadRequest("Invalid delivery method", nil)
	}

	if (input.DeliveryMethod == "instant" || input.DeliveryMethod == "both") &&
		len(input.Credentials) == 0 && len(product.Credentials) == 0 {
		return nil, errors.BadRequest("Credentials are required for instant delivery", nil)
	}

	product.Title = input.Title
	product.Description = input.Description
	product.Price = input.Price
	product.Type = input.Type
	product.Attributes = input.Attributes
	product.Status = input.Status
	product.DeliveryMethod = input.DeliveryMethod
	product.UpdatedAt = time.Now()

	if len(input.Credentials) > 0 {
		product.Credentials = input.Credentials
		product.CredentialsValidated = false
	}

	if len(images) > 0 {
		productImages := make([]entity.ProductImage, len(images))
		for i, img := range images {
			productImages[i] = entity.ProductImage{
				ID:           generateUUID(),
				URL:          img.URL,
				DisplayOrder: img.DisplayOrder,
			}
		}
		product.Images = productImages
	}

	if err := uc.productRepo.Update(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (uc *ProductUseCase) GetProductByID(ctx context.Context, id string, currentUserID string) (*entity.Product, error) {
	log.Printf("GetProductByID called with id=%s, currentUserID=%s", id, currentUserID)

	product, err := uc.productRepo.GetByID(ctx, id)
	if err != nil {
		log.Printf("Error getting product: %v", err)
		return nil, err
	}

	log.Printf("Product found: id=%s, sellerID=%s, hasCredentials=%v",
		product.ID, product.SellerID, product.Credentials != nil)

	isSeller := currentUserID != "" && product.SellerID == currentUserID
	log.Printf("Current user is seller: %v", isSeller)

	if isSeller {
		log.Printf("Returning product with credentials to seller")
		return product, nil
	}

	if currentUserID != "" {
		hasCompletedTransaction, err := uc.transactionRepo.HasCompletedTransaction(ctx, currentUserID, id)
		log.Printf("User has completed transaction: %v, err: %v", hasCompletedTransaction, err)

		if err == nil && hasCompletedTransaction {
			log.Printf("Returning product with credentials to buyer with completed transaction")
			return product, nil
		}
	}

	log.Printf("Hiding credentials for non-seller/non-buyer user")
	productCopy := *product
	productCopy.Credentials = nil
	return &productCopy, nil
}

func (uc *ProductUseCase) ListProducts(ctx context.Context, gameTitleID, productType, status string, minPrice, maxPrice float64, sort string, page, limit int) ([]*entity.Product, int64, error) {

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

		filter["status"] = "active"
	}

	if minPrice > 0 {
		filter["min_price"] = minPrice
	}

	if maxPrice > 0 {
		filter["max_price"] = maxPrice
	}

	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	return uc.productRepo.List(ctx, filter, sort, limit, offset)
}

func (uc *ProductUseCase) DeleteProduct(ctx context.Context, id string, sellerID string) error {

	product, err := uc.productRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if product.SellerID != sellerID {
		return errors.Forbidden("You don't have permission to delete this product", nil)
	}

	return uc.productRepo.SoftDelete(ctx, id)
}

func generateUUID() string {

	return "img-" + time.Now().Format("20060102150405-999999999")
}

func (uc *ProductUseCase) ListBySellerID(ctx context.Context, sellerID, status string, limit, offset int) ([]*entity.Product, int64, error) {

	_, err := uc.userRepo.GetByID(ctx, sellerID)
	if err != nil {
		return nil, 0, errors.BadRequest("Invalid seller", err)
	}

	return uc.productRepo.ListBySellerID(ctx, sellerID, status, limit, offset)
}

func (uc *ProductUseCase) BumpProduct(ctx context.Context, productID string, sellerID string) (*entity.Product, error) {

	product, err := uc.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	if product.SellerID != sellerID {
		return nil, errors.Forbidden("You don't have permission to bump this product", nil)
	}

	if product.Status != "active" {
		return nil, errors.BadRequest("Only active products can be bumped", nil)
	}

	product.BumpedAt = time.Now()
	product.UpdatedAt = time.Now()

	if err := uc.productRepo.Update(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (uc *ProductUseCase) MigrateProductsBumpedAt(ctx context.Context) error {

	products, _, err := uc.productRepo.List(ctx, nil, "", 1000, 0)
	if err != nil {
		return err
	}

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

	if minPrice > 0 {
		filter["min_price"] = minPrice
	}

	if maxPrice > 0 {
		filter["max_price"] = maxPrice
	}

	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	return uc.productRepo.Search(ctx, query, filter, limit, offset)
}

func (uc *ProductUseCase) ValidateCredentials(ctx context.Context, adminID string, productID string, credentials map[string]interface{}) (bool, error) {

	product, err := uc.productRepo.GetByID(ctx, productID)
	if err != nil {
		return false, err
	}

	if product.DeliveryMethod != "instant" && product.DeliveryMethod != "both" {
		return false, errors.BadRequest("Product does not use instant delivery", nil)
	}

	if len(credentials) == 0 {
		return false, errors.BadRequest("No credentials provided", nil)
	}

	product.CredentialsValidated = true
	product.UpdatedAt = time.Now()

	if err := uc.productRepo.Update(ctx, product); err != nil {
		return false, errors.Internal("Failed to update product validation status", err)
	}

	return true, nil
}
