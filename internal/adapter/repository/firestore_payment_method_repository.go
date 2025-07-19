package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
)

type firestorePaymentMethodRepository struct {
	client *firestore.Client
}

func NewFirestorePaymentMethodRepository(client *firestore.Client) repository.PaymentMethodRepository {
	return &firestorePaymentMethodRepository{
		client: client,
	}
}

func (r *firestorePaymentMethodRepository) CreatePaymentMethod(ctx context.Context, paymentMethod *entity.PaymentMethod) error {
	_, err := r.client.Collection("payment_methods").Doc(paymentMethod.ID).Set(ctx, paymentMethod)
	return err
}

func (r *firestorePaymentMethodRepository) GetPaymentMethodByID(ctx context.Context, paymentMethodID string) (*entity.PaymentMethod, error) {
	doc, err := r.client.Collection("payment_methods").Doc(paymentMethodID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var paymentMethod entity.PaymentMethod
	if err := doc.DataTo(&paymentMethod); err != nil {
		return nil, err
	}

	return &paymentMethod, nil
}

func (r *firestorePaymentMethodRepository) GetPaymentMethodsByUserID(ctx context.Context, userID string) ([]entity.PaymentMethod, error) {
	// Simple query without OrderBy to avoid composite index requirement
	query := r.client.Collection("payment_methods").Where("userId", "==", userID).Where("isActive", "==", true)
	iter := query.Documents(ctx)
	defer iter.Stop()

	var paymentMethods []entity.PaymentMethod
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating payment methods: %v", err)
			return []entity.PaymentMethod{}, nil
		}

		var paymentMethod entity.PaymentMethod
		if err := doc.DataTo(&paymentMethod); err != nil {
			log.Printf("Error converting document to payment method: %v", err)
			continue
		}

		paymentMethods = append(paymentMethods, paymentMethod)
	}

	return paymentMethods, nil
}

func (r *firestorePaymentMethodRepository) UpdatePaymentMethod(ctx context.Context, paymentMethod *entity.PaymentMethod) error {
	paymentMethod.UpdatedAt = time.Now()
	_, err := r.client.Collection("payment_methods").Doc(paymentMethod.ID).Set(ctx, paymentMethod)
	return err
}

func (r *firestorePaymentMethodRepository) DeletePaymentMethod(ctx context.Context, paymentMethodID string) error {
	_, err := r.client.Collection("payment_methods").Doc(paymentMethodID).Update(ctx, []firestore.Update{
		{Path: "isActive", Value: false},
		{Path: "updatedAt", Value: time.Now()},
	})
	return err
}

func (r *firestorePaymentMethodRepository) SetDefaultPaymentMethod(ctx context.Context, userID string, paymentMethodID string) error {
	// If paymentMethodID is empty, just unset all defaults
	if paymentMethodID == "" {
		return r.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			query := r.client.Collection("payment_methods").Where("userId", "==", userID).Where("isDefault", "==", true)
			iter := query.Documents(ctx)
			defer iter.Stop()

			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					return err
				}

				err = tx.Update(doc.Ref, []firestore.Update{
					{Path: "isDefault", Value: false},
					{Path: "updatedAt", Value: time.Now()},
				})
				if err != nil {
					return err
				}
			}
			return nil
		})
	}

	return r.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// First, verify the payment method exists and belongs to the user
		docRef := r.client.Collection("payment_methods").Doc(paymentMethodID)
		doc, err := tx.Get(docRef)
		if err != nil {
			return err
		}

		var paymentMethod entity.PaymentMethod
		if err := doc.DataTo(&paymentMethod); err != nil {
			return err
		}

		// Verify ownership
		if paymentMethod.UserID != userID {
			return fmt.Errorf("payment method does not belong to user")
		}

		// Remove default flag from all user's payment methods
		query := r.client.Collection("payment_methods").Where("userId", "==", userID).Where("isDefault", "==", true)
		iter := query.Documents(ctx)
		defer iter.Stop()

		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}

			err = tx.Update(doc.Ref, []firestore.Update{
				{Path: "isDefault", Value: false},
				{Path: "updatedAt", Value: time.Now()},
			})
			if err != nil {
				return err
			}
		}

		// Set the new default payment method
		return tx.Update(docRef, []firestore.Update{
			{Path: "isDefault", Value: true},
			{Path: "updatedAt", Value: time.Now()},
		})
	})
}