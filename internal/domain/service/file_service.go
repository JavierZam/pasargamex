package service

import (
	"context"
	"io"
)

type FileUploadResult struct {
	URL        string
	ObjectName string
	Size       int64
}

type FileUploadService interface {
	UploadFile(ctx context.Context, file io.Reader, fileType, filename, folder string, isPublic bool) (*FileUploadResult, error)

	DeleteFile(ctx context.Context, objectName string) error

	GetFileContent(ctx context.Context, objectName string) (io.ReadCloser, string, int64, error)

	Close() error
}
