package usecase

import (
	"context"
	"time"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
)

// FeeCalculator adalah interface untuk menghitung biaya transaksi
type FeeCalculator interface {
	CalculateFee(amount float64, paymentMethod string) float64
}

// Implementasi sederhana untuk FeeCalculator
type defaultFeeCalculator struct{}

func (fc *defaultFeeCalculator) CalculateFee(amount float64, paymentMethod string) float64 {
	// Implementasi sederhana: 2.5% dari amount
	return amount * 0.025
}

type TransactionUseCase struct {
	transactionRepo repository.TransactionRepository
	productRepo     repository.ProductRepository
	userRepo        repository.UserRepository
	feeCalculator   FeeCalculator
}

func NewTransactionUseCase(
	transactionRepo repository.TransactionRepository,
	productRepo repository.ProductRepository,
	userRepo repository.UserRepository,
) *TransactionUseCase {
	return &TransactionUseCase{
		transactionRepo: transactionRepo,
		productRepo:     productRepo,
		userRepo:        userRepo,
		feeCalculator:   &defaultFeeCalculator{},
	}
}

type CreateTransactionInput struct {
	ProductID      string
	DeliveryMethod string // "instant" atau "middleman"
	Notes          string
}

func (uc *TransactionUseCase) CreateTransaction(ctx context.Context, buyerID string, input CreateTransactionInput) (*entity.Transaction, error) {
	// Validasi produk
	product, err := uc.productRepo.GetByID(ctx, input.ProductID)
	if err != nil {
		return nil, err
	}
	
	// Validasi seller
	seller, err := uc.userRepo.GetByID(ctx, product.SellerID)
	if err != nil {
		return nil, err
	}
	
	// Pastikan buyer tidak membeli produknya sendiri
	if product.SellerID == buyerID {
		return nil, errors.BadRequest("Cannot buy your own product", nil)
	}
	
	// Pastikan produk aktif
	if product.Status != "active" {
		return nil, errors.BadRequest("Product is not available", nil)
	}
	
	// Validasi bahwa seller terverifikasi
	if seller.VerificationStatus != "verified" {
		return nil, errors.BadRequest("Seller is not verified", nil)
	}
	
	// Validasi delivery method
	if input.DeliveryMethod != "instant" && input.DeliveryMethod != "middleman" {
		return nil, errors.BadRequest("Invalid delivery method", nil)
	}
	
	// Untuk instant delivery, pastikan credentials sudah ada
	if input.DeliveryMethod == "instant" && len(product.Credentials) == 0 {
		return nil, errors.BadRequest("Product credentials are not available", nil)
	}
	
	// Hitung fee
	fee := uc.feeCalculator.CalculateFee(product.Price, "")
	totalAmount := product.Price + fee
	
	// Buat transaksi
	transaction := &entity.Transaction{
		ProductID:      input.ProductID,
		SellerID:       product.SellerID,
		BuyerID:        buyerID,
		Status:         "pending",
		DeliveryMethod: input.DeliveryMethod,
		Amount:         product.Price,
		Fee:            fee,
		TotalAmount:    totalAmount,
		PaymentStatus:  "pending",
		SellerReviewed: false,
		BuyerReviewed:  false,
		Notes:          input.Notes,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	// Untuk instant delivery, siapkan credentials (akan dienkripsi/disembunyikan sampai pembayaran)
	if input.DeliveryMethod == "instant" {
		// Simpan credentials dari produk, tapi tidak ditampilkan sampai pembayaran
		transaction.Credentials = product.Credentials
	}
	
	if err := uc.transactionRepo.Create(ctx, transaction); err != nil {
		return nil, err
	}
	
	// Buat log
	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        "pending",
		Notes:         "Transaction created",
		CreatedBy:     buyerID,
		CreatedAt:     time.Now(),
	}
	
	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		// Log error tapi jangan gagalkan operasi
	}
	
	return transaction, nil
}

