package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	file, err := c.FormFile("file")
	if err != nil {
		logger.Error("Error getting file from form: %v", err)
		return response.Error(c, errors.BadRequest("Missing or invalid file", err))
	}

	logger.Debug("Received file: %s, size: %d bytes, type: %s", file.Filename, file.Size, file.Header.Get("Content-Type"))

	if file.Size > h.maxFileSize {
		logger.Warn("File too large: %d bytes (max: %d)", file.Size, h.maxFileSize)
		return response.Error(c, errors.BadRequest(fmt.Sprintf("File size exceeds maximum allowed (%dMB)", h.maxFileSize/(1024*1024)), nil))
	}

	fileType := file.Header.Get("Content-Type")
	if !isAllowedFileType(fileType) {
		logger.Warn("Invalid file type: %s", fileType)
		return response.Error(c, errors.BadRequest("File type not supported", nil))
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

	userID := getUserIDFromContext(c)

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
	var errors []string

	// Process each file
	for i, fileHeader := range files {
		logger.Debug("Processing file %d: %s", i+1, fileHeader.Filename)

		// Validate file size
		if fileHeader.Size > h.maxFileSize {
			errors = append(errors, fmt.Sprintf("%s: file too large", fileHeader.Filename))
			continue
		}

		// Validate file type
		fileType := fileHeader.Header.Get("Content-Type")
		if !isAllowedFileType(fileType) {
			errors = append(errors, fmt.Sprintf("%s: invalid file type", fileHeader.Filename))
			continue
		}

		// Open file
		src, err := fileHeader.Open()
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to open", fileHeader.Filename))
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
			errors = append(errors, fmt.Sprintf("%s: upload failed", fileHeader.Filename))
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
	response := map[string]interface{}{
		"uploaded_count": len(uploadedImages),
		"images":         uploadedImages,
	}

	if len(errors) > 0 {
		response["errors"] = errors
		response["error_count"] = len(errors)
	}

	// Return success if at least one file uploaded
	if len(uploadedImages) > 0 {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"data":    response,
		})
	} else {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"errors":  errors,
		})
	}
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
