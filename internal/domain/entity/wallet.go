package entity

import (
	"time"
)

type Wallet struct {
	ID          string    `json:"id" firestore:"id"`
	UserID      string    `json:"user_id" firestore:"userId"`
	Balance     float64   `json:"balance" firestore:"balance"`
	Currency    string    `json:"currency" firestore:"currency"`
	Status      string    `json:"status" firestore:"status"` // active, suspended, frozen
	LastTxnAt   time.Time `json:"last_txn_at" firestore:"lastTxnAt"`
	CreatedAt   time.Time `json:"created_at" firestore:"createdAt"`
	UpdatedAt   time.Time `json:"updated_at" firestore:"updatedAt"`
}

type WalletTransaction struct {
	ID              string                 `json:"id" firestore:"id"`
	WalletID        string                 `json:"wallet_id" firestore:"walletId"`
	UserID          string                 `json:"user_id" firestore:"userId"`
	Type            string                 `json:"type" firestore:"type"`                       // topup, withdraw, payment, refund, fee
	Amount          float64                `json:"amount" firestore:"amount"`
	PreviousBalance float64                `json:"previous_balance" firestore:"previousBalance"`
	NewBalance      float64                `json:"new_balance" firestore:"newBalance"`
	Status          string                 `json:"status" firestore:"status"`                   // pending, completed, failed, cancelled
	Reference       string                 `json:"reference,omitempty" firestore:"reference,omitempty"` // Reference to transaction/topup ID
	PaymentMethod   string                 `json:"payment_method,omitempty" firestore:"paymentMethod,omitempty"`
	PaymentDetails  map[string]interface{} `json:"payment_details,omitempty" firestore:"paymentDetails,omitempty"`
	Description     string                 `json:"description" firestore:"description"`
	Metadata        map[string]interface{} `json:"metadata,omitempty" firestore:"metadata,omitempty"`
	ProcessedAt     *time.Time             `json:"processed_at,omitempty" firestore:"processedAt,omitempty"`
	CreatedAt       time.Time              `json:"created_at" firestore:"createdAt"`
	UpdatedAt       time.Time              `json:"updated_at" firestore:"updatedAt"`
}

type PaymentMethod struct {
	ID               string                 `json:"id" firestore:"id"`
	UserID           string                 `json:"user_id" firestore:"userId"`
	Type             string                 `json:"type" firestore:"type"`                     // bank_transfer, ewallet, credit_card, crypto
	Provider         string                 `json:"provider" firestore:"provider"`             // bca, mandiri, gopay, ovo, visa, etc
	AccountNumber    string                 `json:"account_number,omitempty" firestore:"accountNumber,omitempty"`
	AccountName      string                 `json:"account_name,omitempty" firestore:"accountName,omitempty"`
	IsDefault        bool                   `json:"is_default" firestore:"isDefault"`
	IsActive         bool                   `json:"is_active" firestore:"isActive"`
	Details          map[string]interface{} `json:"details,omitempty" firestore:"details,omitempty"`
	CreatedAt        time.Time              `json:"created_at" firestore:"createdAt"`
	UpdatedAt        time.Time              `json:"updated_at" firestore:"updatedAt"`
}

type TopupRequest struct {
	ID                string                 `json:"id" firestore:"id"`
	UserID            string                 `json:"user_id" firestore:"userId"`
	WalletID          string                 `json:"wallet_id" firestore:"walletId"`
	Amount            float64                `json:"amount" firestore:"amount"`
	PaymentMethodID   string                 `json:"payment_method_id" firestore:"paymentMethodId"`
	PaymentReference  string                 `json:"payment_reference,omitempty" firestore:"paymentReference,omitempty"`
	Status            string                 `json:"status" firestore:"status"`                   // pending, completed, failed, expired
	PaymentProof      string                 `json:"payment_proof,omitempty" firestore:"paymentProof,omitempty"`
	AdminNotes        string                 `json:"admin_notes,omitempty" firestore:"adminNotes,omitempty"`
	ProcessedBy       string                 `json:"processed_by,omitempty" firestore:"processedBy,omitempty"`
	ProcessedAt       *time.Time             `json:"processed_at,omitempty" firestore:"processedAt,omitempty"`
	ExpiresAt         time.Time              `json:"expires_at" firestore:"expiresAt"`
	CreatedAt         time.Time              `json:"created_at" firestore:"createdAt"`
	UpdatedAt         time.Time              `json:"updated_at" firestore:"updatedAt"`
}

type WithdrawRequest struct {
	ID                string                 `json:"id" firestore:"id"`
	UserID            string                 `json:"user_id" firestore:"userId"`
	WalletID          string                 `json:"wallet_id" firestore:"walletId"`
	Amount            float64                `json:"amount" firestore:"amount"`
	Fee               float64                `json:"fee" firestore:"fee"`
	NetAmount         float64                `json:"net_amount" firestore:"netAmount"`
	PaymentMethodID   string                 `json:"payment_method_id" firestore:"paymentMethodId"`
	Status            string                 `json:"status" firestore:"status"`                   // pending, processing, completed, failed, rejected
	Reason            string                 `json:"reason,omitempty" firestore:"reason,omitempty"`
	AdminNotes        string                 `json:"admin_notes,omitempty" firestore:"adminNotes,omitempty"`
	ProcessedBy       string                 `json:"processed_by,omitempty" firestore:"processedBy,omitempty"`
	ProcessedAt       *time.Time             `json:"processed_at,omitempty" firestore:"processedAt,omitempty"`
	CreatedAt         time.Time              `json:"created_at" firestore:"createdAt"`
	UpdatedAt         time.Time              `json:"updated_at" firestore:"updatedAt"`
}