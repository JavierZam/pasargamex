package usecase

import (
	"context"
	"strings"
	"time"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/errors"
)

type GameTitleUseCase struct {
	gameTitleRepo repository.GameTitleRepository
}

func NewGameTitleUseCase(gameTitleRepo repository.GameTitleRepository) *GameTitleUseCase {
	return &GameTitleUseCase{
		gameTitleRepo: gameTitleRepo,
	}
}

type CreateGameTitleInput struct {
    Name        string
    Description string
    Icon        string
    Banner      string
    Attributes  []GameTitleAttributeInput
    Status      string
}

type GameTitleAttributeInput struct {
	Name        string
	Type        string
	Required    bool
	Options     []string
	Description string
}

func (uc *GameTitleUseCase) CreateGameTitle(ctx context.Context, input CreateGameTitleInput) (*entity.GameTitle, error) {
	// Generate slug from name
	slug := strings.ToLower(strings.ReplaceAll(input.Name, " ", "-"))

	// Check if slug already exists
	existing, err := uc.gameTitleRepo.GetBySlug(ctx, slug)
	if err == nil && existing != nil {
		return nil, errors.BadRequest("Game title with this name already exists", nil)
	}

	// Convert attributes
	attributes := make([]entity.GameTitleAttribute, len(input.Attributes))
	for i, attr := range input.Attributes {
		attributes[i] = entity.GameTitleAttribute{
			Name:        attr.Name,
			Type:        attr.Type,
			Required:    attr.Required,
			Options:     attr.Options,
			Description: attr.Description,
		}
	}

	// Create game title entity
    gameTitle := &entity.GameTitle{
        Name:        input.Name,
        Slug:        slug,
        Description: input.Description,
        Icon:        input.Icon,
        Banner:      input.Banner,
        Attributes:  attributes,
        Status:      input.Status,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }

	// Save to repository
	if err := uc.gameTitleRepo.Create(ctx, gameTitle); err != nil {
		return nil, err
	}

	return gameTitle, nil
}

func (uc *GameTitleUseCase) GetGameTitleByID(ctx context.Context, id string) (*entity.GameTitle, error) {
	return uc.gameTitleRepo.GetByID(ctx, id)
}

func (uc *GameTitleUseCase) GetGameTitleBySlug(ctx context.Context, slug string) (*entity.GameTitle, error) {
	return uc.gameTitleRepo.GetBySlug(ctx, slug)
}

func (uc *GameTitleUseCase) ListGameTitles(ctx context.Context, status string, page, limit int) ([]*entity.GameTitle, int64, error) {
    // Prepare filter
    filter := make(map[string]interface{})
    
    // Jika status tidak diberikan, set default ke "active"
    if status == "" {
        status = "active"
    }
    
    filter["status"] = status

    // Calculate offset
    offset := (page - 1) * limit
    if offset < 0 {
        offset = 0
    }

    return uc.gameTitleRepo.List(ctx, filter, limit, offset)
}

func (uc *GameTitleUseCase) UpdateGameTitle(ctx context.Context, id string, input CreateGameTitleInput) (*entity.GameTitle, error) {
	// Get existing game title
	gameTitle, err := uc.gameTitleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields
    gameTitle.Name = input.Name
    gameTitle.Description = input.Description
    gameTitle.Icon = input.Icon
    gameTitle.Banner = input.Banner
    gameTitle.Status = input.Status
	
	// Convert attributes
	attributes := make([]entity.GameTitleAttribute, len(input.Attributes))
	for i, attr := range input.Attributes {
		attributes[i] = entity.GameTitleAttribute{
			Name:        attr.Name,
			Type:        attr.Type,
			Required:    attr.Required,
			Options:     attr.Options,
			Description: attr.Description,
		}
	}
	gameTitle.Attributes = attributes
	
	gameTitle.UpdatedAt = time.Now()

	// Save to repository
	if err := uc.gameTitleRepo.Update(ctx, gameTitle); err != nil {
		return nil, err
	}

	return gameTitle, nil
}

func (uc *GameTitleUseCase) DeleteGameTitle(ctx context.Context, id string) error {
	// Check if game title exists
	_, err := uc.gameTitleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return uc.gameTitleRepo.Delete(ctx, id)
}