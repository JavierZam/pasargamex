package entity

import (
	"time"
)

type Review struct {
	ID            string    `json:"id" firestore:"id"`
	TransactionID string    `json:"transaction_id" firestore:"transactionId"`
	ProductID     string    `json:"product_id" firestore:"productId"`
	ReviewerID    string    `json:"reviewer_id" firestore:"reviewerId"`
	TargetID      string    `json:"target_id" firestore:"targetId"`
	Type          string    `json:"type" firestore:"type"`
	Rating        int       `json:"rating" firestore:"rating"`
	Content       string    `json:"content" firestore:"content"`
	Images        []string  `json:"images" firestore:"images"`
	Status        string    `json:"status" firestore:"status"`
	ReportCount   int       `json:"report_count" firestore:"reportCount"`
	CreatedAt     time.Time `json:"created_at" firestore:"createdAt"`
	UpdatedAt     time.Time `json:"updated_at" firestore:"updatedAt"`
}

type ReviewReport struct {
	ID          string     `json:"id" firestore:"id"`
	ReviewID    string     `json:"review_id" firestore:"reviewId"`
	ReporterID  string     `json:"reporter_id" firestore:"reporterId"`
	Reason      string     `json:"reason" firestore:"reason"`
	Description string     `json:"description" firestore:"description"`
	Status      string     `json:"status" firestore:"status"`
	CreatedAt   time.Time  `json:"created_at" firestore:"createdAt"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty" firestore:"resolvedAt,omitempty"`
}
