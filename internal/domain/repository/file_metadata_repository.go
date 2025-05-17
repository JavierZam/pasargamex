package repository

import (
	"context"
	"pasargamex/internal/domain/entity"
)

type FileMetadataRepository interface {
	Create(ctx context.Context, metadata *entity.FileMetadata) error
	GetByID(ctx context.Context, id string) (*entity.FileMetadata, error)
	GetByURL(ctx context.Context, url string) (*entity.FileMetadata, error)
	GetByObjectName(ctx context.Context, objectName string) (*entity.FileMetadata, error)
	GetByEntityID(ctx context.Context, entityType, entityID string) ([]*entity.FileMetadata, error)
	GetByUploader(ctx context.Context, userID string, limit, offset int) ([]*entity.FileMetadata, int64, error)
	Update(ctx context.Context, metadata *entity.FileMetadata) error
	Delete(ctx context.Context, id string) error
}
