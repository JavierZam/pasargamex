package entity

import "time"

type Message struct {
	ID            string                 `json:"id" firestore:"id"`
	ChatID        string                 `json:"chat_id" firestore:"chatId"`
	SenderID      string                 `json:"sender_id" firestore:"senderId"`
	Content       string                 `json:"content" firestore:"content"`
	Type          string                 `json:"type" firestore:"type"` // "text", "image", "system", "offer"
	Status        string                 `json:"status" firestore:"status"` // "sent", "delivered", "read"
	Metadata      map[string]interface{} `json:"metadata,omitempty" firestore:"metadata,omitempty"`
	AttachmentURL  string   `json:"attachment_url,omitempty" firestore:"attachmentUrl,omitempty"`   // Deprecated: use AttachmentURLs for multiple images
	AttachmentURLs []string `json:"attachment_urls,omitempty" firestore:"attachmentUrls,omitempty"` // New: Multiple attachments support
	ProductID     string                 `json:"product_id,omitempty" firestore:"productId,omitempty"` // New: ProductID associated with the message
	ReadBy        []string               `json:"read_by" firestore:"readBy"`
	CreatedAt     time.Time              `json:"created_at" firestore:"createdAt"`
}