func (uc *TransactionUseCase) GetTransactionByID(ctx context.Context, userID, transactionID string) (*entity.Transaction, error) {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	
	// Validasi bahwa user adalah buyer atau seller
	if transaction.BuyerID != userID && transaction.SellerID != userID && transaction.AdminID != userID {
		return nil, errors.Forbidden("You don't have permission to view this transaction", nil)
	}
	
	// Sembunyikan credentials jika belum dibayar dan bukan admin
	if transaction.PaymentStatus != "paid" && transaction.AdminID != userID {
		// Buat salinan untuk menghindari modifikasi objek asli
		transactionCopy := *transaction
		transactionCopy.Credentials = nil
		return &transactionCopy, nil
	}
	
	// Sembunyikan credentials jika bukan buyer dan bukan admin
	if transaction.BuyerID != userID && transaction.AdminID != userID {
		// Buat salinan untuk menghindari modifikasi objek asli
		transactionCopy := *transaction
		transactionCopy.Credentials = nil
		return &transactionCopy, nil
	}
	
	return transaction, nil
}

func (uc *TransactionUseCase) ListTransactions(ctx context.Context, userID, role, status string, page, limit int) ([]*entity.Transaction, int64, error) {
	// Validasi role
	if role != "buyer" && role != "seller" {
		// Default ke buyer jika tidak dispesifikasi
		role = "buyer"
	}
	
	// Hitung offset
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	
	// List transaksi
	transactions, total, err := uc.transactionRepo.ListByUserID(ctx, userID, role, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	
	// Proses hasil untuk menyembunyikan credentials sesuai kebutuhan
	for i, transaction := range transactions {
		if transaction.BuyerID != userID || transaction.PaymentStatus != "paid" {
			// Buat salinan untuk menghindari modifikasi objek asli
			transactionCopy := *transaction
			transactionCopy.Credentials = nil
			transactions[i] = &transactionCopy
		}
	}
	
	return transactions, total, nil
}

// ProcessPayment yang diperbarui untuk menangani instant dan middleman berbeda
func (uc *TransactionUseCase) ProcessPayment(ctx context.Context, userID, transactionID, paymentMethod string, paymentDetails map[string]interface{}) (*entity.Transaction, error) {
	// Validasi transaksi
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	
	// Validasi user adalah buyer
	if transaction.BuyerID != userID {
		return nil, errors.Forbidden("Only buyer can make payment", nil)
	}
	
	// Validasi status transaksi
	if transaction.Status != "pending" && transaction.Status != "awaiting_payment" {
		return nil, errors.BadRequest("Invalid transaction status for payment", nil)
	}
	
	// TODO: Implementasi integrasi payment gateway
	// Untuk saat ini, kita langsung update status pembayaran
	
	// Update transaksi
	transaction.PaymentMethod = paymentMethod
	transaction.PaymentDetails = paymentDetails
	transaction.PaymentStatus = "paid"
	
	now := time.Now()
	transaction.PaymentAt = &now
	
	// Menangani berdasarkan delivery method
	if transaction.DeliveryMethod == "instant" {
		// Untuk instant delivery, langsung ke status completed
		transaction.Status = "completed"
		transaction.CompletedAt = &now
		
		// Credentials sudah tersedia (sudah diisi saat create transaction)
	} else if transaction.DeliveryMethod == "middleman" {
		// Untuk middleman, status menjadi processing dan menunggu admin
		transaction.Status = "processing"
		transaction.ProcessingAt = &now
		transaction.MiddlemanStatus = "pending_assignment"
		
		// TODO: Kirim notifikasi ke admin untuk assignment
	}
	
	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return nil, err
	}
	
	// Buat log
	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        transaction.Status,
		Notes:         "Payment processed via " + paymentMethod,
		CreatedBy:     userID,
		CreatedAt:     time.Now(),
	}
	
	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		// Log error tapi jangan gagalkan operasi
	}
	
	return transaction, nil
}

