package handler

import (
	"pasargamex/internal/usecase"
	"pasargamex/pkg/response"
	"pasargamex/pkg/utils"

	"github.com/labstack/echo/v4"
)

type GameTitleHandler struct {
	gameTitleUseCase *usecase.GameTitleUseCase
}

func NewGameTitleHandler(gameTitleUseCase *usecase.GameTitleUseCase) *GameTitleHandler {
	return &GameTitleHandler{
		gameTitleUseCase: gameTitleUseCase,
	}
}

type gameTitleAttributeRequest struct {
	Name        string   `json:"name" validate:"required"`
	Type        string   `json:"type" validate:"required,oneof=string number boolean select"`
	Required    bool     `json:"required"`
	Options     []string `json:"options,omitempty"`
	Description string   `json:"description,omitempty"`
}

type createGameTitleRequest struct {
    Name        string                     `json:"name" validate:"required"`
    Description string                     `json:"description"`
    Icon        string                     `json:"icon" validate:"omitempty,url"`
    Banner      string                     `json:"banner" validate:"omitempty,url"`
    Attributes  []gameTitleAttributeRequest `json:"attributes"`
    Status      string                     `json:"status" validate:"required,oneof=active inactive"`
}

func (h *GameTitleHandler) CreateGameTitle(c echo.Context) error {
	var req createGameTitleRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	// Convert attributes from request
	attributes := make([]usecase.GameTitleAttributeInput, len(req.Attributes))
	for i, attr := range req.Attributes {
		attributes[i] = usecase.GameTitleAttributeInput{
			Name:        attr.Name,
			Type:        attr.Type,
			Required:    attr.Required,
			Options:     attr.Options,
			Description: attr.Description,
		}
	}

	gameTitle, err := h.gameTitleUseCase.CreateGameTitle(c.Request().Context(), usecase.CreateGameTitleInput{
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		Banner:      req.Banner,
		Attributes:  attributes,
		Status:      req.Status,
	})

	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, gameTitle)
}

func (h *GameTitleHandler) GetGameTitle(c echo.Context) error {
	id := c.Param("id")
	gameTitle, err := h.gameTitleUseCase.GetGameTitleByID(c.Request().Context(), id)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, gameTitle)
}

func (h *GameTitleHandler) GetGameTitleBySlug(c echo.Context) error {
	slug := c.Param("slug")
	gameTitle, err := h.gameTitleUseCase.GetGameTitleBySlug(c.Request().Context(), slug)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, gameTitle)
}

func (h *GameTitleHandler) ListGameTitles(c echo.Context) error {
	status := c.QueryParam("status")
	pagination := utils.GetPaginationParams(c)

	gameTitles, total, err := h.gameTitleUseCase.ListGameTitles(
		c.Request().Context(),
		status,
		pagination.Page,
		pagination.PageSize,
	)

	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, gameTitles, total, pagination.Page, pagination.PageSize)
}

func (h *GameTitleHandler) UpdateGameTitle(c echo.Context) error {
	id := c.Param("id")
	
	var req createGameTitleRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	// Convert attributes from request
	attributes := make([]usecase.GameTitleAttributeInput, len(req.Attributes))
	for i, attr := range req.Attributes {
		attributes[i] = usecase.GameTitleAttributeInput{
			Name:        attr.Name,
			Type:        attr.Type,
			Required:    attr.Required,
			Options:     attr.Options,
			Description: attr.Description,
		}
	}

	gameTitle, err := h.gameTitleUseCase.UpdateGameTitle(c.Request().Context(), id, usecase.CreateGameTitleInput{
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,      
		Banner:      req.Banner,     
		Attributes:  attributes,
		Status:      req.Status,
	})

	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, gameTitle)
}

func (h *GameTitleHandler) DeleteGameTitle(c echo.Context) error {
	id := c.Param("id")
	
	if err := h.gameTitleUseCase.DeleteGameTitle(c.Request().Context(), id); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, map[string]interface{}{
		"message": "Game title deleted successfully",
	})
}