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
)

type firestoreGameTitleRepository struct {
	client *firestore.Client
}

func NewFirestoreGameTitleRepository(client *firestore.Client) repository.GameTitleRepository {
	return &firestoreGameTitleRepository{
		client: client,
	}
}

func (r *firestoreGameTitleRepository) Create(ctx context.Context, gameTitle *entity.GameTitle) error {
	// Generate ID if not provided
	if gameTitle.ID == "" {
		doc := r.client.Collection("game_titles").NewDoc()
		gameTitle.ID = doc.ID
	}

	// Set timestamps
	now := time.Now()
	if gameTitle.CreatedAt.IsZero() {
		gameTitle.CreatedAt = now
	}
	gameTitle.UpdatedAt = now

	// Save to Firestore
	_, err := r.client.Collection("game_titles").Doc(gameTitle.ID).Set(ctx, gameTitle)
	if err != nil {
		return errors.Internal("Failed to create game title", err)
	}

	return nil
}

func (r *firestoreGameTitleRepository) GetByID(ctx context.Context, id string) (*entity.GameTitle, error) {
	doc, err := r.client.Collection("game_titles").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, errors.NotFound("Game Title", err)
		}
		return nil, errors.Internal("Failed to get game title", err)
	}

	var gameTitle entity.GameTitle
	if err := doc.DataTo(&gameTitle); err != nil {
		return nil, errors.Internal("Failed to parse game title data", err)
	}

	return &gameTitle, nil
}

func (r *firestoreGameTitleRepository) GetBySlug(ctx context.Context, slug string) (*entity.GameTitle, error) {
	query := r.client.Collection("game_titles").Where("slug", "==", slug).Limit(1)
	iter := query.Documents(ctx)
	doc, err := iter.Next()
	
	if err != nil {
		if err == iterator.Done {
			return nil, errors.NotFound("Game Title", nil)
		}
		return nil, errors.Internal("Failed to query game title", err)
	}

	var gameTitle entity.GameTitle
	if err := doc.DataTo(&gameTitle); err != nil {
		return nil, errors.Internal("Failed to parse game title data", err)
	}

	return &gameTitle, nil
}

func (r *firestoreGameTitleRepository) List(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]*entity.GameTitle, int64, error) {
	query := r.client.Collection("game_titles").OrderBy("name", firestore.Asc)

	// Apply filters
	for key, value := range filter {
		query = query.Where(key, "==", value)
	}

	// Get total count (this is expensive in Firestore but necessary for pagination)
	allDocs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, 0, errors.Internal("Failed to count game titles", err)
	}
	total := int64(len(allDocs))

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// Execute query
	iter := query.Documents(ctx)
	var gameTitles []*entity.GameTitle

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, errors.Internal("Failed to iterate game titles", err)
		}

		var gameTitle entity.GameTitle
		if err := doc.DataTo(&gameTitle); err != nil {
			return nil, 0, errors.Internal("Failed to parse game title data", err)
		}
		gameTitles = append(gameTitles, &gameTitle)
	}

	return gameTitles, total, nil
}

func (r *firestoreGameTitleRepository) Update(ctx context.Context, gameTitle *entity.GameTitle) error {
	gameTitle.UpdatedAt = time.Now()

	_, err := r.client.Collection("game_titles").Doc(gameTitle.ID).Set(ctx, gameTitle)
	if err != nil {
		return errors.Internal("Failed to update game title", err)
	}

	return nil
}

func (r *firestoreGameTitleRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.Collection("game_titles").Doc(id).Delete(ctx)
	if err != nil {
		return errors.Internal("Failed to delete game title", err)
	}

	return nil
}