// AssignMiddleman untuk admin assignment pada middleman mode
func (uc *TransactionUseCase) AssignMiddleman(ctx context.Context, adminID, transactionID string) (*entity.Transaction, error) {
	// TODO: Validasi adminID adalah admin yang valid
	
	// Validasi transaksi
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	
	// Validasi delivery method adalah middleman
	if transaction.DeliveryMethod != "middleman" {
		return nil, errors.BadRequest("Transaction is not using middleman delivery method", nil)
	}
	
	// Validasi status transaksi
	if transaction.Status != "processing" || transaction.MiddlemanStatus != "pending_assignment" {
		return nil, errors.BadRequest("Transaction is not ready for middleman assignment", nil)
	}
	
	// Update transaksi
	transaction.AdminID = adminID
	transaction.MiddlemanStatus = "assigned"
	transaction.UpdatedAt = time.Now()
	
	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return nil, err
	}
	
	// Buat log
	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        transaction.Status,
		Notes:         "Middleman assigned",
		CreatedBy:     adminID,
		CreatedAt:     time.Now(),
	}
	
	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		// Log error tapi jangan gagalkan operasi
	}
	
	return transaction, nil
}

// VerifyAndCompleteMiddleman untuk admin menyelesaikan transaksi dengan middleman
func (uc *TransactionUseCase) VerifyAndCompleteMiddleman(ctx context.Context, adminID, transactionID string, credentials map[string]interface{}) (*entity.Transaction, error) {
	// Validasi transaksi
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	
	// Validasi admin adalah yang ditugaskan
	if transaction.AdminID != adminID {
		return nil, errors.Forbidden("Only assigned admin can complete this transaction", nil)
	}
	
	// Validasi status transaksi
	if transaction.Status != "processing" || transaction.MiddlemanStatus != "assigned" {
		return nil, errors.BadRequest("Transaction is not ready for completion", nil)
	}
	
	// Update transaksi
	transaction.Status = "completed"
	transaction.MiddlemanStatus = "completed"
	transaction.Credentials = credentials
	
	now := time.Now()
	transaction.CompletedAt = &now
	transaction.UpdatedAt = now
	
	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return nil, err
	}
	
	// Buat log
	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        "completed",
		Notes:         "Transaction completed by middleman",
		CreatedBy:     adminID,
		CreatedAt:     time.Now(),
	}
	
	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		// Log error tapi jangan gagalkan operasi
	}
	
	return transaction, nil
}

func (uc *TransactionUseCase) GetTransactionLogs(ctx context.Context, userID, transactionID string) ([]*entity.TransactionLog, error) {
	// Validasi transaksi dan akses
	transaction, err := uc.GetTransactionByID(ctx, userID, transactionID)
	if err != nil {
		return nil, err
	}
	
	// Jika sudah melewati GetTransactionByID, berarti user memiliki akses
	
	// Ambil logs
	logs, err := uc.transactionRepo.ListLogsByTransactionID(ctx, transactionID)
	if err != nil {
		return nil, errors.Internal("Failed to get transaction logs", err)
	}
	
	return logs, nil
}

