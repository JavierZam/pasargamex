package repository

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
)

type firestoreFileMetadataRepository struct {
	client *firestore.Client
}

func NewFirestoreFileMetadataRepository(client *firestore.Client) repository.FileMetadataRepository {
	return &firestoreFileMetadataRepository{
		client: client,
	}
}

func (r *firestoreFileMetadataRepository) Create(ctx context.Context, metadata *entity.FileMetadata) error {
	_, err := r.client.Collection("file_metadata").Doc(metadata.ID).Set(ctx, metadata)
	if err != nil {
		return errors.Internal("Failed to create file metadata", err)
	}
	return nil
}

func (r *firestoreFileMetadataRepository) GetByURL(ctx context.Context, url string) (*entity.FileMetadata, error) {
	iter := r.client.Collection("file_metadata").Where("url", "==", url).Limit(1).Documents(ctx)
	doc, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, errors.NotFound("File metadata", nil)
		}
		return nil, errors.Internal("Failed to query file metadata", err)
	}

	var metadata entity.FileMetadata
	if err := doc.DataTo(&metadata); err != nil {
		return nil, errors.Internal("Failed to parse file metadata", err)
	}

	return &metadata, nil
}

func (r *firestoreFileMetadataRepository) GetByEntityID(ctx context.Context, entityType, entityID string) ([]*entity.FileMetadata, error) {
	iter := r.client.Collection("file_metadata").
		Where("entityType", "==", entityType).
		Where("entityId", "==", entityID).
		Documents(ctx)

	var metadataList []*entity.FileMetadata
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, errors.Internal("Failed to iterate file metadata", err)
		}

		var metadata entity.FileMetadata
		if err := doc.DataTo(&metadata); err != nil {
			return nil, errors.Internal("Failed to parse file metadata", err)
		}
		metadataList = append(metadataList, &metadata)
	}

	return metadataList, nil
}

func (r *firestoreFileMetadataRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.Collection("file_metadata").Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return errors.NotFound("File metadata", err)
		}
		return errors.Internal("Failed to delete file metadata", err)
	}
	return nil
}
