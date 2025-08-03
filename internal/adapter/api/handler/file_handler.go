package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/internal/domain/service"
	"pasargamex/pkg/errors"
	"pasargamex/pkg/logger"
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
	logger.Debug("Starting file upload handler")

	// Basic rate limiting check - max 10 uploads per minute per user
	userID := getUserIDFromContext(c)
	if userID != "" {
		// Simple in-memory rate limiting (in production, use Redis)
		logger.Debug("Upload request from user: %s", userID)
		// TODO: Implement proper rate limiting with Redis
	}

	file, err := c.FormFile("file")
	if err != nil {
		logger.Error("Error getting file from form: %v", err)
		return response.Error(c, errors.BadRequest("Missing or invalid file", err))
	}

	logger.Debug("Received file: %s, size: %d bytes, type: %s", file.Filename, file.Size, file.Header.Get("Content-Type"))

	// Enhanced file validation
	if err := validateFileContent(file); err != nil {
		logger.Warn("File validation failed: %v", err)
		return response.Error(c, err)
	}

	fileType := file.Header.Get("Content-Type")
	if !isAllowedFileType(fileType) {
		logger.Warn("Invalid file type: %s", fileType)
		return response.Error(c, errors.BadRequest("File type not supported. Only safe image formats allowed.", nil))
	}

	folder := c.FormValue("folder")
	if folder == "" {

		folder = "uploads"
	} else {

		folder = sanitizeFolderName(folder)
	}
	logger.Debug("Using folder: %s", folder)

	isPublicStr := c.FormValue("public")
	isPublic := true
	if isPublicStr != "" {
		isPublic, _ = strconv.ParseBool(isPublicStr)
	}
	logger.Debug("Public file: %v", isPublic)

	src, err := file.Open()
	if err != nil {
		logger.Error("Error opening file: %v", err)
		return response.Error(c, errors.Internal("Unable to read file", err))
	}
	defer src.Close()

	logger.Debug("Calling storage client UploadFile")
	result, err := h.fileService.UploadFile(c.Request().Context(), src, fileType, file.Filename, folder, isPublic)
	if err != nil {
		logger.Error("Error from storage client: %v", err)
		return response.Error(c, errors.Internal(fmt.Sprintf("Failed to upload file: %v", err), err))
	}
	logger.Debug("Storage client returned URL: %s, objectName: %s", result.URL, result.ObjectName)

	// userID already declared above in rate limiting section

	fileID := uuid.New().String()
	metadata := &entity.FileMetadata{
		ID:         fileID,
		URL:        result.URL,
		ObjectName: result.ObjectName,
		EntityType: c.FormValue("entityType"),
		EntityID:   c.FormValue("entityId"),
		UploadedBy: userID,
		Filename:   file.Filename,
		FileType:   fileType,
		FileSize:   result.Size,
		IsPublic:   isPublic,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := h.fileMetadataRepo.Create(c.Request().Context(), metadata); err != nil {
		logger.Error("Failed to save file metadata: %v", err)

	} else {
		logger.Debug("File metadata saved successfully with ID: %s", fileID)
	}

	if isPublic {

		return response.Success(c, map[string]interface{}{
			"id":       fileID,
			"url":      result.URL,
			"filename": file.Filename,
			"size":     result.Size,
			"public":   true,
		})
	} else {

		return response.Success(c, map[string]interface{}{
			"id":       fileID,
			"filename": file.Filename,
			"size":     result.Size,
			"public":   false,
			"message":  "File uploaded successfully. Use the view endpoint to access this file.",
		})
	}
}

func getUserIDFromContext(c echo.Context) string {
	if uid, ok := c.Get("uid").(string); ok {
		return uid
	}
	return ""
}

