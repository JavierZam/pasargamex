package handler

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/internal/domain/service"
	"pasargamex/pkg/response"
)

type FileHandler struct {
	fileService      service.FileUploadService
	fileMetadataRepo repository.FileMetadataRepository
	productRepo      repository.ProductRepository
	maxFileSize      int64
}

func NewFileHandler(fileService service.FileUploadService, fileMetadataRepo repository.FileMetadataRepository, productRepo repository.ProductRepository) *FileHandler {
	return &FileHandler{
		fileService:      fileService,
		fileMetadataRepo: fileMetadataRepo,
		productRepo:      productRepo,
		maxFileSize:      5 * 1024 * 1024,
	}
}

func SetupFileHandler(fileService service.FileUploadService, fileMetadataRepo repository.FileMetadataRepository, productRepo repository.ProductRepository) {
	fileHandler = NewFileHandler(fileService, fileMetadataRepo, productRepo)
}

var (
	fileHandler *FileHandler
)

func GetFileHandler() *FileHandler {
	return fileHandler
}

func (h *FileHandler) UploadFile(c echo.Context) error {
	log.Printf("Starting file upload handler")

	// Get the file from the request
	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("Error getting file from form: %v", err)
		return response.Error(c, response.NewError("INVALID_FILE", "Missing or invalid file", http.StatusBadRequest))
	}

	log.Printf("Received file: %s, size: %d bytes, type: %s", file.Filename, file.Size, file.Header.Get("Content-Type"))

	// Check file size
	if file.Size > h.maxFileSize {
		log.Printf("File too large: %d bytes (max: %d)", file.Size, h.maxFileSize)
		return response.Error(c, response.NewError("FILE_TOO_LARGE", fmt.Sprintf("File size exceeds maximum allowed (%dMB)", h.maxFileSize/(1024*1024)), http.StatusBadRequest))
	}

	// Validate file type
	fileType := file.Header.Get("Content-Type")
	if !isAllowedFileType(fileType) {
		log.Printf("Invalid file type: %s", fileType)
		return response.Error(c, response.NewError("INVALID_FILE_TYPE", "File type not supported", http.StatusBadRequest))
	}

	// Get the folder from the request
	folder := c.FormValue("folder")
	if folder == "" {
		// Default to a general uploads folder
		folder = "uploads"
	} else {
		// Sanitize folder name
		folder = sanitizeFolderName(folder)
	}
	log.Printf("Using folder: %s", folder)

	// Determine if file should be public
	isPublicStr := c.FormValue("public")
	isPublic := true // Default to public for backward compatibility
	if isPublicStr != "" {
		isPublic, _ = strconv.ParseBool(isPublicStr)
	}
	log.Printf("Public file: %v", isPublic)

	// Open the file
	src, err := file.Open()
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return response.Error(c, response.NewError("FILE_READ_ERROR", "Unable to read file", http.StatusInternalServerError))
	}
	defer src.Close()

	// Upload the file
	log.Printf("Calling storage client UploadFile")
	fileURL, err := h.fileService.UploadFile(c.Request().Context(), src, fileType, folder, isPublic)
	if err != nil {
		log.Printf("Error from storage client: %v", err)
		return response.Error(c, response.NewError("UPLOAD_FAILED", fmt.Sprintf("Failed to upload file: %v", err), http.StatusInternalServerError))
	}
	log.Printf("Storage client returned URL: %s", fileURL)

	// Temporarily disable metadata storage for testing
	/*
	   // Save file metadata if repository is available
	   if h.fileMetadataRepo != nil {
	       // ... metadata code
	   }
	*/

	// Return the file URL
	log.Printf("Upload successful, returning URL to client")
	return response.Success(c, map[string]string{
		"url": fileURL,
	})
}

// DeleteFile handles file deletion requests
func (h *FileHandler) DeleteFile(c echo.Context) error {
	// Parse request
	var req struct {
		URL string `json:"url" validate:"required,url"`
	}

	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	// Delete metadata if repository is available
	// if h.fileMetadataRepo != nil {
	// 	metadata, err := h.fileMetadataRepo.GetByURL(c.Request().Context(), req.URL)
	// 	if err == nil && metadata != nil {
	// 		if err := h.fileMetadataRepo.Delete(c.Request().Context(), metadata.ID); err != nil {
	// 			log.Printf("Failed to delete file metadata: %v", err)
	// 			// Continue with deletion anyway
	// 		}
	// 	}
	// }

	// Delete the file
	if err := h.fileService.DeleteFile(c.Request().Context(), req.URL); err != nil {
		return response.Error(c, response.NewError("DELETE_FAILED", fmt.Sprintf("Failed to delete file: %v", err), http.StatusInternalServerError))
	}

	return response.Success(c, map[string]string{
		"message": "File deleted successfully",
	})
}

// Helper functions
func isAllowedFileType(fileType string) bool {
	allowedTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"application/pdf",
	}

	for _, allowedType := range allowedTypes {
		if fileType == allowedType {
			return true
		}
	}

	return false
}

func sanitizeFolderName(folder string) string {
	// Remove any path traversal attempts
	folder = filepath.Base(folder)

	// Allow only alphanumeric, dash and underscore
	validChars := []rune{}
	for _, char := range folder {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			validChars = append(validChars, char)
		}
	}

	// Ensure a valid folder name
	sanitized := string(validChars)
	if sanitized == "" {
		return "uploads"
	}

	return sanitized
}

