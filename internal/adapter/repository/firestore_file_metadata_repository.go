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
	"pasargamex/pkg/logger"
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
	metadata.UpdatedAt = time.Now()
	_, err := r.client.Collection("file_metadata").Doc(metadata.ID).Set(ctx, metadata)
	if err != nil {
		return errors.Internal("Failed to create file metadata", err)
	}
	return nil
}

func (r *firestoreFileMetadataRepository) GetByID(ctx context.Context, id string) (*entity.FileMetadata, error) {
	doc, err := r.client.Collection("file_metadata").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, errors.NotFound("File metadata", err)
		}
		return nil, errors.Internal("Failed to get file metadata", err)
	}

	var metadata entity.FileMetadata
	if err := doc.DataTo(&metadata); err != nil {
		return nil, errors.Internal("Failed to parse file metadata", err)
	}

	return &metadata, nil
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

func (r *firestoreFileMetadataRepository) GetByObjectName(ctx context.Context, objectName string) (*entity.FileMetadata, error) {
	iter := r.client.Collection("file_metadata").Where("objectName", "==", objectName).Limit(1).Documents(ctx)
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
	query := r.client.Collection("file_metadata").
		Where("entityType", "==", entityType).
		Where("entityId", "==", entityID).
		OrderBy("createdAt", firestore.Desc)

	iter := query.Documents(ctx)
	defer iter.Stop()

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

			logger.Error("Failed to parse file metadata: %v", err)
			continue
		}
		metadataList = append(metadataList, &metadata)
	}

	return metadataList, nil
}

func (r *firestoreFileMetadataRepository) GetByUploader(ctx context.Context, userID string, limit, offset int) ([]*entity.FileMetadata, int64, error) {

	countQuery := r.client.Collection("file_metadata").Where("uploadedBy", "==", userID)
	countDocs, err := countQuery.Documents(ctx).GetAll()
	if err != nil {
		return nil, 0, errors.Internal("Failed to count files", err)
	}
	total := int64(len(countDocs))

	query := r.client.Collection("file_metadata").
		Where("uploadedBy", "==", userID).
		OrderBy("createdAt", firestore.Desc)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	iter := query.Documents(ctx)
	defer iter.Stop()

	var metadataList []*entity.FileMetadata
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, errors.Internal("Failed to iterate file metadata", err)
		}

		var metadata entity.FileMetadata
		if err := doc.DataTo(&metadata); err != nil {

			logger.Error("Failed to parse file metadata: %v", err)
			continue
		}
		metadataList = append(metadataList, &metadata)
	}

	return metadataList, total, nil
}

func (r *firestoreFileMetadataRepository) Update(ctx context.Context, metadata *entity.FileMetadata) error {
	metadata.UpdatedAt = time.Now()
	_, err := r.client.Collection("file_metadata").Doc(metadata.ID).Set(ctx, metadata)
	if err != nil {
		return errors.Internal("Failed to update file metadata", err)
	}
	return nil
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
