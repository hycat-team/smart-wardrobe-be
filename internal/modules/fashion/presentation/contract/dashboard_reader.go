package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"

	"github.com/google/uuid"
)

type IFashionDashboardReader interface {
	GetCategoriesByIDs(ctx context.Context, categoryIDs []uuid.UUID) ([]dto.DashboardCategoryDTO, error)
	GetCategoryIDsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]dto.DashboardFashionItemCategoryDTO, error)
}

type FashionDashboardReader struct {
	categoryUC    uc_interfaces.ICategoryUseCase
	fashionItemUC uc_interfaces.IFashionItemUseCase
}

func NewFashionDashboardReader(
	categoryUC uc_interfaces.ICategoryUseCase,
	fashionItemUC uc_interfaces.IFashionItemUseCase,
) IFashionDashboardReader {
	return &FashionDashboardReader{
		categoryUC:    categoryUC,
		fashionItemUC: fashionItemUC,
	}
}

func (r *FashionDashboardReader) GetCategoriesByIDs(ctx context.Context, categoryIDs []uuid.UUID) ([]dto.DashboardCategoryDTO, error) {
	categories, err := r.categoryUC.ListCategories(ctx, categoryIDs)
	if err != nil {
		return nil, err
	}
	result := make([]dto.DashboardCategoryDTO, 0, len(categories))
	for _, c := range categories {
		result = append(result, dto.DashboardCategoryDTO{ID: c.ID, Name: c.Name})
	}
	return result, nil
}

func (r *FashionDashboardReader) GetCategoryIDsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]dto.DashboardFashionItemCategoryDTO, error) {
	return r.fashionItemUC.GetCategoryIDsByFashionItemIDs(ctx, fashionItemIDs)
}
