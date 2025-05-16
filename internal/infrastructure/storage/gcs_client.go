package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

type CloudStorageClient struct {
	client     *storage.Client
	bucketName string
	projectID  string
}

func NewCloudStorageClient(ctx context.Context, bucketName, projectID string, credentialsPath string) (*CloudStorageClient, error) {
	var opts []option.ClientOption
	if credentialsPath != "" {
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
		fmt.Printf("Warning: Failed to set CORS configuration: %v\n", err)
	}

	return storageClient, nil
}

func (c *CloudStorageClient) setBucketCORS(ctx context.Context) error {
	bucket := c.client.Bucket(c.bucketName)

	corsConfig := storage.CORS{
		MaxAge:          3600, // 1 hour
		Methods:         []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		Origins:         []string{"*"}, // Replace with your domains in production
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

func (c *CloudStorageClient) UploadFile(ctx context.Context, file io.Reader, fileType, folder string, isPublic bool) (string, error) {
	if strings.HasPrefix(folder, "public/") || strings.HasPrefix(folder, "private/") {
	} else {
		if isPublic {
			folder = "public/" + folder
		} else {
			folder = "private/" + folder
		}
	}

	filename := fmt.Sprintf("%s/%s-%s", folder, uuid.New().String(), time.Now().Format("20060102150405"))

	switch fileType {
	case "image/jpeg", "image/jpg":
		filename += ".jpg"
	case "image/png":
		filename += ".png"
	case "image/gif":
		filename += ".gif"
	case "application/pdf":
		filename += ".pdf"
	default:
		filename += ".bin"
	}

	obj := c.client.Bucket(c.bucketName).Object(filename)
	wc := obj.NewWriter(ctx)
	wc.ContentType = fileType
	wc.CacheControl = "public, max-age=86400" // 1 day caching

	if _, err := io.Copy(wc, file); err != nil {
		return "", fmt.Errorf("failed to copy file to GCS: %v", err)
	}

	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %v", err)
	}

	if isPublic {
		if err := obj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
			return "", fmt.Errorf("failed to set ACL: %v", err)
		}
	}

	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", c.bucketName, filename), nil
}

func (c *CloudStorageClient) DeleteFile(ctx context.Context, fileURL string) error {
	// Extract the object name from the URL
	// Expected URL format: https://storage.googleapis.com/bucket-name/file-path
	const prefix = "https://storage.googleapis.com/"
	if !strings.HasPrefix(fileURL, prefix) {
		return fmt.Errorf("invalid GCS URL format")
	}

	path := fileURL[len(prefix):]
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] != c.bucketName {
		return fmt.Errorf("invalid GCS URL format or bucket mismatch")
	}

	objectName := parts[1]

	obj := c.client.Bucket(c.bucketName).Object(objectName)
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}

	return nil
}

func (c *CloudStorageClient) GenerateSignedUploadURL(ctx context.Context, fileType, folder string, isPublic bool) (string, error) {
	if isPublic {
		folder = "public/" + folder
	} else {
		folder = "private/" + folder
	}

	filename := fmt.Sprintf("%s/%s-%s", folder, uuid.New().String(), time.Now().Format("20060102150405"))

	switch fileType {
	case "image/jpeg", "image/jpg":
		filename += ".jpg"
	case "image/png":
		filename += ".png"
	case "image/gif":
		filename += ".gif"
	case "application/pdf":
		filename += ".pdf"
	default:
		filename += ".bin"
	}

	opts := &storage.SignedURLOptions{
		Method:      http.MethodPut,
		ContentType: fileType,
		Expires:     time.Now().Add(15 * time.Minute),
	}

	url, err := storage.SignedURL(c.bucketName, filename, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %v", err)
	}

	return url, nil
}

func (c *CloudStorageClient) Close() error {
	return c.client.Close()
}
