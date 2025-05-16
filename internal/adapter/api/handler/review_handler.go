package handler

import (
	"strconv"

	"github.com/labstack/echo/v4"

	"pasargamex/internal/usecase"
	"pasargamex/pkg/errors"
	"pasargamex/pkg/response"
)

type ReviewHandler struct {
	reviewUseCase *usecase.ReviewUseCase
}

func NewReviewHandler(reviewUseCase *usecase.ReviewUseCase) *ReviewHandler {
	return &ReviewHandler{
		reviewUseCase: reviewUseCase,
	}
}

type createReviewRequest struct {
	Rating  int      `json:"rating" validate:"required,min=1,max=5"`
	Content string   `json:"content" validate:"required"`
	Images  []string `json:"images,omitempty"`
}

func (h *ReviewHandler) CreateReview(c echo.Context) error {

	transactionID := c.Param("transactionId")
	if transactionID == "" {
		return response.Error(c, errors.BadRequest("Transaction ID is required", nil))
	}

	var req createReviewRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	userID := c.Get("uid").(string)

	review, err := h.reviewUseCase.CreateReview(c.Request().Context(), userID, usecase.CreateReviewInput{
		TransactionID: transactionID,
		Rating:        req.Rating,
		Content:       req.Content,
		Images:        req.Images,
	})

	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, review)
}

func (h *ReviewHandler) GetReviews(c echo.Context) error {

	userID := c.QueryParam("userId")
	reviewType := c.QueryParam("type")

	ratingStr := c.QueryParam("rating")
	var rating int
	if ratingStr != "" {
		var err error
		rating, err = strconv.Atoi(ratingStr)
		if err != nil || rating < 1 || rating > 5 {
			return response.Error(c, errors.BadRequest("Invalid rating value", nil))
		}
	}

	pageStr := c.QueryParam("page")
	limitStr := c.QueryParam("limit")

	page := 1
	limit := 20

	if pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}
	}

	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			limit = 20
		}
	}

	reviews, total, err := h.reviewUseCase.ListReviews(
		c.Request().Context(),
		userID,
		reviewType,
		rating,
		page,
		limit,
	)

	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, reviews, total, page, limit)
}

type reportReviewRequest struct {
	Reason      string `json:"reason" validate:"required,oneof=inappropriate spam fake offensive other"`
	Description string `json:"description" validate:"required"`
}

func (h *ReviewHandler) ReportReview(c echo.Context) error {

	reviewID := c.Param("reviewId")
	if reviewID == "" {
		return response.Error(c, errors.BadRequest("Review ID is required", nil))
	}

	var req reportReviewRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	userID := c.Get("uid").(string)

	report, err := h.reviewUseCase.ReportReview(
		c.Request().Context(),
		userID,
		reviewID,
		req.Reason,
		req.Description,
	)

	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, report)
}

type updateReviewStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=active hidden deleted"`
	Reason string `json:"reason,omitempty"`
}

func (h *ReviewHandler) UpdateReviewStatus(c echo.Context) error {

	reviewID := c.Param("reviewId")
	if reviewID == "" {
		return response.Error(c, errors.BadRequest("Review ID is required", nil))
	}

	var req updateReviewStatusRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	adminID := c.Get("uid").(string)

	review, err := h.reviewUseCase.UpdateReviewStatus(
		c.Request().Context(),
		adminID,
		reviewID,
		req.Status,
		req.Reason,
	)

	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, review)
}

func (h *ReviewHandler) GetReviewReports(c echo.Context) error {

	status := c.QueryParam("status")

	pageStr := c.QueryParam("page")
	limitStr := c.QueryParam("limit")

	page := 1
	limit := 20

	if pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}
	}

	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			limit = 20
		}
	}

	reports, total, err := h.reviewUseCase.ListReportedReviews(
		c.Request().Context(),
		status,
		page,
		limit,
	)

	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, reports, total, page, limit)
}
