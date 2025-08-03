package entity

import "time"

type Chat struct {
	ID            string         `json:"id" firestore:"id"`
	Participants  []string       `json:"participants" firestore:"participants"`
	ProductID     string         `json:"product_id,omitempty" firestore:"productId,omitempty"`
	TransactionID string         `json:"transaction_id,omitempty" firestore:"transactionId,omitempty"`
	Type          string         `json:"type" firestore:"type"` // "direct", "group_transaction", "middleman"
	SellerID      string         `json:"seller_id,omitempty" firestore:"sellerId,omitempty"`     // Seller participant ID
	BuyerID       string         `json:"buyer_id,omitempty" firestore:"buyerId,omitempty"`       // Buyer participant ID  
	MiddlemanID   string         `json:"middleman_id,omitempty" firestore:"middlemanId,omitempty"` // Middleman participant ID
	CreatedAt     time.Time      `json:"created_at" firestore:"createdAt"`
	UpdatedAt     time.Time      `json:"updated_at" firestore:"updatedAt"`
	LastMessageAt time.Time      `json:"last_message_at" firestore:"lastMessageAt"`
	LastMessage   string         `json:"last_message,omitempty" firestore:"lastMessage,omitempty"`
	UnreadCount   map[string]int `json:"unread_count" firestore:"unreadCount"` // Map of userID to unread count
}
