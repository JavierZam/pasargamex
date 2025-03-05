package repository

import (
	"context"
	"pasargamex/internal/domain/entity"
)

type ReviewRepository interface {
	// Review methods
	Create(ctx context.Context, review *entity.Review) error
	GetByID(ctx context.Context, id string) (*entity.Review, error)
	GetByTransactionID(ctx context.Context, transactionID string) (*entity.Review, error)
	List(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]*entity.Review, int64, error)
	Update(ctx context.Context, review *entity.Review) error
	Delete(ctx context.Context, id string) error
	
	// Report methods
	CreateReport(ctx context.Context, report *entity.ReviewReport) error
	GetReportByID(ctx context.Context, id string) (*entity.ReviewReport, error)
	ListReports(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]*entity.ReviewReport, int64, error)
	UpdateReport(ctx context.Context, report *entity.ReviewReport) error
}