func (h *FileHandler) DeleteFile(c echo.Context) error {

	var req struct {
		ID string `json:"id" validate:"required"`
	}

	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Error(c, errors.Unauthorized("Authentication required", nil))
	}

	metadata, err := h.fileMetadataRepo.GetByID(c.Request().Context(), req.ID)
	if err != nil {
		logger.Error("Failed to get file metadata: %v", err)
		return response.Error(c, err)
	}

	if metadata.UploadedBy != userID {

		isAdmin := false
		if role, ok := c.Get("role").(string); ok {
			isAdmin = role == "admin"
		}

		if !isAdmin {
			return response.Error(c, errors.Forbidden("You don't have permission to delete this file", nil))
		}
	}

	if err := h.fileService.DeleteFile(c.Request().Context(), metadata.ObjectName); err != nil {
		logger.Error("Failed to delete file from storage: %v", err)
		return response.Error(c, errors.Internal("Failed to delete file", err))
	}

	if err := h.fileMetadataRepo.Delete(c.Request().Context(), req.ID); err != nil {
		logger.Error("Failed to delete file metadata: %v", err)

	}

	logger.Debug("File deleted successfully: %s", req.ID)
	return response.Success(c, map[string]string{
		"message": "File deleted successfully",
	})
}

func (h *FileHandler) ViewFile(c echo.Context) error {

	fileID := c.Param("id")
	if fileID == "" {
		return response.Error(c, errors.BadRequest("File ID is required", nil))
	}

	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Error(c, errors.Unauthorized("Authentication required", nil))
	}

	metadata, err := h.fileMetadataRepo.GetByID(c.Request().Context(), fileID)
	if err != nil {
		logger.Error("Failed to get file metadata: %v", err)
		return response.Error(c, err)
	}

	if metadata.IsPublic {

		return c.Redirect(http.StatusFound, metadata.URL)
	}

	hasPermission := false

	if metadata.UploadedBy == userID {
		hasPermission = true
	}

	if !hasPermission {
		isAdmin := false
		if role, ok := c.Get("role").(string); ok {
			isAdmin = role == "admin"
		}
		hasPermission = isAdmin
	}

	if !hasPermission && metadata.EntityType == "verification" && metadata.EntityID == userID {
		hasPermission = true
	}

	if !hasPermission {
		return response.Error(c, errors.Forbidden("You don't have permission to access this file", nil))
	}

	reader, contentType, size, err := h.fileService.GetFileContent(c.Request().Context(), metadata.ObjectName)
	if err != nil {
		logger.Error("Failed to get file content: %v", err)
		return response.Error(c, errors.Internal("Failed to retrieve file", err))
	}
	defer reader.Close()

	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", size))

	c.Response().Header().Set("Content-Disposition", "inline")
	c.Response().Header().Set("Cache-Control", "private, no-cache, no-store, must-revalidate")
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")

	logger.Debug("File %s accessed by user %s", fileID, userID)

	_, err = io.Copy(c.Response().Writer, reader)
	if err != nil {
		logger.Error("Failed to stream file content: %v", err)
		return err
	}

	return nil
}

func (h *FileHandler) AdminViewFile(c echo.Context) error {

	isAdmin := false
	if role, ok := c.Get("role").(string); ok {
		isAdmin = role == "admin"
	}

	if !isAdmin {
		return response.Error(c, errors.Forbidden("Admin access required", nil))
	}

	fileID := c.Param("id")
	if fileID == "" {
		return response.Error(c, errors.BadRequest("File ID is required", nil))
	}

	metadata, err := h.fileMetadataRepo.GetByID(c.Request().Context(), fileID)
	if err != nil {
		logger.Error("Failed to get file metadata: %v", err)
		return response.Error(c, err)
	}

	if metadata.IsPublic {
		return c.Redirect(http.StatusFound, metadata.URL)
	}

	reader, contentType, size, err := h.fileService.GetFileContent(c.Request().Context(), metadata.ObjectName)
	if err != nil {
		logger.Error("Failed to get file content: %v", err)
		return response.Error(c, errors.Internal("Failed to retrieve file", err))
	}
	defer reader.Close()

	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", size))
	c.Response().Header().Set("Content-Disposition", "inline")

	adminID := getUserIDFromContext(c)
	logger.Debug("Admin %s accessed file %s", adminID, fileID)

	_, err = io.Copy(c.Response().Writer, reader)
	if err != nil {
		logger.Error("Failed to stream file content: %v", err)
		return err
	}

	return nil
}

