package repository

import (
	"context"

	"pasargamex/internal/domain/entity"
)

type ProductRepository interface {
	Create(ctx context.Context, product *entity.Product) error
	GetByID(ctx context.Context, id string) (*entity.Product, error)
	List(ctx context.Context, filter map[string]interface{}, sort string, limit, offset int) ([]*entity.Product, int64, error)
	Update(ctx context.Context, product *entity.Product) error
	Delete(ctx context.Context, id string) error
	SoftDelete(ctx context.Context, id string) error
	IncrementViews(ctx context.Context, id string) error
	ListBySellerID(ctx context.Context, sellerID string, status string, limit, offset int) ([]*entity.Product, int64, error)
	Search(ctx context.Context, query string, filter map[string]interface{}, limit, offset int) ([]*entity.Product, int64, error)
}
