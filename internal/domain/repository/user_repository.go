package repository

import (
	"context"

	"pasargamex/internal/domain/entity"
)

type UserRepository interface {
    Create(ctx context.Context, user *entity.User) error
    GetByID(ctx context.Context, id string) (*entity.User, error)
    GetByEmail(ctx context.Context, email string) (*entity.User, error)
    Update(ctx context.Context, user *entity.User) error
    Delete(ctx context.Context, id string) error
    FindByField(ctx context.Context, field, value string, limit, offset int) ([]*entity.User, int64, error)
}