package mapper

import (
	fashion_contract "smart-wardrobe-be/internal/modules/fashion/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

func MapToCategoryRes(category *fashion_contract.CategoryRes) *dto.CategoryRes {
	if category == nil {
		return nil
	}

	return &dto.CategoryRes{
		ID:        category.ID,
		Name:      category.Name,
		Slug:      category.Slug,
		SortOrder: category.SortOrder,
	}
}

func MapToCategoryResList(categories []*fashion_contract.CategoryRes) []*dto.CategoryRes {
	resList := make([]*dto.CategoryRes, len(categories))
	for i, category := range categories {
		resList[i] = MapToCategoryRes(category)
	}
	return resList
}
