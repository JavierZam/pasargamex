package handler

import (
	"log"
	"strconv"

	"pasargamex/internal/usecase"
	"pasargamex/pkg/errors"
	"pasargamex/pkg/response"
	"pasargamex/pkg/utils"

	"github.com/labstack/echo/v4"
)

type ProductHandler struct {
	productUseCase *usecase.ProductUseCase
}

func NewProductHandler(productUseCase *usecase.ProductUseCase) *ProductHandler {
	return &ProductHandler{
		productUseCase: productUseCase,
	}
}

type productImageRequest struct {
	URL          string `json:"url" validate:"required,url"`
	DisplayOrder int    `json:"display_order"`
}

type validateCredentialsRequest struct {
	ProductID   string                 `json:"product_id" validate:"required"`
	Credentials map[string]interface{} `json:"credentials" validate:"required"`
}

type createProductRequest struct {
	GameTitleID    string                 `json:"game_title_id" validate:"required"`
	Title          string                 `json:"title" validate:"required"`
	Description    string                 `json:"description"`
	Price          float64                `json:"price" validate:"required,gt=0"`
	Type           string                 `json:"type" validate:"required,oneof=account topup boosting item"`
	Attributes     map[string]interface{} `json:"attributes"`
	Images         []productImageRequest  `json:"images"`
	Status         string                 `json:"status" validate:"required,oneof=draft active"`
	DeliveryMethod string                 `json:"delivery_method" validate:"required,oneof=instant middleman both"`
	Credentials    map[string]interface{} `json:"credentials,omitempty"`
}

func (h *ProductHandler) CreateProduct(c echo.Context) error {
	var req createProductRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	sellerID := c.Get("uid").(string)

	images := make([]usecase.ProductImageInput, len(req.Images))
	for i, img := range req.Images {
		images[i] = usecase.ProductImageInput{
			URL:          img.URL,
			DisplayOrder: img.DisplayOrder,
		}
	}

	product, err := h.productUseCase.CreateProduct(
		c.Request().Context(),
		sellerID,
		usecase.CreateProductInput{
			GameTitleID:    req.GameTitleID,
			Title:          req.Title,
			Description:    req.Description,
			Price:          req.Price,
			Type:           req.Type,
			Attributes:     req.Attributes,
			Status:         req.Status,
			DeliveryMethod: req.DeliveryMethod,
			Credentials:    req.Credentials,
		},
		images,
	)

	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, product)
}

func (h *ProductHandler) UpdateProduct(c echo.Context) error {
	id := c.Param("id")

	var req createProductRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	sellerID := c.Get("uid").(string)

	images := make([]usecase.ProductImageInput, len(req.Images))
	for i, img := range req.Images {
		images[i] = usecase.ProductImageInput{
			URL:          img.URL,
			DisplayOrder: img.DisplayOrder,
		}
	}

	product, err := h.productUseCase.UpdateProduct(
		c.Request().Context(),
		id,
		sellerID,
		usecase.CreateProductInput{
			GameTitleID:    req.GameTitleID,
			Title:          req.Title,
			Description:    req.Description,
			Price:          req.Price,
			Type:           req.Type,
			Attributes:     req.Attributes,
			Status:         req.Status,
			DeliveryMethod: req.DeliveryMethod,
			Credentials:    req.Credentials,
		},
		images,
	)

	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, product)
}

func (h *ProductHandler) GetProduct(c echo.Context) error {
	id := c.Param("id")

	var currentUserID string
	if uid, ok := c.Get("uid").(string); ok && uid != "" {
		currentUserID = uid
	}

	product, err := h.productUseCase.GetProductByID(c.Request().Context(), id, currentUserID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, product)
}

func (h *ProductHandler) ListProducts(c echo.Context) error {

	gameTitleID := c.QueryParam("game_title_id")
	productType := c.QueryParam("type")
	status := c.QueryParam("status")
	sort := c.QueryParam("sort")

	minPriceStr := c.QueryParam("min_price")
	maxPriceStr := c.QueryParam("max_price")

	var minPrice, maxPrice float64
	var err error

	if minPriceStr != "" {
		minPrice, err = strconv.ParseFloat(minPriceStr, 64)
		if err != nil {
			return response.Error(c, err)
		}
	}

	if maxPriceStr != "" {
		maxPrice, err = strconv.ParseFloat(maxPriceStr, 64)
		if err != nil {
			return response.Error(c, err)
		}
	}

	pagination := utils.GetPaginationParams(c)

	products, total, err := h.productUseCase.ListProducts(
		c.Request().Context(),
		gameTitleID,
		productType,
		status,
		minPrice,
		maxPrice,
		sort,
		pagination.Page,
		pagination.PageSize,
	)

	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, products, total, pagination.Page, pagination.PageSize)
}

