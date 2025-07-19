package repository

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/utils"
)

type firestoreTopupRepository struct {
	client *firestore.Client
}

func NewFirestoreTopupRepository(client *firestore.Client) repository.TopupRepository {
	return &firestoreTopupRepository{
		client: client,
	}
}

func (r *firestoreTopupRepository) CreateTopupRequest(ctx context.Context, topup *entity.TopupRequest) error {
	_, err := r.client.Collection("topup_requests").Doc(topup.ID).Set(ctx, topup)
	return err
}

func (r *firestoreTopupRepository) GetTopupRequestByID(ctx context.Context, topupID string) (*entity.TopupRequest, error) {
	doc, err := r.client.Collection("topup_requests").Doc(topupID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var topup entity.TopupRequest
	if err := doc.DataTo(&topup); err != nil {
		return nil, err
	}

	return &topup, nil
}

func (r *firestoreTopupRepository) GetTopupRequestsByUserID(ctx context.Context, userID string, pagination *utils.Pagination) ([]entity.TopupRequest, error) {
	// Simple query without OrderBy to avoid composite index requirement
	query := r.client.Collection("topup_requests").Where("userId", "==", userID)
	
	if pagination.Page > 1 {
		query = query.Offset((pagination.Page - 1) * pagination.Limit)
	}
	query = query.Limit(pagination.Limit)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var topups []entity.TopupRequest
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating topup requests: %v", err)
			return []entity.TopupRequest{}, nil
		}

		var topup entity.TopupRequest
		if err := doc.DataTo(&topup); err != nil {
			log.Printf("Error converting document to topup: %v", err)
			continue
		}

		topups = append(topups, topup)
	}

	return topups, nil
}

func (r *firestoreTopupRepository) UpdateTopupRequest(ctx context.Context, topup *entity.TopupRequest) error {
	topup.UpdatedAt = time.Now()
	_, err := r.client.Collection("topup_requests").Doc(topup.ID).Set(ctx, topup)
	return err
}

func (r *firestoreTopupRepository) GetPendingTopupRequests(ctx context.Context, pagination *utils.Pagination) ([]entity.TopupRequest, error) {
	// Simple query without OrderBy to avoid composite index requirement
	query := r.client.Collection("topup_requests").Where("status", "==", "pending")
	
	if pagination.Page > 1 {
		query = query.Offset((pagination.Page - 1) * pagination.Limit)
	}
	query = query.Limit(pagination.Limit)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var topups []entity.TopupRequest
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating pending topup requests: %v", err)
			return []entity.TopupRequest{}, nil
		}

		var topup entity.TopupRequest
		if err := doc.DataTo(&topup); err != nil {
			log.Printf("Error converting document to topup: %v", err)
			continue
		}

		topups = append(topups, topup)
	}

	// Always return non-nil slice
	if topups == nil {
		topups = []entity.TopupRequest{}
	}

	return topups, nil
}