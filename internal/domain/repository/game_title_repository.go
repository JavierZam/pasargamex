package repository

import (
	"context"

	"pasargamex/internal/domain/entity"
)

type GameTitleRepository interface {
	Create(ctx context.Context, gameTitle *entity.GameTitle) error
	GetByID(ctx context.Context, id string) (*entity.GameTitle, error)
	GetBySlug(ctx context.Context, slug string) (*entity.GameTitle, error)
	List(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]*entity.GameTitle, int64, error)
	Update(ctx context.Context, gameTitle *entity.GameTitle) error
	Delete(ctx context.Context, id string) error
}
