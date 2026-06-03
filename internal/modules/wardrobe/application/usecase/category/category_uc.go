package category

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/pkg/logger"
)

type CategoryUseCase struct {
	logger       logger.Interface
	categoryRepo repositories.ICategoryRepository
}

func NewCategoryUseCase(
	l logger.Interface,
	categoryRepo repositories.ICategoryRepository,
) uc_interfaces.ICategoryUseCase {
	return &CategoryUseCase{
		logger:       l,
		categoryRepo: categoryRepo,
	}
}

func (uc *CategoryUseCase) GetCategories(ctx context.Context) ([]*dto.CategoryRes, error) {
	categories, err := uc.categoryRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	return mapper.MapToCategoryResList(categories), nil
}
