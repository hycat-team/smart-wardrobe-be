package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

type ICategoryUseCase interface {
	GetCategories(ctx context.Context) ([]*dto.CategoryRes, error)
}
