package repository

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"

	"github.com/google/uuid"
)

type firestoreReviewRepository struct {
	client *firestore.Client
}

func NewFirestoreReviewRepository(client *firestore.Client) repository.ReviewRepository {
	return &firestoreReviewRepository{
		client: client,
	}
}

// Review Methods
func (r *firestoreReviewRepository) Create(ctx context.Context, review *entity.Review) error {
	if review.ID == "" {
		review.ID = uuid.New().String()
	}
	
	now := time.Now()
	review.CreatedAt = now
	review.UpdatedAt = now
	
	_, err := r.client.Collection("reviews").Doc(review.ID).Set(ctx, review)
	if err != nil {
		return errors.Internal("Failed to create review", err)
	}
	
	return nil
}

func (r *firestoreReviewRepository) GetByID(ctx context.Context, id string) (*entity.Review, error) {
	doc, err := r.client.Collection("reviews").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, errors.NotFound("Review", err)
		}
		return nil, errors.Internal("Failed to get review", err)
	}
	
	var review entity.Review
	if err := doc.DataTo(&review); err != nil {
		return nil, errors.Internal("Failed to parse review data", err)
	}
	
	return &review, nil
}

func (r *firestoreReviewRepository) GetByTransactionID(ctx context.Context, transactionID string) (*entity.Review, error) {
	query := r.client.Collection("reviews").Where("transactionId", "==", transactionID).Limit(1)
	iter := query.Documents(ctx)
	doc, err := iter.Next()
	
	if err != nil {
		if err == iterator.Done {
			return nil, errors.NotFound("Review for transaction", nil)
		}
		return nil, errors.Internal("Failed to query review", err)
	}
	
	var review entity.Review
	if err := doc.DataTo(&review); err != nil {
		return nil, errors.Internal("Failed to parse review data", err)
	}
	
	return &review, nil
}

func (r *firestoreReviewRepository) List(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]*entity.Review, int64, error) {
	// Implementasi mirip seperti repository sebelumnya
	// ...
	// Untuk contoh ini kita skip implementasi lengkapnya
	return []*entity.Review{}, 0, nil
}

func (r *firestoreReviewRepository) Update(ctx context.Context, review *entity.Review) error {
	review.UpdatedAt = time.Now()
	
	_, err := r.client.Collection("reviews").Doc(review.ID).Set(ctx, review)
	if err != nil {
		return errors.Internal("Failed to update review", err)
	}
	
	return nil
}

func (r *firestoreReviewRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.Collection("reviews").Doc(id).Delete(ctx)
	if err != nil {
		return errors.Internal("Failed to delete review", err)
	}
	
	return nil
}

// Report Methods
func (r *firestoreReviewRepository) CreateReport(ctx context.Context, report *entity.ReviewReport) error {
	if report.ID == "" {
		report.ID = uuid.New().String()
	}
	
	now := time.Now()
	report.CreatedAt = now
	
	_, err := r.client.Collection("review_reports").Doc(report.ID).Set(ctx, report)
	if err != nil {
		return errors.Internal("Failed to create review report", err)
	}
	
	return nil
}

func (r *firestoreReviewRepository) GetReportByID(ctx context.Context, id string) (*entity.ReviewReport, error) {
	doc, err := r.client.Collection("review_reports").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, errors.NotFound("Review report", err)
		}
		return nil, errors.Internal("Failed to get review report", err)
	}
	
	var report entity.ReviewReport
	if err := doc.DataTo(&report); err != nil {
		return nil, errors.Internal("Failed to parse review report data", err)
	}
	
	return &report, nil
}

func (r *firestoreReviewRepository) ListReports(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]*entity.ReviewReport, int64, error) {
	// Implementasi mirip seperti repository sebelumnya
	// ...
	// Untuk contoh ini kita skip implementasi lengkapnya
	return []*entity.ReviewReport{}, 0, nil
}

func (r *firestoreReviewRepository) UpdateReport(ctx context.Context, report *entity.ReviewReport) error {
	_, err := r.client.Collection("review_reports").Doc(report.ID).Set(ctx, report)
	if err != nil {
		return errors.Internal("Failed to update review report", err)
	}
	
	return nil
}