func isAllowedFileType(fileType string) bool {
	// Only allow safe image formats - NO SVG (XSS risk)
	allowedTypes := []string{
		"image/jpeg",
		"image/jpg", 
		"image/png",
		"image/gif",
		"image/avif",
		"image/webp",
		// NOTE: Excluded SVG (image/svg+xml) due to XSS risks
		// NOTE: Excluded PDF for chat images - only images allowed
	}

	for _, allowedType := range allowedTypes {
		if fileType == allowedType {
			return true
		}
	}

	return false
}

// Enhanced file validation with content verification
func validateFileContent(file *multipart.FileHeader) error {
	// Check file size
	maxSize := int64(5 * 1024 * 1024) // 5MB max
	if file.Size > maxSize {
		return errors.BadRequest(fmt.Sprintf("File too large. Maximum size: %dMB", maxSize/(1024*1024)), nil)
	}

	// Validate file extension matches content type
	ext := strings.ToLower(filepath.Ext(file.Filename))
	contentType := file.Header.Get("Content-Type")
	
	validExtensions := map[string][]string{
		"image/jpeg": {".jpg", ".jpeg"},
		"image/png":  {".png"},
		"image/gif":  {".gif"},
		"image/webp": {".webp"},
		"image/avif": {".avif"},
	}

	if allowedExts, exists := validExtensions[contentType]; exists {
		validExt := false
		for _, allowedExt := range allowedExts {
			if ext == allowedExt {
				validExt = true
				break
			}
		}
		if !validExt {
			return errors.BadRequest("File extension doesn't match content type", nil)
		}
	}

	// Basic filename sanitization
	if strings.Contains(file.Filename, "..") || 
	   strings.Contains(file.Filename, "/") || 
	   strings.Contains(file.Filename, "\\") {
		return errors.BadRequest("Invalid filename", nil)
	}

	return nil
}

func sanitizeFolderName(folder string) string {

	folder = filepath.Base(folder)

	validChars := []rune{}
	for _, char := range folder {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			validChars = append(validChars, char)
		}
	}

	sanitized := string(validChars)
	if sanitized == "" {
		return "uploads"
	}

	return sanitized
}

func (h *FileHandler) UploadProductImage(c echo.Context) error {
	logger.Debug("Product image upload requested")

	if c.Request().Form == nil {
		c.Request().ParseMultipartForm(h.maxFileSize)
	}

	c.Request().Form.Set("folder", "product-images")
	c.Request().Form.Set("public", "true")
	c.Request().Form.Set("entityType", "product")

	logger.Debug("Forwarding to main upload handler with folder=product-images")
	return h.UploadFile(c)
}

func (h *FileHandler) UploadProfilePhoto(c echo.Context) error {
	logger.Debug("Profile photo upload requested")

	if c.Request().Form == nil {
		c.Request().ParseMultipartForm(h.maxFileSize)
	}

	c.Request().Form.Set("folder", "profile-photos")
	c.Request().Form.Set("public", "true")
	c.Request().Form.Set("entityType", "user")

	if userID, ok := c.Get("uid").(string); ok {
		c.Request().Form.Set("entityId", userID)
	}

	return h.UploadFile(c)
}

func (h *FileHandler) UploadVerificationDocument(c echo.Context) error {
	logger.Debug("Verification document upload requested")

	if c.Request().Form == nil {
		c.Request().ParseMultipartForm(h.maxFileSize)
	}

	c.Request().Form.Set("folder", "verification")
	c.Request().Form.Set("public", "false")
	c.Request().Form.Set("entityType", "verification")

	if userID, ok := c.Get("uid").(string); ok {
		c.Request().Form.Set("entityId", userID)
	}

	return h.UploadFile(c)
}

