package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"

	"github.com/google/uuid"
)

type CategoryContract struct {
	categoryUC uc_interfaces.ICategoryUseCase
}

func NewCategoryContract(categoryUC uc_interfaces.ICategoryUseCase) ICategoryContract {
	return &CategoryContract{
		categoryUC: categoryUC,
	}
}

func (c *CategoryContract) GetCategory(ctx context.Context, id uuid.UUID) (*dto.CategoryContractDTO, error) {
	return c.categoryUC.GetCategory(ctx, id)
}