func (h *FileHandler) UploadProductImage(c echo.Context) error {
	log.Printf("Product image upload requested")

	// First parse the multipart form
	err := c.Request().ParseMultipartForm(h.maxFileSize)
	if err != nil {
		log.Printf("Failed to parse multipart form: %v", err)
		// Continue anyway, the file might be parsable by Echo
	}

	// Initialize form if needed
	if c.Request().Form == nil {
		log.Printf("Request form was nil, initializing")
		c.Request().Form = make(url.Values)
	}

	// Set form values
	c.Request().Form.Set("folder", "product-images")
	c.Request().Form.Set("public", "true")

	log.Printf("Forwarding to main upload handler with folder=product-images")
	return h.UploadFile(c)
}

// UploadProfilePhoto is a convenience method for profile photos
func (h *FileHandler) UploadProfilePhoto(c echo.Context) error {
	// Initialize Form if it's nil
	if c.Request().Form == nil {
		c.Request().ParseMultipartForm(h.maxFileSize)
	}

	// Override folder to profile-photos
	c.Request().Form.Set("folder", "profile-photos")
	c.Request().Form.Set("public", "true")
	c.Request().Form.Set("entityType", "user")

	// Get user ID from context
	if userID, ok := c.Get("uid").(string); ok {
		c.Request().Form.Set("entityId", userID)
	}

	return h.UploadFile(c)
}

// UploadVerificationDocument is a convenience method for verification docs
func (h *FileHandler) UploadVerificationDocument(c echo.Context) error {
	// Initialize Form if it's nil
	if c.Request().Form == nil {
		c.Request().ParseMultipartForm(h.maxFileSize)
	}

	// Override folder to verification and set private
	c.Request().Form.Set("folder", "verification")
	c.Request().Form.Set("public", "false")
	c.Request().Form.Set("entityType", "verification")

	// Get user ID from context
	if userID, ok := c.Get("uid").(string); ok {
		c.Request().Form.Set("entityId", userID)
	}

	return h.UploadFile(c)
}

func (h *FileHandler) UploadAndLinkProductImage(c echo.Context) error {
	// Get product ID from path
	productID := c.Param("productId")
	if productID == "" {
		return response.Error(c, response.NewError("MISSING_PRODUCT_ID", "Product ID is required", http.StatusBadRequest))
	}

	// Get user ID from context
	userID, ok := c.Get("uid").(string)
	if !ok {
		return response.Error(c, response.NewError("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}

	// Get the file from the request
	file, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, response.NewError("INVALID_FILE", "Missing or invalid file", http.StatusBadRequest))
	}

	// Check file size
	if file.Size > h.maxFileSize {
		return response.Error(c, response.NewError("FILE_TOO_LARGE", fmt.Sprintf("File size exceeds maximum allowed (%dMB)", h.maxFileSize/(1024*1024)), http.StatusBadRequest))
	}

	// Validate file type
	fileType := file.Header.Get("Content-Type")
	if !isAllowedFileType(fileType) {
		return response.Error(c, response.NewError("INVALID_FILE_TYPE", "File type not supported", http.StatusBadRequest))
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		return response.Error(c, response.NewError("FILE_READ_ERROR", "Unable to read file", http.StatusInternalServerError))
	}
	defer src.Close()

	// Upload the file to product-images folder
	fileURL, err := h.fileService.UploadFile(c.Request().Context(), src, fileType, "product-images", true)
	if err != nil {
		return response.Error(c, response.NewError("UPLOAD_FAILED", fmt.Sprintf("Failed to upload file: %v", err), http.StatusInternalServerError))
	}

	// We need to add productRepo to FileHandler - will need to inject this
	// Get existing product
	product, err := h.productRepo.GetByID(c.Request().Context(), productID)
	if err != nil {
		return response.Error(c, err)
	}

	// Check ownership
	if product.SellerID != userID {
		return response.Error(c, response.NewError("FORBIDDEN", "You don't have permission to update this product", http.StatusForbidden))
	}

	// Add the new image to product
	displayOrder := 0
	if len(product.Images) > 0 {
		displayOrder = len(product.Images)
	}

	newImage := entity.ProductImage{
		ID:           "img-" + time.Now().Format("20060102150405"),
		URL:          fileURL,
		DisplayOrder: displayOrder,
	}

	product.Images = append(product.Images, newImage)
	product.UpdatedAt = time.Now()

	// Update product in database
	if err := h.productRepo.Update(c.Request().Context(), product); err != nil {
		return response.Error(c, err)
	}

	// Save file metadata
	if h.fileMetadataRepo != nil {
		metadata := &entity.FileMetadata{
			ID:         uuid.New().String(),
			URL:        fileURL,
			EntityType: "product",
			EntityID:   productID,
			UploadedBy: userID,
			IsPublic:   true,
			CreatedAt:  time.Now(),
		}

		if err := h.fileMetadataRepo.Create(c.Request().Context(), metadata); err != nil {
			// Log error but don't fail the request
			log.Printf("Failed to save file metadata: %v", err)
		}
	}

	return response.Success(c, map[string]interface{}{
		"url":     fileURL,
		"product": product,
	})
}