// CancelTransaction untuk membatalkan transaksi
func (uc *TransactionUseCase) CancelTransaction(ctx context.Context, userID, transactionID, reason string) (*entity.Transaction, error) {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	
	// Validasi apakah transaksi bisa dibatalkan
	if transaction.Status == "completed" || transaction.Status == "cancelled" {
		return nil, errors.BadRequest("Transaction cannot be cancelled in current status", nil)
	}
	
	// Validasi apakah user memiliki hak untuk membatalkan
	isBuyer := transaction.BuyerID == userID
	isSeller := transaction.SellerID == userID
	isAdmin := transaction.AdminID == userID
	
	if !isBuyer && !isSeller && !isAdmin {
		return nil, errors.Forbidden("You don't have permission to cancel this transaction", nil)
	}
	
	// Validasi status transition berdasarkan role
	if !uc.isValidStatusTransition(transaction.Status, "cancelled", transaction.DeliveryMethod, isBuyer, isSeller) && !isAdmin {
		return nil, errors.BadRequest("Cannot cancel transaction in current status", nil)
	}
	
	// Update transaksi
	transaction.Status = "cancelled"
	transaction.CancellationReason = reason
	
	now := time.Now()
	transaction.CancelledAt = &now
	transaction.UpdatedAt = now
	
	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return nil, err
	}
	
	// Buat log
	var createdBy string
	var notes string
	
	if isBuyer {
		createdBy = transaction.BuyerID
		notes = "Transaction cancelled by buyer"
	} else if isSeller {
		createdBy = transaction.SellerID
		notes = "Transaction cancelled by seller"
	} else {
		createdBy = transaction.AdminID
		notes = "Transaction cancelled by admin"
	}
	
	if reason != "" {
		notes += ": " + reason
	}
	
	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        "cancelled",
		Notes:         notes,
		CreatedBy:     createdBy,
		CreatedAt:     time.Now(),
	}
	
	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		// Log error tapi jangan gagalkan operasi
	}
	
	return transaction, nil
}

// CreateDispute menciptakan dispute untuk transaksi
func (uc *TransactionUseCase) CreateDispute(ctx context.Context, userID, transactionID, reason string) (*entity.Transaction, error) {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	
	// Validasi status transaksi
	if transaction.Status != "processing" {
		return nil, errors.BadRequest("Transaction cannot be disputed in current status", nil)
	}
	
	// Validasi user adalah buyer atau seller
	isBuyer := transaction.BuyerID == userID
	isSeller := transaction.SellerID == userID
	
	if !isBuyer && !isSeller {
		return nil, errors.Forbidden("Only buyer or seller can create a dispute", nil)
	}
	
	// Update transaksi
	transaction.Status = "disputed"
	transaction.UpdatedAt = time.Now()
	
	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return nil, err
	}
	
	// Buat log
	var createdBy string
	var notes string
	
	if isBuyer {
		createdBy = transaction.BuyerID
		notes = "Dispute created by buyer"
	} else {
		createdBy = transaction.SellerID
		notes = "Dispute created by seller"
	}
	
	if reason != "" {
		notes += ": " + reason
	}
	
	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        "disputed",
		Notes:         notes,
		CreatedBy:     createdBy,
		CreatedAt:     time.Now(),
	}
	
	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		// Log error tapi jangan gagalkan operasi
	}
	
	return transaction, nil
}

func (uc *TransactionUseCase) ResolveDispute(ctx context.Context, adminID, transactionID, resolution string, refund bool) (*entity.Transaction, error) {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	
	// Validasi status transaksi
	if transaction.Status != "disputed" {
		return nil, errors.BadRequest("Transaction is not in disputed status", nil)
	}
	
	// Tentukan status baru berdasarkan resolution
	var newStatus string
	now := time.Now()
	
	if refund {
		// Refund ke buyer
		newStatus = "cancelled"
		transaction.Status = newStatus
		transaction.CancelledAt = &now
		transaction.PaymentStatus = "refunded"
		transaction.RefundedAt = &now
	} else {
		// Lanjutkan transaksi
		newStatus = "completed"
		transaction.Status = newStatus
		transaction.CompletedAt = &now
	}
	
	transaction.UpdatedAt = now
	
	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return nil, err
	}
	
	// Buat log
	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        newStatus,
		Notes:         "Dispute resolved by admin: " + resolution,
		CreatedBy:     adminID,
		CreatedAt:     now,
	}
	
	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		// Log error tapi jangan gagalkan operasi
	}
	
	return transaction, nil
}

