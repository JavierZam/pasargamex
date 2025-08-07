package entity

import "time"

type TransactionApproval struct {
	ID            string                 `json:"id" firestore:"id"`
	TransactionID string                 `json:"transaction_id" firestore:"transactionId"`
	ApproverType  string                 `json:"approver_type" firestore:"approverType"` // seller, middleman, buyer, system
	ApproverID    string                 `json:"approver_id" firestore:"approverId"`
	ApprovalStep  string                 `json:"approval_step" firestore:"approvalStep"` // delivery_confirmed, verification_complete, item_received
	Status        string                 `json:"status" firestore:"status"`              // pending, approved, rejected
	EvidenceURL   string                 `json:"evidence_url,omitempty" firestore:"evidenceUrl,omitempty"`
	Notes         string                 `json:"notes,omitempty" firestore:"notes,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty" firestore:"ipAddress,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty" firestore:"userAgent,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" firestore:"metadata,omitempty"`
	ApprovedAt    *time.Time             `json:"approved_at,omitempty" firestore:"approvedAt,omitempty"`
	CreatedAt     time.Time              `json:"created_at" firestore:"createdAt"`
	UpdatedAt     time.Time              `json:"updated_at" firestore:"updatedAt"`
}

type MiddlemanProfile struct {
	ID                     string    `json:"id" firestore:"id"`
	UserID                 string    `json:"user_id" firestore:"userId"`
	KYCStatus              string    `json:"kyc_status" firestore:"kycStatus"` // pending, verified, rejected
	SecurityDeposit        float64   `json:"security_deposit" firestore:"securityDeposit"`
	PerformanceScore       float64   `json:"performance_score" firestore:"performanceScore"` // 0.00 - 5.00
	TotalTransactions      int       `json:"total_transactions" firestore:"totalTransactions"`
	SuccessfulTransactions int       `json:"successful_transactions" firestore:"successfulTransactions"`
	DisputeCount          int       `json:"dispute_count" firestore:"disputeCount"`
	DailyLimit            float64   `json:"daily_limit" firestore:"dailyLimit"`
	MonthlyLimit          float64   `json:"monthly_limit" firestore:"monthlyLimit"`
	TrustLevel            string    `json:"trust_level" firestore:"trustLevel"` // bronze, silver, gold, platinum
	IsActive              bool      `json:"is_active" firestore:"isActive"`
	LastAuditAt           *time.Time `json:"last_audit_at,omitempty" firestore:"lastAuditAt,omitempty"`
	CreatedAt             time.Time `json:"created_at" firestore:"createdAt"`
	UpdatedAt             time.Time `json:"updated_at" firestore:"updatedAt"`
}

func (mp *MiddlemanProfile) IsEligible(transactionAmount float64) bool {
	if !mp.IsActive || mp.KYCStatus != "verified" {
		return false
	}
	
	// Check daily limits based on trust level
	maxAmount := mp.DailyLimit
	if maxAmount == 0 {
		// Default limits by trust level
		switch mp.TrustLevel {
		case "bronze":
			maxAmount = 5000000 // 5 juta
		case "silver":
			maxAmount = 20000000 // 20 juta
		case "gold":
			maxAmount = 50000000 // 50 juta
		case "platinum":
			maxAmount = 999999999 // Unlimited
		default:
			maxAmount = 1000000 // 1 juta for new users
		}
	}
	
	return transactionAmount <= maxAmount
}

func (mp *MiddlemanProfile) CalculatePerformanceScore() float64 {
	if mp.TotalTransactions == 0 {
		return 0.0
	}
	
	successRate := float64(mp.SuccessfulTransactions) / float64(mp.TotalTransactions)
	disputeRate := float64(mp.DisputeCount) / float64(mp.TotalTransactions)
	
	// Base score from success rate (0-4 points)
	baseScore := successRate * 4.0
	
	// Penalty for disputes (up to -1 point)
	penalty := disputeRate * 1.0
	
	// Bonus for volume (up to +1 point)
	volumeBonus := 0.0
	if mp.TotalTransactions >= 100 {
		volumeBonus = 1.0
	} else if mp.TotalTransactions >= 50 {
		volumeBonus = 0.5
	} else if mp.TotalTransactions >= 10 {
		volumeBonus = 0.25
	}
	
	score := baseScore - penalty + volumeBonus
	if score > 5.0 {
		score = 5.0
	}
	if score < 0.0 {
		score = 0.0
	}
	
	return score
}

type SecurityLog struct {
	ID            string                 `json:"id" firestore:"id"`
	TransactionID string                 `json:"transaction_id,omitempty" firestore:"transactionId,omitempty"`
	UserID        string                 `json:"user_id" firestore:"userId"`
	EventType     string                 `json:"event_type" firestore:"eventType"`
	RiskScore     int                    `json:"risk_score" firestore:"riskScore"` // 0-100
	Details       map[string]interface{} `json:"details,omitempty" firestore:"details,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty" firestore:"ipAddress,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty" firestore:"userAgent,omitempty"`
	CreatedAt     time.Time              `json:"created_at" firestore:"createdAt"`
}