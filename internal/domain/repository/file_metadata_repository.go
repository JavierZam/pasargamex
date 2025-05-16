package repository

import (
	"context"
	"pasargamex/internal/domain/entity"
)

type FileMetadataRepository interface {
	Create(ctx context.Context, metadata *entity.FileMetadata) error
	GetByURL(ctx context.Context, url string) (*entity.FileMetadata, error)
	GetByEntityID(ctx context.Context, entityType, entityID string) ([]*entity.FileMetadata, error)
	Delete(ctx context.Context, id string) error
}
