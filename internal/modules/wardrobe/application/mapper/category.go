package mapper

import (
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

func MapToCategoryRes(category *fashion_dto.CategoryRes) *dto.CategoryRes {
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

func MapToCategoryResList(categories []*fashion_dto.CategoryRes) []*dto.CategoryRes {
	resList := make([]*dto.CategoryRes, len(categories))
	for i, category := range categories {
		resList[i] = MapToCategoryRes(category)
	}
	return resList
}
