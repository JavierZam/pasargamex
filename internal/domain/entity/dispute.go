package entity

import (
	"time"
)

type Dispute struct {
	ID            string    `json:"id" firestore:"id"`
	TransactionID string    `json:"transaction_id" firestore:"transactionId"`
	ProductID     string    `json:"product_id" firestore:"productId"`
	ReporterID    string    `json:"reporter_id" firestore:"reporterId"`
	ReporterRole  string    `json:"reporter_role" firestore:"reporterRole"` // buyer, seller
	RespondentID  string    `json:"respondent_id" firestore:"respondentId"`
	
	// Dispute Details
	Category      string `json:"category" firestore:"category"`           // credential_invalid, account_recovered, fraud, other
	Subject       string `json:"subject" firestore:"subject"`
	Description   string `json:"description" firestore:"description"`
	Priority      string `json:"priority" firestore:"priority"`           // low, medium, high, critical
	
	// Evidence
	Evidence      []DisputeEvidence `json:"evidence" firestore:"evidence"`
	
	// Status Management
	Status        string `json:"status" firestore:"status"`               // pending, investigating, resolved, closed, escalated
	Resolution    string `json:"resolution,omitempty" firestore:"resolution,omitempty"` // refund, replacement, partial_refund, dismissed
	
	// Assignment
	AssignedAdminID string    `json:"assigned_admin_id,omitempty" firestore:"assignedAdminId,omitempty"`
	AssignedAt      *time.Time `json:"assigned_at,omitempty" firestore:"assignedAt,omitempty"`
	
	// Timeline
	CreatedAt     time.Time  `json:"created_at" firestore:"createdAt"`
	UpdatedAt     time.Time  `json:"updated_at" firestore:"updatedAt"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty" firestore:"resolvedAt,omitempty"`
	
	// Response deadline
	ResponseDeadline *time.Time `json:"response_deadline,omitempty" firestore:"responseDeadline,omitempty"`
	
	// Financial Impact
	DisputeAmount   float64 `json:"dispute_amount" firestore:"disputeAmount"`
	RefundAmount    float64 `json:"refund_amount,omitempty" firestore:"refundAmount,omitempty"`
	
	// Communication
	ChatID          string `json:"chat_id,omitempty" firestore:"chatId,omitempty"`
	LastMessageAt   *time.Time `json:"last_message_at,omitempty" firestore:"lastMessageAt,omitempty"`
	
	// Resolution Notes
	AdminNotes      string `json:"admin_notes,omitempty" firestore:"adminNotes,omitempty"`
	ResolutionNotes string `json:"resolution_notes,omitempty" firestore:"resolutionNotes,omitempty"`
}

type DisputeEvidence struct {
	ID          string    `json:"id" firestore:"id"`
	Type        string    `json:"type" firestore:"type"`         // screenshot, video, text, file
	Title       string    `json:"title" firestore:"title"`
	Description string    `json:"description" firestore:"description"`
	FileURL     string    `json:"file_url,omitempty" firestore:"fileUrl,omitempty"`
	Content     string    `json:"content,omitempty" firestore:"content,omitempty"` // for text evidence
	UploadedAt  time.Time `json:"uploaded_at" firestore:"uploadedAt"`
	UploadedBy  string    `json:"uploaded_by" firestore:"uploadedBy"`
}

// Dispute Log for audit trail
type DisputeLog struct {
	ID        string                 `json:"id" firestore:"id"`
	DisputeID string                 `json:"dispute_id" firestore:"disputeId"`
	UserID    string                 `json:"user_id" firestore:"userId"`
	UserRole  string                 `json:"user_role" firestore:"userRole"`
	Action    string                 `json:"action" firestore:"action"`
	Details   map[string]interface{} `json:"details" firestore:"details"`
	Timestamp time.Time              `json:"timestamp" firestore:"timestamp"`
}