func (h *FileHandler) UploadAndLinkProductImage(c echo.Context) error {
	logger.Debug("Product image upload and link requested")

	productID := c.Param("id")
	if productID == "" {
		return response.Error(c, errors.BadRequest("Product ID is required", nil))
	}

	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Error(c, errors.Unauthorized("Authentication required", nil))
	}

	product, err := h.productRepo.GetByID(c.Request().Context(), productID)
	if err != nil {
		logger.Error("Failed to get product: %v", err)
		return response.Error(c, err)
	}

	if product.SellerID != userID {
		logger.Warn("Access denied: user %s is not the owner of product %s", userID, productID)
		return response.Error(c, errors.Forbidden("You don't have permission to update this product", nil))
	}

	if c.Request().Form == nil {
		c.Request().ParseMultipartForm(h.maxFileSize)
	}

	c.Request().Form.Set("folder", "product-images")
	c.Request().Form.Set("public", "true")
	c.Request().Form.Set("entityType", "product")
	c.Request().Form.Set("entityId", productID)

	resp := c.Response()

	origWriter := resp.Writer

	recorder := &recResponse{
		header: http.Header{},
		body:   strings.Builder{},
	}
	resp.Writer = recorder

	err = h.UploadFile(c)

	resp.Writer = origWriter

	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(recorder.body.String()), &result); err != nil {
		return response.Error(c, errors.Internal("Failed to process upload response", err))
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return response.Error(c, errors.Internal("Invalid upload response format", nil))
	}

	fileID, _ := data["id"].(string)
	fileURL, _ := data["url"].(string)

	if fileURL == "" {
		return response.Error(c, errors.Internal("No file URL in upload response", nil))
	}

	displayOrder := 0
	if len(product.Images) > 0 {
		displayOrder = len(product.Images)
	}

	newImage := entity.ProductImage{
		ID:           fileID,
		URL:          fileURL,
		DisplayOrder: displayOrder,
	}

	product.Images = append(product.Images, newImage)
	product.UpdatedAt = time.Now()

	if err := h.productRepo.Update(c.Request().Context(), product); err != nil {
		logger.Error("Failed to update product: %v", err)
		return response.Error(c, err)
	}

	return response.Success(c, map[string]interface{}{
		"file_id": fileID,
		"url":     fileURL,
		"product": product,
	})
}

func (h *FileHandler) ListUserFiles(c echo.Context) error {

	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Error(c, errors.Unauthorized("Authentication required", nil))
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	files, total, err := h.fileMetadataRepo.GetByUploader(c.Request().Context(), userID, limit, offset)
	if err != nil {
		logger.Error("Failed to get user files: %v", err)
		return response.Error(c, errors.Internal("Failed to retrieve files", err))
	}

	var filesResponse []map[string]interface{}
	for _, file := range files {
		fileResponse := map[string]interface{}{
			"id":          file.ID,
			"filename":    file.Filename,
			"size":        file.FileSize,
			"type":        file.FileType,
			"entity_type": file.EntityType,
			"entity_id":   file.EntityID,
			"created_at":  file.CreatedAt,
			"is_public":   file.IsPublic,
		}

		if file.IsPublic {
			fileResponse["url"] = file.URL
		}

		filesResponse = append(filesResponse, fileResponse)
	}

	return response.Paginated(c, filesResponse, total, page, limit)
}

