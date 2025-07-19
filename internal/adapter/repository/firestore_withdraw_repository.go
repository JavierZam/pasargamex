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

type firestoreWithdrawRepository struct {
	client *firestore.Client
}

func NewFirestoreWithdrawRepository(client *firestore.Client) repository.WithdrawRepository {
	return &firestoreWithdrawRepository{
		client: client,
	}
}

func (r *firestoreWithdrawRepository) CreateWithdrawRequest(ctx context.Context, withdraw *entity.WithdrawRequest) error {
	_, err := r.client.Collection("withdraw_requests").Doc(withdraw.ID).Set(ctx, withdraw)
	return err
}

func (r *firestoreWithdrawRepository) GetWithdrawRequestByID(ctx context.Context, withdrawID string) (*entity.WithdrawRequest, error) {
	doc, err := r.client.Collection("withdraw_requests").Doc(withdrawID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var withdraw entity.WithdrawRequest
	if err := doc.DataTo(&withdraw); err != nil {
		return nil, err
	}

	return &withdraw, nil
}

func (r *firestoreWithdrawRepository) GetWithdrawRequestsByUserID(ctx context.Context, userID string, pagination *utils.Pagination) ([]entity.WithdrawRequest, error) {
	// Simple query without OrderBy to avoid composite index requirement
	query := r.client.Collection("withdraw_requests").Where("userId", "==", userID)
	
	if pagination.Page > 1 {
		query = query.Offset((pagination.Page - 1) * pagination.Limit)
	}
	query = query.Limit(pagination.Limit)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var withdraws []entity.WithdrawRequest
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating withdraw requests: %v", err)
			return []entity.WithdrawRequest{}, nil
		}

		var withdraw entity.WithdrawRequest
		if err := doc.DataTo(&withdraw); err != nil {
			log.Printf("Error converting document to withdraw: %v", err)
			continue
		}

		withdraws = append(withdraws, withdraw)
	}

	return withdraws, nil
}

func (r *firestoreWithdrawRepository) UpdateWithdrawRequest(ctx context.Context, withdraw *entity.WithdrawRequest) error {
	withdraw.UpdatedAt = time.Now()
	_, err := r.client.Collection("withdraw_requests").Doc(withdraw.ID).Set(ctx, withdraw)
	return err
}

func (r *firestoreWithdrawRepository) GetPendingWithdrawRequests(ctx context.Context, pagination *utils.Pagination) ([]entity.WithdrawRequest, error) {
	// Use simple query for pending status only to avoid composite index
	query := r.client.Collection("withdraw_requests").Where("status", "==", "pending")
	
	if pagination.Page > 1 {
		query = query.Offset((pagination.Page - 1) * pagination.Limit)
	}
	query = query.Limit(pagination.Limit)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var withdraws []entity.WithdrawRequest
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating pending withdraw requests: %v", err)
			return []entity.WithdrawRequest{}, nil
		}

		var withdraw entity.WithdrawRequest
		if err := doc.DataTo(&withdraw); err != nil {
			log.Printf("Error converting document to withdraw: %v", err)
			continue
		}

		withdraws = append(withdraws, withdraw)
	}

	// Also get processing status (separate query to avoid 'in' operator complexity)
	processingQuery := r.client.Collection("withdraw_requests").Where("status", "==", "processing").Limit(pagination.Limit)
	processingIter := processingQuery.Documents(ctx)
	defer processingIter.Stop()

	for {
		doc, err := processingIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating processing withdraw requests: %v", err)
			break
		}

		var withdraw entity.WithdrawRequest
		if err := doc.DataTo(&withdraw); err != nil {
			log.Printf("Error converting document to withdraw: %v", err)
			continue
		}

		withdraws = append(withdraws, withdraw)
	}

	// Always return non-nil slice
	if withdraws == nil {
		withdraws = []entity.WithdrawRequest{}
	}

	return withdraws, nil
}