package entity

import (
	"time"
)

type FileMetadata struct {
	ID         string    `json:"id" firestore:"id"`
	URL        string    `json:"url" firestore:"url"`
	EntityType string    `json:"entity_type" firestore:"entityType"`
	EntityID   string    `json:"entity_id" firestore:"entityId"`
	UploadedBy string    `json:"uploaded_by" firestore:"uploadedBy"`
	IsPublic   bool      `json:"is_public" firestore:"isPublic"`
	CreatedAt  time.Time `json:"created_at" firestore:"createdAt"`
}