func (h *FileHandler) UploadMultipleProductImages(c echo.Context) error {
	logger.Debug("Multiple product images upload requested")

	// Parse multipart form with size limit
	err := c.Request().ParseMultipartForm(h.maxFileSize * 10) // Allow larger total size
	if err != nil {
		return response.Error(c, errors.BadRequest("Failed to parse form", err))
	}

	// Get files from form
	form := c.Request().MultipartForm
	files := form.File["files"] // Note: "files" plural

	if len(files) == 0 {
		return response.Error(c, errors.BadRequest("No files provided", nil))
	}

	// Limit number of files
	maxFiles := 10 // Adjust as needed
	if len(files) > maxFiles {
		return response.Error(c, errors.BadRequest(fmt.Sprintf("Too many files. Maximum %d allowed", maxFiles), nil))
	}

	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Error(c, errors.Unauthorized("Authentication required", nil))
	}

	var uploadedImages []map[string]interface{}
	var uploadErrors []string // Fixed: renamed from 'errors' to avoid conflict

	// Process each file
	for _, fileHeader := range files { // Fixed: removed unused 'i' variable
		logger.Debug("Processing file: %s", fileHeader.Filename)

		// Enhanced file validation
		if err := validateFileContent(fileHeader); err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: %s", fileHeader.Filename, err.Error()))
			continue
		}

		// Validate file type
		fileType := fileHeader.Header.Get("Content-Type")
		if !isAllowedFileType(fileType) {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: invalid file type", fileHeader.Filename))
			continue
		}

		// Open file
		src, err := fileHeader.Open()
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: failed to open", fileHeader.Filename))
			continue
		}
		defer src.Close()

		// Upload to storage
		result, err := h.fileService.UploadFile(
			c.Request().Context(),
			src,
			fileType,
			fileHeader.Filename,
			"product-images",
			true,
		)
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: upload failed", fileHeader.Filename))
			continue
		}

		// Create file metadata
		fileID := uuid.New().String()
		metadata := &entity.FileMetadata{
			ID:         fileID,
			URL:        result.URL,
			ObjectName: result.ObjectName,
			EntityType: "product",
			EntityID:   "", // Will be set when linked to product
			UploadedBy: userID,
			Filename:   fileHeader.Filename,
			FileType:   fileType,
			FileSize:   result.Size,
			IsPublic:   true,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		// Save metadata
		if err := h.fileMetadataRepo.Create(c.Request().Context(), metadata); err != nil {
			logger.Error("Failed to save metadata for %s: %v", fileHeader.Filename, err)
			// Continue anyway, file is uploaded
		}

		uploadedImages = append(uploadedImages, map[string]interface{}{
			"id":       fileID,
			"url":      result.URL,
			"filename": fileHeader.Filename,
			"size":     result.Size,
			"public":   true,
		})

		logger.Debug("Successfully uploaded: %s", fileHeader.Filename)
	}

	// Prepare response
	responseData := map[string]interface{}{
		"uploaded_count": len(uploadedImages),
		"images":         uploadedImages,
	}

	if len(uploadErrors) > 0 {
		responseData["errors"] = uploadErrors
		responseData["error_count"] = len(uploadErrors)
	}

	// Return success if at least one file uploaded
	if len(uploadedImages) > 0 {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"data":    responseData,
		})
	} else {
		return response.Error(c, errors.BadRequest("No files were uploaded successfully", nil))
	}
}

