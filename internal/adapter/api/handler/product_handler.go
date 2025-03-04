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

type createProductRequest struct {
	GameTitleID string                 `json:"game_title_id" validate:"required"`
	Title       string                 `json:"title" validate:"required"`
	Description string                 `json:"description"`
	Price       float64                `json:"price" validate:"required,gt=0"`
	Type        string                 `json:"type" validate:"required,oneof=account topup boosting item"`
	Attributes  map[string]interface{} `json:"attributes"`
	Images      []productImageRequest  `json:"images"`
	Status      string                 `json:"status" validate:"required,oneof=draft active"`
}

func (h *ProductHandler) CreateProduct(c echo.Context) error {
	var req createProductRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	// Get user ID from context
	sellerID := c.Get("uid").(string)

	// Convert images
	images := make([]usecase.ProductImageInput, len(req.Images))
	for i, img := range req.Images {
		images[i] = usecase.ProductImageInput{
			URL:          img.URL,
			DisplayOrder: img.DisplayOrder,
		}
	}

	// Call use case
	product, err := h.productUseCase.CreateProduct(
		c.Request().Context(),
		sellerID,
		usecase.CreateProductInput{
			GameTitleID: req.GameTitleID,
			Title:       req.Title,
			Description: req.Description,
			Price:       req.Price,
			Type:        req.Type,
			Attributes:  req.Attributes,
			Status:      req.Status,
		},
		images,
	)

	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, product)
}

func (h *ProductHandler) GetProduct(c echo.Context) error {
	id := c.Param("id")

	product, err := h.productUseCase.GetProductByID(c.Request().Context(), id)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, product)
}

func (h *ProductHandler) ListProducts(c echo.Context) error {
	// Parse query parameters
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
	
	// Get pagination parameters
	pagination := utils.GetPaginationParams(c)

	// Call use case
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
    
    // Parse query parameters
    query := c.QueryParam("q")
    if query == "" {
        return response.Error(c, errors.BadRequest("Search query is required", nil))
    }
    
    log.Printf("Search query: '%s'", query)
    
    // Parse other parameters
    gameTitleID := c.QueryParam("game_title_id")
    productType := c.QueryParam("type")
    status := c.QueryParam("status")
    
    // Default status to active if not provided
    if status == "" {
        status = "active"
    }
    
    // PERBAIKAN: Parsing price dengan logging yang lebih baik
    var minPrice, maxPrice float64
    minPriceStr := c.QueryParam("min_price")
    maxPriceStr := c.QueryParam("max_price")
    
    if minPriceStr != "" {
        var err error
        minPrice, err = strconv.ParseFloat(minPriceStr, 64)
        if err != nil {
            log.Printf("Error parsing min_price '%s': %v", minPriceStr, err)
            // Default to 0 instead of returning error
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
            // Default to 0 instead of returning error
            maxPrice = 0
        } else {
            log.Printf("Using max_price filter: %.2f", maxPrice)
        }
    }
    
    // Get pagination parameters
    pagination := utils.GetPaginationParams(c)
    
    // Call use case for search specifically
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
    // Get user ID from context
    sellerID := c.Get("uid").(string)
    
    // Get pagination parameters
    pagination := utils.GetPaginationParams(c)
    
    // Get status filter
    status := c.QueryParam("status")
    
    // Call use case - Gunakan ListBySellerID bukan ListProducts
    products, total, err := h.productUseCase.ListBySellerID(
        c.Request().Context(),
        sellerID,       // Gunakan sellerID di sini
        status,
        pagination.PageSize,
        pagination.Offset,
    )
    
    if err != nil {
        return response.Error(c, err)
    }
    
    return response.Paginated(c, products, total, pagination.Page, pagination.PageSize)
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
	
	// Get user ID from context
	sellerID := c.Get("uid").(string)
	
	// Convert images
	images := make([]usecase.ProductImageInput, len(req.Images))
	for i, img := range req.Images {
		images[i] = usecase.ProductImageInput{
			URL:          img.URL,
			DisplayOrder: img.DisplayOrder,
		}
	}
	
	// Call use case
	product, err := h.productUseCase.UpdateProduct(
		c.Request().Context(),
		id,
		sellerID,
		usecase.CreateProductInput{
			GameTitleID: req.GameTitleID,
			Title:       req.Title,
			Description: req.Description,
			Price:       req.Price,
			Type:        req.Type,
			Attributes:  req.Attributes,
			Status:      req.Status,
		},
		images,
	)
	
	if err != nil {
		return response.Error(c, err)
	}
	
	return response.Success(c, product)
}

func (h *ProductHandler) DeleteProduct(c echo.Context) error {
	id := c.Param("id")
	
	// Get user ID from context
	sellerID := c.Get("uid").(string)
	
	// Call use case
	err := h.productUseCase.DeleteProduct(c.Request().Context(), id, sellerID)
	if err != nil {
		return response.Error(c, err)
	}
	
	return response.Success(c, map[string]string{
		"message": "Product deleted successfully",
	})
}

func (h *ProductHandler) BumpProduct(c echo.Context) error {
    id := c.Param("id")
    
    // Get user ID from context
    sellerID := c.Get("uid").(string)
    
    // Call use case
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
    // Hanya admin yang bisa menjalankan migrasi
    // Tambahkan pengecekan role admin di sini
    
    if err := h.productUseCase.MigrateProductsBumpedAt(c.Request().Context()); err != nil {
        return response.Error(c, err)
    }
    
    return response.Success(c, map[string]string{
        "message": "Products migration completed successfully",
    })
}