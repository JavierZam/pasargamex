package usecase

import (
	"context"
	"time"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
)

type ReviewUseCase struct {
	reviewRepo  repository.ReviewRepository
	userRepo    repository.UserRepository
	// Nanti kita perlu menambahkan repository transaction
	// transactionRepo repository.TransactionRepository
}

func NewReviewUseCase(
	reviewRepo repository.ReviewRepository,
	userRepo repository.UserRepository,
) *ReviewUseCase {
	return &ReviewUseCase{
		reviewRepo: reviewRepo,
		userRepo:   userRepo,
	}
}

type CreateReviewInput struct {
	TransactionID string
	Rating        int
	Content       string
	Images        []string
}

func (uc *ReviewUseCase) CreateReview(ctx context.Context, reviewerID string, input CreateReviewInput) (*entity.Review, error) {
	// TODO: Validasi transaksi
	// transaction, err := uc.transactionRepo.GetByID(ctx, input.TransactionID)
	// if err != nil {
	//     return nil, err
	// }
	
	// Pengecekan apakah review sudah ada
	existingReview, err := uc.reviewRepo.GetByTransactionID(ctx, input.TransactionID)
	if err == nil && existingReview != nil {
		return nil, errors.BadRequest("Review for this transaction already exists", nil)
	}
	
	// TODO: Validasi user adalah bagian dari transaksi
	// Tentukan siapa yang di-review (buyer atau seller)
	// Di sini saya contohkan seller yang di-review
	targetID := "dummy-seller-id" // Seharusnya dari transaksi
	reviewType := "seller_review"
	productID := "dummy-product-id" // Seharusnya dari transaksi
	
	// Buat review
	review := &entity.Review{
		TransactionID: input.TransactionID,
		ProductID:     productID,
		ReviewerID:    reviewerID,
		TargetID:      targetID,
		Type:          reviewType,
		Rating:        input.Rating,
		Content:       input.Content,
		Images:        input.Images,
		Status:        "active",
		ReportCount:   0,
	}
	
	if err := uc.reviewRepo.Create(ctx, review); err != nil {
		return nil, err
	}
	
	// Update rating user yang di-review
	if err := uc.updateUserRating(ctx, targetID, reviewType, input.Rating); err != nil {
		// Log error tapi jangan gagalkan operasi
		// TODO: Implement logger
	}
	
	return review, nil
}

func (uc *ReviewUseCase) GetReviewByID(ctx context.Context, id string) (*entity.Review, error) {
	return uc.reviewRepo.GetByID(ctx, id)
}

func (uc *ReviewUseCase) ListReviews(ctx context.Context, userID, type_ string, rating int, page, limit int) ([]*entity.Review, int64, error) {
	filter := make(map[string]interface{})
	
	if userID != "" {
		filter["targetId"] = userID
	}
	
	if type_ != "" {
		filter["type"] = type_
	}
	
	if rating > 0 {
		filter["rating"] = rating
	}
	
	// Hanya tampilkan review aktif
	filter["status"] = "active"
	
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	
	return uc.reviewRepo.List(ctx, filter, limit, offset)
}

func (uc *ReviewUseCase) ReportReview(ctx context.Context, reporterID, reviewID, reason, description string) (*entity.ReviewReport, error) {
	// Validasi review
	review, err := uc.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	
	// Validasi user tidak melaporkan review miliknya sendiri
	if review.ReviewerID == reporterID {
		return nil, errors.BadRequest("Cannot report your own review", nil)
	}
	
	// Buat report
	report := &entity.ReviewReport{
		ReviewID:    reviewID,
		ReporterID:  reporterID,
		Reason:      reason,
		Description: description,
		Status:      "pending",
	}
	
	if err := uc.reviewRepo.CreateReport(ctx, report); err != nil {
		return nil, err
	}
	
	// Update report count di review
	review.ReportCount++
	review.Status = "reported" // Ubah status jika perlu
	
	if err := uc.reviewRepo.Update(ctx, review); err != nil {
		// Log error tapi jangan gagalkan operasi
		// TODO: Implement logger
	}
	
	return report, nil
}

// updateUserRating menghitung ulang rating user berdasarkan review baru
func (uc *ReviewUseCase) updateUserRating(ctx context.Context, userID, reviewType string, newRating int) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	
	if reviewType == "seller_review" {
		// Update seller rating
		totalRating := user.SellerRating * float64(user.SellerReviewCount)
		user.SellerReviewCount++
		user.SellerRating = (totalRating + float64(newRating)) / float64(user.SellerReviewCount)
	} else if reviewType == "buyer_review" {
		// Update buyer rating
		totalRating := user.BuyerRating * float64(user.BuyerReviewCount)
		user.BuyerReviewCount++
		user.BuyerRating = (totalRating + float64(newRating)) / float64(user.BuyerReviewCount)
	}
	
	return uc.userRepo.Update(ctx, user)
}

// Admin methods
func (uc *ReviewUseCase) UpdateReviewStatus(ctx context.Context, adminID, reviewID, status, reason string) (*entity.Review, error) {
	// TODO: Validate admin
	
	review, err := uc.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	
	review.Status = status
	review.UpdatedAt = time.Now()
	
	if err := uc.reviewRepo.Update(ctx, review); err != nil {
		return nil, err
	}
	
	return review, nil
}

func (uc *ReviewUseCase) ListReportedReviews(ctx context.Context, status string, page, limit int) ([]*entity.ReviewReport, int64, error) {
	filter := make(map[string]interface{})
	
	if status != "" {
		filter["status"] = status
	}
	
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	
	return uc.reviewRepo.ListReports(ctx, filter, limit, offset)
}

func (uc *ReviewUseCase) ResolveReport(ctx context.Context, adminID, reportID, status string) (*entity.ReviewReport, error) {
	// TODO: Validate admin
	
	report, err := uc.reviewRepo.GetReportByID(ctx, reportID)
	if err != nil {
		return nil, err
	}
	
	report.Status = status
	now := time.Now()
	report.ResolvedAt = &now
	
	if err := uc.reviewRepo.UpdateReport(ctx, report); err != nil {
		return nil, err
	}
	
	return report, nil
}