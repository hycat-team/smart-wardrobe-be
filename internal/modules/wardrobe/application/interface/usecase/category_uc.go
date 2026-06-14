package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

type ICategoryUseCase interface {
	GetCategories(ctx context.Context) ([]*dto.CategoryRes, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*dto.CategoryRes, error)
	CreateCategory(ctx context.Context, input dto.CreateCategoryReq) (*dto.CategoryRes, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, input dto.UpdateCategoryReq) (*dto.CategoryRes, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
}