// ConfirmDelivery untuk buyer mengkonfirmasi penerimaan barang/jasa
func (uc *TransactionUseCase) ConfirmDelivery(ctx context.Context, buyerID, transactionID string) (*entity.Transaction, error) {
	transaction, err := uc.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	
	// Validasi user adalah buyer
	if transaction.BuyerID != buyerID {
		return nil, errors.Forbidden("Only buyer can confirm delivery", nil)
	}
	
	// Validasi status transaksi
	if transaction.Status != "processing" {
		return nil, errors.BadRequest("Transaction is not in processing status", nil)
	}
	
	// Update transaksi
	transaction.Status = "completed"
	now := time.Now()
	transaction.CompletedAt = &now
	transaction.UpdatedAt = now
	
	if err := uc.transactionRepo.Update(ctx, transaction); err != nil {
		return nil, err
	}
	
	// Buat log
	log := &entity.TransactionLog{
		TransactionID: transaction.ID,
		Status:        "completed",
		Notes:         "Delivery confirmed by buyer",
		CreatedBy:     buyerID,
		CreatedAt:     now,
	}
	
	if err := uc.transactionRepo.CreateLog(ctx, log); err != nil {
		// Log error tapi jangan gagalkan operasi
	}
	
	return transaction, nil
}

// Helper - Validasi flow status transaksi
func (uc *TransactionUseCase) isValidStatusTransition(currentStatus, newStatus, deliveryMethod string, isBuyer, isSeller bool) bool {
	// Buat mapping status yang valid
	validTransitions := map[string]map[string]struct {
		allowed bool
		roles   []string // "buyer", "seller", "admin"
	}{
		"pending": {
			"awaiting_payment": {true, []string{"buyer"}},
			"cancelled":        {true, []string{"buyer", "seller"}},
		},
		"awaiting_payment": {
			"processing": {true, []string{"system", "admin"}}, // Dari payment gateway callback
			"completed":  {true, []string{"system", "admin"}}, // Untuk instant delivery
			"cancelled":  {true, []string{"buyer", "seller", "admin"}},
		},
		"processing": {
			"completed": {true, []string{"admin", "buyer"}}, // Untuk middleman atau konfirmasi buyer
			"cancelled": {true, []string{"admin"}},
			"disputed":  {true, []string{"buyer", "seller"}},
		},
		"disputed": {
			"completed": {true, []string{"admin"}},
			"cancelled": {true, []string{"admin"}},
		},
	}
	
	// Periksa jika transisi valid
	if transition, exists := validTransitions[currentStatus][newStatus]; exists {
		// Periksa apakah role user diizinkan
		for _, role := range transition.roles {
			if (role == "buyer" && isBuyer) || (role == "seller" && isSeller) || role == "system" || role == "admin" {
				return true
			}
		}
	}
	
	return false
}

func (uc *TransactionUseCase) ListAdminTransactions(ctx context.Context, adminID string, filter map[string]interface{}, page, limit int) ([]*entity.Transaction, int64, error) {
    // Validasi admin
    // TODO: Implement proper admin validation
    
    // Calculate offset
    offset := (page - 1) * limit
    if offset < 0 {
        offset = 0
    }
    
    // Get transactions from repository
    transactions, total, err := uc.transactionRepo.List(ctx, filter, limit, offset)
    if err != nil {
        return nil, 0, errors.Internal("Failed to list transactions", err)
    }
    
    return transactions, total, nil
}

// ListPendingMiddlemanTransactions untuk admin melihat transaksi yang membutuhkan middleman
func (uc *TransactionUseCase) ListPendingMiddlemanTransactions(ctx context.Context, adminID string, page, limit int) ([]*entity.Transaction, int64, error) {
    // Validasi admin
    // TODO: Implement proper admin validation
    
    // Calculate offset
    offset := (page - 1) * limit
    if offset < 0 {
        offset = 0
    }
    
    // Get pending middleman transactions from repository
    transactions, total, err := uc.transactionRepo.ListPendingMiddlemanTransactions(ctx, limit, offset)
    if err != nil {
        return nil, 0, errors.Internal("Failed to list pending middleman transactions", err)
    }
    
    return transactions, total, nil
}