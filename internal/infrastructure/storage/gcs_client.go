package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"google.golang.org/api/option"

	"pasargamex/internal/domain/service"
	"pasargamex/pkg/logger"
)

type CloudStorageClient struct {
	client     *storage.Client
	bucketName string
	projectID  string
}

func NewCloudStorageClient(ctx context.Context, bucketName, projectID string, credentialsPath string) (*CloudStorageClient, error) {
	var opts []option.ClientOption
	
	// Try to get service account from environment variable first (for production)
	if credentialsJSON := os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON"); credentialsJSON != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(credentialsJSON)))
	} else if credentialsPath != "" {
		// Fallback to file path (for local development)
		opts = append(opts, option.WithCredentialsFile(credentialsPath))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %v", err)
	}

	storageClient := &CloudStorageClient{
		client:     client,
		bucketName: bucketName,
		projectID:  projectID,
	}

	if err := storageClient.setBucketCORS(ctx); err != nil {
		logger.Warn("Failed to set CORS configuration: %v", err)
	}

	return storageClient, nil
}

func (c *CloudStorageClient) setBucketCORS(ctx context.Context) error {
	bucket := c.client.Bucket(c.bucketName)

	corsConfig := storage.CORS{
		MaxAge:          3600,
		Methods:         []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		Origins:         []string{"*"},
		ResponseHeaders: []string{"Content-Type", "x-goog-resumable"},
	}

	bucketAttrs, err := bucket.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get bucket attributes: %v", err)
	}

	if len(bucketAttrs.CORS) == 0 {
		bucketUpdate := storage.BucketAttrsToUpdate{
			CORS: []storage.CORS{corsConfig},
		}

		_, err := bucket.Update(ctx, bucketUpdate)
		if err != nil {
			return fmt.Errorf("failed to update bucket CORS: %v", err)
		}
	}

	return nil
}

func sanitizeFilename(filename string) string {

	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")

	if len(filename) > 100 {
		filename = filename[len(filename)-100:]
	}

	return filename
}

func (c *CloudStorageClient) UploadFile(ctx context.Context, file io.Reader, fileType, filename, folder string, isPublic bool) (*service.FileUploadResult, error) {
	logger.Debug("Starting file upload to GCS: type=%s, folder=%s, public=%v", fileType, folder, isPublic)

	if !strings.HasPrefix(folder, "public/") && !strings.HasPrefix(folder, "private/") {
		if isPublic {
			folder = "public/" + folder
		} else {
			folder = "private/" + folder
		}
	}

	safeFilename := sanitizeFilename(filename)

	objectName := fmt.Sprintf("%s/%s-%s", folder, uuid.New().String(), safeFilename)

	logger.Debug("Generated object name: %s", objectName)

	obj := c.client.Bucket(c.bucketName).Object(objectName)
	wc := obj.NewWriter(ctx)
	wc.ContentType = fileType
	wc.CacheControl = "public, max-age=86400"

	written, err := io.Copy(wc, file)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file to GCS: %v", err)
	}

	if err := wc.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %v", err)
	}

	logger.Debug("File uploaded successfully to GCS: %d bytes written", written)

	fileURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", c.bucketName, objectName)

	return &service.FileUploadResult{
		URL:        fileURL,
		ObjectName: objectName,
		Size:       written,
	}, nil
}

func (c *CloudStorageClient) DeleteFile(ctx context.Context, objectName string) error {
	logger.Debug("Deleting file: %s from bucket %s", objectName, c.bucketName)

	obj := c.client.Bucket(c.bucketName).Object(objectName)
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}

	logger.Debug("File successfully deleted")
	return nil
}

func (c *CloudStorageClient) GetFileContent(ctx context.Context, objectName string) (io.ReadCloser, string, int64, error) {
	logger.Debug("Getting file content: %s", objectName)

	obj := c.client.Bucket(c.bucketName).Object(objectName)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to get object attributes: %v", err)
	}

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to open file: %v", err)
	}

	return reader, attrs.ContentType, attrs.Size, nil
}

func (c *CloudStorageClient) Close() error {
	return c.client.Close()
}
