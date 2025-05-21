package entity

import "time"

type Chat struct {
	ID            string         `json:"id" firestore:"id"`
	Participants  []string       `json:"participants" firestore:"participants"`
	ProductID     string         `json:"product_id,omitempty" firestore:"productId,omitempty"`
	TransactionID string         `json:"transaction_id,omitempty" firestore:"transactionId,omitempty"`
	Type          string         `json:"type" firestore:"type"` // "direct", "middleman"
	CreatedAt     time.Time      `json:"created_at" firestore:"createdAt"`
	UpdatedAt     time.Time      `json:"updated_at" firestore:"updatedAt"`
	LastMessageAt time.Time      `json:"last_message_at" firestore:"lastMessageAt"`
	LastMessage   string         `json:"last_message,omitempty" firestore:"lastMessage,omitempty"`
	UnreadCount   map[string]int `json:"unread_count" firestore:"unreadCount"` // Map of userID to unread count
}
