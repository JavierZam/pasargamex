package repository

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/firestore"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
)

type firestoreUserRepository struct {
	client *firestore.Client
}

func NewFirestoreUserRepository(client *firestore.Client) repository.UserRepository {
	return &firestoreUserRepository{
		client: client,
	}
}

func (r *firestoreUserRepository) Create(ctx context.Context, user *entity.User) error {
	_, err := r.client.Collection("users").Doc(user.ID).Set(ctx, user)
	return err
}

func (r *firestoreUserRepository) GetByID(ctx context.Context, id string) (*entity.User, error) {
	doc, err := r.client.Collection("users").Doc(id).Get(ctx)
	if err != nil {
		return nil, err
	}

	var user entity.User
	if err := doc.DataTo(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *firestoreUserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := r.client.Collection("users").Where("email", "==", email).Limit(1)
	iter := query.Documents(ctx)
	doc, err := iter.Next()
	if err != nil {
		return nil, err
	}

	var user entity.User
	if err := doc.DataTo(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *firestoreUserRepository) Update(ctx context.Context, user *entity.User) error {
    log.Printf("Updating user in Firestore, ID: %s", user.ID)
    
    // Konversi struct User ke map untuk digunakan dengan MergeAll
    updateData := map[string]interface{}{
        "username":  user.Username,
        "phone":     user.Phone,
        "bio":       user.Bio,
        "updatedAt": time.Now(),
    }
    
    log.Printf("Update data: %+v", updateData)
    
    _, err := r.client.Collection("users").Doc(user.ID).Set(ctx, updateData, firestore.MergeAll)
    
    if err != nil {
        log.Printf("Firestore update error: %v", err)
        return err
    }
    
    log.Printf("User updated successfully in Firestore")
    return nil
}

func (r *firestoreUserRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.Collection("users").Doc(id).Delete(ctx)
	return err
}