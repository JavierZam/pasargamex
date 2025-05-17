package entity

import (
	"time"
)

type FileMetadata struct {
	ID         string    `json:"id" firestore:"id"`
	URL        string    `json:"url" firestore:"url"`
	ObjectName string    `json:"object_name" firestore:"objectName"`
	EntityType string    `json:"entity_type" firestore:"entityType"`
	EntityID   string    `json:"entity_id" firestore:"entityId"`
	UploadedBy string    `json:"uploaded_by" firestore:"uploadedBy"`
	Filename   string    `json:"filename" firestore:"filename"`
	FileType   string    `json:"file_type" firestore:"fileType"`
	FileSize   int64     `json:"file_size" firestore:"fileSize"`
	IsPublic   bool      `json:"is_public" firestore:"isPublic"`
	CreatedAt  time.Time `json:"created_at" firestore:"createdAt"`
	UpdatedAt  time.Time `json:"updated_at,omitempty" firestore:"updatedAt"`
}
