package handler

import (
	"pasargamex/internal/usecase"
	"pasargamex/pkg/response"
	"pasargamex/pkg/utils"
	"pasargamex/pkg/errors"

	"github.com/labstack/echo/v4"
)

type WishlistHandler struct {
	wishlistUseCase *usecase.WishlistUseCase
}

func NewWishlistHandler(wishlistUseCase *usecase.WishlistUseCase) *WishlistHandler {
	return &WishlistHandler{
		wishlistUseCase: wishlistUseCase,
	}
}

func (h *WishlistHandler) AddToWishlist(c echo.Context) error {
	userID := c.Get("uid").(string)
	productID := c.Param("productId")

	if productID == "" {
		return response.Error(c, errors.BadRequest("Product ID is required", nil))
	}

	result, err := h.wishlistUseCase.AddToWishlist(c.Request().Context(), userID, productID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, result)
}

func (h *WishlistHandler) RemoveFromWishlist(c echo.Context) error {
	userID := c.Get("uid").(string)
	productID := c.Param("productId")

	if productID == "" {
		return response.Error(c, errors.BadRequest("Product ID is required", nil))
	}

	err := h.wishlistUseCase.RemoveFromWishlist(c.Request().Context(), userID, productID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, map[string]string{
		"message": "Product removed from wishlist successfully",
	})
}

func (h *WishlistHandler) GetUserWishlist(c echo.Context) error {
	userID := c.Get("uid").(string)
	
	pagination := utils.GetPaginationParams(c)

	items, total, err := h.wishlistUseCase.GetUserWishlist(
		c.Request().Context(),
		userID,
		pagination.Page,
		pagination.PageSize,
	)

	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, items, total, pagination.Page, pagination.PageSize)
}

func (h *WishlistHandler) CheckWishlistStatus(c echo.Context) error {
	userID := c.Get("uid").(string)
	productID := c.Param("productId")

	if productID == "" {
		return response.Error(c, errors.BadRequest("Product ID is required", nil))
	}

	isInWishlist, err := h.wishlistUseCase.IsInWishlist(c.Request().Context(), userID, productID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, map[string]interface{}{
		"product_id":     productID,
		"is_in_wishlist": isInWishlist,
	})
}

func (h *WishlistHandler) GetWishlistCount(c echo.Context) error {
	userID := c.Get("uid").(string)

	count, err := h.wishlistUseCase.GetWishlistCount(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, map[string]interface{}{
		"count": count,
	})
}