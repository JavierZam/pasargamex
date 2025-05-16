package handler

import (
	"fmt"
	"log"
	"net/http"
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
	maxFileSize      int64
}

func NewFileHandler(fileService service.FileUploadService, fileMetadataRepo repository.FileMetadataRepository) *FileHandler {
	return &FileHandler{
		fileService:      fileService,
		fileMetadataRepo: fileMetadataRepo,
		maxFileSize:      5 * 1024 * 1024,
	}
}

var (
	fileHandler *FileHandler
)

// Update this function to accept both services
func SetupFileHandler(fileService service.FileUploadService, fileMetadataRepo repository.FileMetadataRepository) {
	fileHandler = NewFileHandler(fileService, fileMetadataRepo)
}

func GetFileHandler() *FileHandler {
	return fileHandler
}

// UploadFile handles file upload requests
func (h *FileHandler) UploadFile(c echo.Context) error {
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

	// Get the folder from the request
	folder := c.FormValue("folder")
	if folder == "" {
		// Default to a general uploads folder
		folder = "uploads"
	} else {
		// Sanitize folder name
		folder = sanitizeFolderName(folder)
	}

	// Determine if file should be public
	isPublicStr := c.FormValue("public")
	isPublic := true // Default to public for backward compatibility
	if isPublicStr != "" {
		isPublic, _ = strconv.ParseBool(isPublicStr)
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		return response.Error(c, response.NewError("FILE_READ_ERROR", "Unable to read file", http.StatusInternalServerError))
	}
	defer src.Close()

	// Upload the file
	fileURL, err := h.fileService.UploadFile(c.Request().Context(), src, fileType, folder, isPublic)
	if err != nil {
		return response.Error(c, response.NewError("UPLOAD_FAILED", fmt.Sprintf("Failed to upload file: %v", err), http.StatusInternalServerError))
	}

	// Save file metadata if repository is available
	if h.fileMetadataRepo != nil {
		// Get user ID from context (set by auth middleware)
		userID, ok := c.Get("uid").(string)
		if !ok {
			userID = "anonymous" // Fallback if no user ID is found
		}

		// Get entity type and ID if provided
		entityType := c.FormValue("entityType")
		entityID := c.FormValue("entityId")

		// Create metadata record
		fileMetadata := &entity.FileMetadata{
			ID:         uuid.New().String(),
			URL:        fileURL,
			EntityType: entityType,
			EntityID:   entityID,
			UploadedBy: userID,
			IsPublic:   isPublic,
			CreatedAt:  time.Now(),
		}

		if err := h.fileMetadataRepo.Create(c.Request().Context(), fileMetadata); err != nil {
			// Log the error but don't fail the request
			log.Printf("Failed to save file metadata: %v", err)
		}
	}

	// Return the file URL
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
	if h.fileMetadataRepo != nil {
		metadata, err := h.fileMetadataRepo.GetByURL(c.Request().Context(), req.URL)
		if err == nil && metadata != nil {
			// We found metadata, delete it
			if err := h.fileMetadataRepo.Delete(c.Request().Context(), metadata.ID); err != nil {
				log.Printf("Failed to delete file metadata: %v", err)
				// Continue with deletion anyway
			}
		}
	}

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

// Helper methods for common upload scenarios

// UploadProductImage is a convenience method for product images
func (h *FileHandler) UploadProductImage(c echo.Context) error {
	// Initialize Form if it's nil
	if c.Request().Form == nil {
		c.Request().ParseMultipartForm(h.maxFileSize)
	}

	// Override folder to product-images
	c.Request().Form.Set("folder", "product-images")
	c.Request().Form.Set("public", "true")
	c.Request().Form.Set("entityType", "product")

	// Get product ID if provided
	if productID := c.FormValue("productId"); productID != "" {
		c.Request().Form.Set("entityId", productID)
	}

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
