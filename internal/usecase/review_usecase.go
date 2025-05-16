package usecase

import (
	"context"
	"time"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
	"pasargamex/pkg/logger"
	"pasargamex/pkg/utils"
)

type ReviewUseCase struct {
	reviewRepo repository.ReviewRepository
	userRepo   repository.UserRepository
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

	existingReview, err := uc.reviewRepo.GetByTransactionID(ctx, input.TransactionID)
	if err == nil && existingReview != nil {
		return nil, errors.BadRequest("Review for this transaction already exists", nil)
	}

	targetID := "dummy-seller-id"
	reviewType := "seller_review"
	productID := "dummy-product-id"

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

	if err := uc.updateUserRating(ctx, targetID, reviewType, input.Rating); err != nil {

		logger.Error("Failed to update user rating for user %s: %v", targetID, err)
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

	filter["status"] = "active"

	pagination := utils.NewPaginationParams(page, limit)

	return uc.reviewRepo.List(ctx, filter, pagination.PageSize, pagination.Offset)
}

func (uc *ReviewUseCase) ReportReview(ctx context.Context, reporterID, reviewID, reason, description string) (*entity.ReviewReport, error) {

	review, err := uc.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}

	if review.ReviewerID == reporterID {
		return nil, errors.BadRequest("Cannot report your own review", nil)
	}

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

	review.ReportCount++
	review.Status = "reported"

	if err := uc.reviewRepo.Update(ctx, review); err != nil {

		logger.Error("Failed to update review status after reporting review ID %s: %v", reviewID, err)
	}

	return report, nil
}

func (uc *ReviewUseCase) updateUserRating(ctx context.Context, userID, reviewType string, newRating int) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if reviewType == "seller_review" {

		totalRating := user.SellerRating * float64(user.SellerReviewCount)
		user.SellerReviewCount++
		user.SellerRating = (totalRating + float64(newRating)) / float64(user.SellerReviewCount)
	} else if reviewType == "buyer_review" {

		totalRating := user.BuyerRating * float64(user.BuyerReviewCount)
		user.BuyerReviewCount++
		user.BuyerRating = (totalRating + float64(newRating)) / float64(user.BuyerReviewCount)
	}

	return uc.userRepo.Update(ctx, user)
}

func (uc *ReviewUseCase) UpdateReviewStatus(ctx context.Context, adminID, reviewID, status, reason string) (*entity.Review, error) {

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

	pagination := utils.NewPaginationParams(page, limit)

	return uc.reviewRepo.ListReports(ctx, filter, pagination.PageSize, pagination.Offset)
}

func (uc *ReviewUseCase) ResolveReport(ctx context.Context, adminID, reportID, status string) (*entity.ReviewReport, error) {

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
