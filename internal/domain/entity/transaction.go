package entity

import (
	"time"
)

type Transaction struct {
	ID                string                 `json:"id" firestore:"id"`
	ProductID         string                 `json:"product_id" firestore:"productId"`
	SellerID          string                 `json:"seller_id" firestore:"sellerId"`
	BuyerID           string                 `json:"buyer_id" firestore:"buyerId"`
	Status            string                 `json:"status" firestore:"status"` // "pending", "awaiting_payment", "processing", "completed", "cancelled", "disputed"
	DeliveryMethod    string                 `json:"delivery_method" firestore:"deliveryMethod"` // "instant", "middleman"
	Amount            float64                `json:"amount" firestore:"amount"`
	Fee               float64                `json:"fee" firestore:"fee"` // Processing fee
	TotalAmount       float64                `json:"total_amount" firestore:"totalAmount"` // Amount + Fee
	PaymentMethod     string                 `json:"payment_method,omitempty" firestore:"paymentMethod,omitempty"`
	PaymentStatus     string                 `json:"payment_status" firestore:"paymentStatus"` // "pending", "paid", "failed", "refunded"
	PaymentDetails    map[string]interface{} `json:"payment_details,omitempty" firestore:"paymentDetails,omitempty"`
	
	// Data untuk pengiriman digital
	Credentials       map[string]interface{} `json:"credentials,omitempty" firestore:"credentials,omitempty"` // Hanya ditampilkan ke buyer setelah pembayaran
	
	// Untuk middleman mode
	AdminID           string                 `json:"admin_id,omitempty" firestore:"adminId,omitempty"` // Admin yang menangani transaksi
	MiddlemanStatus   string                 `json:"middleman_status,omitempty" firestore:"middlemanStatus,omitempty"` // "assigned", "verifying", "completed"
	
	// Review status
	SellerReviewed    bool                   `json:"seller_reviewed" firestore:"sellerReviewed"`
	BuyerReviewed     bool                   `json:"buyer_reviewed" firestore:"buyerReviewed"`
	
	// Notes dan keterangan
	Notes             string                 `json:"notes,omitempty" firestore:"notes,omitempty"`
	CancellationReason string                `json:"cancellation_reason,omitempty" firestore:"cancellationReason,omitempty"`
	
	// Timestamps
	CreatedAt         time.Time              `json:"created_at" firestore:"createdAt"`
	UpdatedAt         time.Time              `json:"updated_at" firestore:"updatedAt"`
	PaymentAt         *time.Time             `json:"payment_at,omitempty" firestore:"paymentAt,omitempty"`
	ProcessingAt      *time.Time             `json:"processing_at,omitempty" firestore:"processingAt,omitempty"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty" firestore:"completedAt,omitempty"`
	CancelledAt       *time.Time             `json:"cancelled_at,omitempty" firestore:"cancelledAt,omitempty"`
	RefundedAt        *time.Time             `json:"refunded_at,omitempty" firestore:"refundedAt,omitempty"`
}

type TransactionLog struct {
	ID            string    `json:"id" firestore:"id"`
	TransactionID string    `json:"transaction_id" firestore:"transactionId"`
	Status        string    `json:"status" firestore:"status"`
	Notes         string    `json:"notes,omitempty" firestore:"notes,omitempty"`
	CreatedBy     string    `json:"created_by" firestore:"createdBy"` // User ID, admin ID, atau "system"
	CreatedAt     time.Time `json:"created_at" firestore:"createdAt"`
}