package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"

	"github.com/google/uuid"
)

type ICategoryUseCase interface {
	GetCategory(ctx context.Context, id uuid.UUID) (*dto.CategoryRes, error)
	ListCategories(ctx context.Context, ids []uuid.UUID) ([]*dto.CategoryRes, error)
	GetAllCategories(ctx context.Context) ([]*dto.CategoryRes, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*dto.CategoryRes, error)
	GetCategoryByName(ctx context.Context, name string) (*dto.CategoryRes, error)
	CreateCategory(ctx context.Context, input dto.CreateCategoryInput) (*dto.CategoryRes, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, input dto.UpdateCategoryInput) (*dto.CategoryRes, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
}