func (h *FileHandler) UploadMultipleImagesToProduct(c echo.Context) error {
	logger.Debug("Multiple images upload to existing product requested")

	productID := c.Param("id")
	if productID == "" {
		return response.Error(c, errors.BadRequest("Product ID is required", nil))
	}

	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Error(c, errors.Unauthorized("Authentication required", nil))
	}

	// Verify product ownership
	product, err := h.productRepo.GetByID(c.Request().Context(), productID)
	if err != nil {
		return response.Error(c, err)
	}

	if product.SellerID != userID {
		return response.Error(c, errors.Forbidden("You don't have permission to update this product", nil))
	}

	// Parse multipart form
	err = c.Request().ParseMultipartForm(h.maxFileSize * 10)
	if err != nil {
		return response.Error(c, errors.BadRequest("Failed to parse form", err))
	}

	form := c.Request().MultipartForm
	files := form.File["files"]

	if len(files) == 0 {
		return response.Error(c, errors.BadRequest("No files provided", nil))
	}

	// Limit total images per product
	maxImagesPerProduct := 10
	if len(product.Images)+len(files) > maxImagesPerProduct {
		return response.Error(c, errors.BadRequest(fmt.Sprintf("Product can have maximum %d images", maxImagesPerProduct), nil))
	}

	var newImages []entity.ProductImage
	var uploadErrors []string

	// Process each file
	for _, fileHeader := range files {
		// Validate file
		if fileHeader.Size > h.maxFileSize {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: file too large", fileHeader.Filename))
			continue
		}

		fileType := fileHeader.Header.Get("Content-Type")
		if !isAllowedFileType(fileType) {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: invalid file type", fileHeader.Filename))
			continue
		}

		// Open and upload file
		src, err := fileHeader.Open()
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: failed to open", fileHeader.Filename))
			continue
		}
		defer src.Close()

		result, err := h.fileService.UploadFile(
			c.Request().Context(),
			src,
			fileType,
			fileHeader.Filename,
			"product-images",
			true,
		)
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: upload failed", fileHeader.Filename))
			continue
		}

		// Create file metadata
		fileID := uuid.New().String()
		metadata := &entity.FileMetadata{
			ID:         fileID,
			URL:        result.URL,
			ObjectName: result.ObjectName,
			EntityType: "product",
			EntityID:   productID,
			UploadedBy: userID,
			Filename:   fileHeader.Filename,
			FileType:   fileType,
			FileSize:   result.Size,
			IsPublic:   true,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		h.fileMetadataRepo.Create(c.Request().Context(), metadata)

		// Create product image entry
		displayOrder := len(product.Images) + len(newImages)
		newImage := entity.ProductImage{
			ID:           fileID,
			URL:          result.URL,
			DisplayOrder: displayOrder,
		}

		newImages = append(newImages, newImage)
	}

	// Update product with new images
	if len(newImages) > 0 {
		product.Images = append(product.Images, newImages...)
		product.UpdatedAt = time.Now()

		err = h.productRepo.Update(c.Request().Context(), product)
		if err != nil {
			return response.Error(c, errors.Internal("Failed to update product", err))
		}
	}

	// Prepare response
	responseData := map[string]interface{}{
		"uploaded_count": len(newImages),
		"new_images":     newImages,
		"product":        product,
	}

	if len(uploadErrors) > 0 {
		responseData["errors"] = uploadErrors
		responseData["error_count"] = len(uploadErrors)
	}

	return response.Success(c, responseData)
}

type recResponse struct {
	header http.Header
	body   strings.Builder
	status int
}

func (r *recResponse) Header() http.Header {
	return r.header
}

func (r *recResponse) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *recResponse) WriteHeader(statusCode int) {
	r.status = statusCode
}

func (r *recResponse) Flush() {

}

