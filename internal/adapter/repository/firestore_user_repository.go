package repository

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

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

	updateData := map[string]interface{}{
		"username":  user.Username,
		"phone":     user.Phone,
		"bio":       user.Bio,
		"updatedAt": time.Now(),

		"fullName":           user.FullName,
		"address":            user.Address,
		"dateOfBirth":        user.DateOfBirth,
		"idNumber":           user.IdNumber,
		"idCardImage":        user.IdCardImage,
		"verificationStatus": user.VerificationStatus,
	}

	cleanUpdateData := make(map[string]interface{})
	for key, value := range updateData {

		if strVal, ok := value.(string); ok && strVal == "" {
			continue
		}

		if timeVal, ok := value.(time.Time); ok && timeVal.IsZero() {
			continue
		}

		cleanUpdateData[key] = value
	}

	log.Printf("Update data: %+v", cleanUpdateData)

	_, err := r.client.Collection("users").Doc(user.ID).Set(ctx, cleanUpdateData, firestore.MergeAll)

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

func (r *firestoreUserRepository) FindByField(ctx context.Context, field, value string, limit, offset int) ([]*entity.User, int64, error) {
	query := r.client.Collection("users").Where(field, "==", value)

	countDocs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, 0, err
	}
	total := int64(len(countDocs))

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	iter := query.Documents(ctx)
	var users []*entity.User

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, err
		}

		var user entity.User
		if err := doc.DataTo(&user); err != nil {
			return nil, 0, err
		}
		users = append(users, &user)
	}

	return users, total, nil
}

func (r *firestoreUserRepository) GetUserByRole(ctx context.Context, role string, limit int) []*entity.User {
	query := r.client.Collection("users").Where("role", "==", role).Limit(limit)

	iter := query.Documents(ctx)
	var users []*entity.User

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error getting user by role: %v", err)
			return []*entity.User{}
		}

		var user entity.User
		if err := doc.DataTo(&user); err != nil {
			log.Printf("Error parsing user data: %v", err)
			continue
		}

		users = append(users, &user)
	}

	return users
}
