package service

import (
	"context"
	"io"
)

type FileUploadService interface {
	UploadFile(ctx context.Context, file io.Reader, fileType, folder string, isPublic bool) (string, error)
	DeleteFile(ctx context.Context, fileURL string) error
	GenerateSignedUploadURL(ctx context.Context, fileType, folder string, isPublic bool) (string, error)
	Close() error
}
