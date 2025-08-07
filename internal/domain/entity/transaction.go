package entity

import (
	"time"
)

type Transaction struct {
	ID             string                 `json:"id" firestore:"id"`
	ProductID      string                 `json:"product_id" firestore:"productId"`
	SellerID       string                 `json:"seller_id" firestore:"sellerId"`
	BuyerID        string                 `json:"buyer_id" firestore:"buyerId"`
	Status         string                 `json:"status" firestore:"status"`
	DeliveryMethod string                 `json:"delivery_method" firestore:"deliveryMethod"`
	Amount         float64                `json:"amount" firestore:"amount"`
	Fee            float64                `json:"fee" firestore:"fee"`
	TotalAmount    float64                `json:"total_amount" firestore:"totalAmount"`
	PaymentMethod    string                 `json:"payment_method,omitempty" firestore:"paymentMethod,omitempty"`
	PaymentStatus    string                 `json:"payment_status" firestore:"paymentStatus"`
	PaymentDetails   map[string]interface{} `json:"payment_details,omitempty" firestore:"paymentDetails,omitempty"`
	
	// Midtrans Integration Fields
	MidtransOrderID  string `json:"midtrans_order_id,omitempty" firestore:"midtransOrderId,omitempty"`
	MidtransToken    string `json:"midtrans_token,omitempty" firestore:"midtransToken,omitempty"`
	MidtransRedirectURL string `json:"midtrans_redirect_url,omitempty" firestore:"midtransRedirectUrl,omitempty"`
	
	// Security & Approval Fields
	RequiredApprovals []string               `json:"required_approvals,omitempty" firestore:"requiredApprovals,omitempty"`
	CompletedApprovals []string             `json:"completed_approvals,omitempty" firestore:"completedApprovals,omitempty"`
	SecurityLevel     string                 `json:"security_level,omitempty" firestore:"securityLevel,omitempty"`
	EscrowStatus      string                 `json:"escrow_status,omitempty" firestore:"escrowStatus,omitempty"` // held, released, refunded

	Credentials map[string]interface{} `json:"-" firestore:"credentials,omitempty"`
	
	// Credential Delivery Fields
	CredentialsDelivered bool       `json:"credentials_delivered" firestore:"credentialsDelivered"`
	CredentialsDeliveredAt *time.Time `json:"credentials_delivered_at,omitempty" firestore:"credentialsDeliveredAt,omitempty"`
	BuyerConfirmedCredentials bool   `json:"buyer_confirmed_credentials" firestore:"buyerConfirmedCredentials"`
	BuyerConfirmedAt *time.Time      `json:"buyer_confirmed_at,omitempty" firestore:"buyerConfirmedAt,omitempty"`
	
	// Auto-release timer
	AutoReleaseAt *time.Time `json:"auto_release_at,omitempty" firestore:"autoReleaseAt,omitempty"`

	AdminID         string `json:"admin_id,omitempty" firestore:"adminId,omitempty"`
	MiddlemanStatus string `json:"middleman_status,omitempty" firestore:"middlemanStatus,omitempty"`
	MiddlemanChatID string `json:"middleman_chat_id,omitempty" firestore:"middlemanChatId,omitempty"` // New: Chat ID for middleman transaction

	SellerReviewed bool `json:"seller_reviewed" firestore:"sellerReviewed"`
	BuyerReviewed  bool `json:"buyer_reviewed" firestore:"buyerReviewed"`

	Notes              string `json:"notes,omitempty" firestore:"notes,omitempty"`
	CancellationReason string `json:"cancellation_reason,omitempty" firestore:"cancellationReason,omitempty"`

	CreatedAt    time.Time  `json:"created_at" firestore:"createdAt"`
	UpdatedAt    time.Time  `json:"updated_at" firestore:"updatedAt"`
	PaymentAt    *time.Time `json:"payment_at,omitempty" firestore:"paymentAt,omitempty"`
	ProcessingAt *time.Time `json:"processing_at,omitempty" firestore:"processingAt,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty" firestore:"completedAt,omitempty"`
	CancelledAt  *time.Time `json:"cancelled_at,omitempty" firestore:"cancelledAt,omitempty"`
	RefundedAt   *time.Time `json:"refunded_at,omitempty" firestore:"refundedAt,omitempty"`
}

type TransactionLog struct {
	ID            string    `json:"id" firestore:"id"`
	TransactionID string    `json:"transaction_id" firestore:"transactionId"`
	Status        string    `json:"status" firestore:"status"`
	Notes         string    `json:"notes,omitempty" firestore:"notes,omitempty"`
	CreatedBy     string    `json:"created_by" firestore:"createdBy"`
	CreatedAt     time.Time `json:"created_at" firestore:"createdAt"`
}