func (h *ProductHandler) SearchProducts(c echo.Context) error {
	log.Printf("=== SearchProducts handler called ===")

	query := c.QueryParam("q")
	if query == "" {
		return response.Error(c, errors.BadRequest("Search query is required", nil))
	}

	log.Printf("Search query: '%s'", query)

	gameTitleID := c.QueryParam("game_title_id")
	productType := c.QueryParam("type")
	status := c.QueryParam("status")

	if status == "" {
		status = "active"
	}

	var minPrice, maxPrice float64
	minPriceStr := c.QueryParam("min_price")
	maxPriceStr := c.QueryParam("max_price")

	if minPriceStr != "" {
		var err error
		minPrice, err = strconv.ParseFloat(minPriceStr, 64)
		if err != nil {
			log.Printf("Error parsing min_price '%s': %v", minPriceStr, err)

			minPrice = 0
		} else {
			log.Printf("Using min_price filter: %.2f", minPrice)
		}
	}

	if maxPriceStr != "" {
		var err error
		maxPrice, err = strconv.ParseFloat(maxPriceStr, 64)
		if err != nil {
			log.Printf("Error parsing max_price '%s': %v", maxPriceStr, err)

			maxPrice = 0
		} else {
			log.Printf("Using max_price filter: %.2f", maxPrice)
		}
	}

	pagination := utils.GetPaginationParams(c)

	products, total, err := h.productUseCase.SearchProducts(
		c.Request().Context(),
		query,
		gameTitleID,
		productType,
		status,
		minPrice,
		maxPrice,
		pagination.Page,
		pagination.PageSize,
	)

	if err != nil {
		log.Printf("Error searching products: %v", err)
		return response.Error(c, err)
	}

	log.Printf("Search returned %d products", len(products))

	return response.Paginated(c, products, total, pagination.Page, pagination.PageSize)
}

func (h *ProductHandler) ListMyProducts(c echo.Context) error {

	sellerID := c.Get("uid").(string)

	pagination := utils.GetPaginationParams(c)

	status := c.QueryParam("status")

	products, total, err := h.productUseCase.ListBySellerID(
		c.Request().Context(),
		sellerID,
		status,
		pagination.PageSize,
		pagination.Offset,
	)

	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, products, total, pagination.Page, pagination.PageSize)
}

func (h *ProductHandler) DeleteProduct(c echo.Context) error {
	id := c.Param("id")

	sellerID := c.Get("uid").(string)

	err := h.productUseCase.DeleteProduct(c.Request().Context(), id, sellerID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, map[string]string{
		"message": "Product deleted successfully",
	})
}

func (h *ProductHandler) DeleteProductImage(c echo.Context) error {
	productID := c.Param("id")
	imageID := c.Param("imageId")

	if productID == "" {
		return response.Error(c, errors.BadRequest("Product ID is required", nil))
	}

	if imageID == "" {
		return response.Error(c, errors.BadRequest("Image ID is required", nil))
	}

	sellerID := c.Get("uid").(string)

	product, err := h.productUseCase.DeleteProductImage(c.Request().Context(), productID, imageID, sellerID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, map[string]interface{}{
		"message": "Image deleted successfully",
		"product": product,
	})
}

func (h *ProductHandler) BumpProduct(c echo.Context) error {
	id := c.Param("id")

	sellerID := c.Get("uid").(string)

	product, err := h.productUseCase.BumpProduct(c.Request().Context(), id, sellerID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, map[string]interface{}{
		"message": "Product bumped successfully",
		"product": product,
	})
}

func (h *ProductHandler) MigrateProductsBumpedAt(c echo.Context) error {

	if err := h.productUseCase.MigrateProductsBumpedAt(c.Request().Context()); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, map[string]string{
		"message": "Products migration completed successfully",
	})
}

func (h *ProductHandler) ValidateCredentials(c echo.Context) error {
	var req validateCredentialsRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	adminID := c.Get("uid").(string)

	result, err := h.productUseCase.ValidateCredentials(
		c.Request().Context(),
		adminID,
		req.ProductID,
		req.Credentials,
	)

	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, map[string]interface{}{
		"product_id": req.ProductID,
		"validated":  result,
		"message":    "Credentials have been validated",
	})
}

func (h *ProductHandler) GetProductsBySeller(c echo.Context) error {
	sellerID := c.Param("sellerId")
	if sellerID == "" {
		return response.Error(c, errors.BadRequest("Seller ID is required", nil))
	}

	productType := c.QueryParam("type")
	status := c.QueryParam("status")

	pagination := utils.GetPaginationParams(c)

	products, total, err := h.productUseCase.ListProductsBySeller(
		c.Request().Context(),
		sellerID,
		productType,
		status,
		pagination.Page,
		pagination.PageSize,
	)

	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, products, total, pagination.Page, pagination.PageSize)
}

func (h *ProductHandler) GetSellerProfile(c echo.Context) error {
	sellerID := c.Param("sellerId")
	if sellerID == "" {
		return response.Error(c, errors.BadRequest("Seller ID is required", nil))
	}

	productType := c.QueryParam("type")
	pagination := utils.GetPaginationParams(c)

	profileData, err := h.productUseCase.GetSellerProfileWithProducts(
		c.Request().Context(),
		sellerID,
		productType,
		pagination.Page,
		pagination.PageSize,
	)

	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, profileData)
}