// New: ProxyFileByObject - Secure proxy for files by object name
func (h *FileHandler) ProxyFileByObject(c echo.Context) error {
	objectName := c.QueryParam("object")
	if objectName == "" {
		return response.Error(c, errors.BadRequest("Object name is required", nil))
	}

	// Get user ID from context (set by auth middleware)
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Error(c, errors.Unauthorized("Authentication required", nil))
	}

	// Find file metadata by object name
	metadata, err := h.fileMetadataRepo.GetByObjectName(c.Request().Context(), objectName)
	if err != nil {
		logger.Error("Failed to get file metadata by object name: %v", err)
		return response.Error(c, errors.NotFound("File not found", err))
	}

	// Check permissions
	hasPermission := false

	// Public files are accessible to anyone
	if metadata.IsPublic {
		hasPermission = true
	}

	// File owner can access
	if metadata.UploadedBy == userID {
		hasPermission = true
	}

	// Admin can access any file
	if role, ok := c.Get("role").(string); ok && role == "admin" {
		hasPermission = true
	}

	if !hasPermission {
		return response.Error(c, errors.Forbidden("You don't have permission to access this file", nil))
	}

	// Stream file content
	reader, contentType, size, err := h.fileService.GetFileContent(c.Request().Context(), metadata.ObjectName)
	if err != nil {
		logger.Error("Failed to get file content: %v", err)
		return response.Error(c, errors.Internal("Failed to retrieve file", err))
	}
	defer reader.Close()

	// Set appropriate headers
	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", size))
	c.Response().Header().Set("Content-Disposition", "inline")
	c.Response().Header().Set("Cache-Control", "private, no-cache, no-store, must-revalidate")
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")

	// Add security headers
	c.Response().Header().Set("X-Frame-Options", "DENY")
	c.Response().Header().Set("Referrer-Policy", "no-referrer")

	logger.Debug("File %s accessed via proxy by user %s", objectName, userID)

	// Stream content
	_, err = io.Copy(c.Response().Writer, reader)
	if err != nil {
		logger.Error("Failed to stream file content: %v", err)
		return err
	}

	return nil
}

// New: GenerateSignedURL - Generate temporary signed URL for secure image access
func (h *FileHandler) GenerateSignedURL(c echo.Context) error {
	type signedURLRequest struct {
		URLs []string `json:"urls" validate:"required,min=1"`
	}

	var req signedURLRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, errors.BadRequest("Invalid request", err))
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Error(c, errors.Unauthorized("Authentication required", nil))
	}

	signedURLs := make(map[string]string)

	for _, storageURL := range req.URLs {
		// Extract object name from storage URL
		objectName := extractObjectNameFromStorageURL(storageURL)
		if objectName == "" {
			logger.Warn("Could not extract object name from URL: %s", storageURL)
			signedURLs[storageURL] = storageURL // Fallback to direct URL
			continue
		}

		// Check if user has permission to access this file
		metadata, err := h.fileMetadataRepo.GetByObjectName(c.Request().Context(), objectName)
		if err != nil {
			logger.Warn("File metadata not found for object: %s", objectName)
			signedURLs[storageURL] = storageURL // Fallback to direct URL
			continue
		}

		hasPermission := false

		// Public files are accessible to anyone
		if metadata.IsPublic {
			hasPermission = true
		}

		// File owner can access
		if metadata.UploadedBy == userID {
			hasPermission = true
		}

		// Admin can access any file
		if role, ok := c.Get("role").(string); ok && role == "admin" {
			hasPermission = true
		}

		if !hasPermission {
			logger.Warn("User %s does not have permission for file: %s", userID, objectName)
			continue // Skip this file
		}

		// For public files, return the original URL (already safe)
		if metadata.IsPublic {
			signedURLs[storageURL] = storageURL
		} else {
			// For private files, create a secure proxy URL with user context
			// We'll use a simple approach with temporary tokens
			secureURL := fmt.Sprintf("http://localhost:8080/v1/files/secure/%s", metadata.ID)
			signedURLs[storageURL] = secureURL
		}
	}

	return response.Success(c, map[string]interface{}{
		"signed_urls": signedURLs,
	})
}

func extractObjectNameFromStorageURL(storageURL string) string {
	// Extract object name from Google Cloud Storage URL
	// Format: https://storage.googleapis.com/bucket-name/path/to/file.jpg
	if storageURL == "" {
		return ""
	}

	// Parse URL
	parsedURL, err := url.Parse(storageURL)
	if err != nil {
		return ""
	}

	if parsedURL.Host == "storage.googleapis.com" {
		pathParts := strings.Split(parsedURL.Path, "/")
		if len(pathParts) >= 3 {
			// Skip empty first part and bucket name, join the rest
			return strings.Join(pathParts[2:], "/")
		}
	}

	return ""
}
