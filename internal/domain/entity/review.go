package entity

import (
	"time"
)

// Review adalah ulasan yang diberikan setelah transaksi
type Review struct {
	ID           string    `json:"id" firestore:"id"`
	TransactionID string    `json:"transaction_id" firestore:"transactionId"`
	ProductID    string    `json:"product_id" firestore:"productId"`
	ReviewerID   string    `json:"reviewer_id" firestore:"reviewerId"`
	TargetID     string    `json:"target_id" firestore:"targetId"`
	Type         string    `json:"type" firestore:"type"` // "seller_review" atau "buyer_review"
	Rating       int       `json:"rating" firestore:"rating"` // 1-5
	Content      string    `json:"content" firestore:"content"`
	Images       []string  `json:"images" firestore:"images"`
	Status       string    `json:"status" firestore:"status"` // "active", "hidden", "reported", "deleted"
	ReportCount  int       `json:"report_count" firestore:"reportCount"`
	CreatedAt    time.Time `json:"created_at" firestore:"createdAt"`
	UpdatedAt    time.Time `json:"updated_at" firestore:"updatedAt"`
}

// ReviewReport adalah laporan terhadap ulasan yang tidak pantas
type ReviewReport struct {
	ID          string     `json:"id" firestore:"id"`
	ReviewID    string     `json:"review_id" firestore:"reviewId"`
	ReporterID  string     `json:"reporter_id" firestore:"reporterId"`
	Reason      string     `json:"reason" firestore:"reason"` // "inappropriate", "spam", "fake", "offensive", "other"
	Description string     `json:"description" firestore:"description"`
	Status      string     `json:"status" firestore:"status"` // "pending", "resolved", "rejected"
	CreatedAt   time.Time  `json:"created_at" firestore:"createdAt"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty" firestore:"resolvedAt,omitempty"`
}