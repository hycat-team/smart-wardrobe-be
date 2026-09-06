package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"

	"github.com/google/uuid"
)

type ICategoryUseCase interface {
	GetCategory(ctx context.Context, id uuid.UUID) (*dto.CategoryContractDTO, error)
	ListCategories(ctx context.Context, ids []uuid.UUID) ([]*dto.CategoryContractDTO, error)
	GetAllCategories(ctx context.Context) ([]*dto.CategoryContractDTO, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*dto.CategoryContractDTO, error)
	GetCategoryByName(ctx context.Context, name string) (*dto.CategoryContractDTO, error)
	CreateCategory(ctx context.Context, input dto.CreateCategoryInput) (*dto.CategoryContractDTO, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, input dto.UpdateCategoryInput) (*dto.CategoryContractDTO, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
}
