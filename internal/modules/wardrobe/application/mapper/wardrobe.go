package mapper

import (
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/utils/stringutils"
)

func MapToWardrobeItemRes(item *entities.WardrobeItem) *dto.WardrobeItemRes {
	if item == nil {
		return nil
	}

	var categoryRes *dto.CategoryRes
	if item.Category != nil {
		categoryRes = &dto.CategoryRes{
			ID:   item.Category.ID,
			Name: item.Category.Name,
			Slug: item.Category.Slug,
		}
	}

	return &dto.WardrobeItemRes{
		ID:            item.ID,
		UserID:        item.UserID,
		Category:      categoryRes,
		ImageUrl:      item.ImageUrl,
		ImagePublicID: item.ImagePublicID,
		Color:         stringutils.GetString(item.Color),
		Style:         stringutils.GetString(item.Style),
		Material:      stringutils.GetString(item.Material),
		Pattern:       stringutils.GetString(item.Pattern),
		Fit:           stringutils.GetString(item.Fit),
		Seasonality:   stringutils.GetString(item.Seasonality),
		Status:        item.Status,
		CreatedAt:     item.CreatedAt,
		// Description:   stringutils.GetString(item.Description),
	}
}

func MapToSearchWardrobeItemRes(item *entities.WardrobeItem) *dto.SearchWardrobeItemRes {
	if item == nil {
		return nil
	}

	var categoryRes *dto.CategoryRes
	if item.Category != nil {
		categoryRes = &dto.CategoryRes{
			ID:   item.Category.ID,
			Name: item.Category.Name,
			Slug: item.Category.Slug,
		}
	}

	return &dto.SearchWardrobeItemRes{
		ID:            item.ID,
		Category:      categoryRes,
		ImageUrl:      item.ImageUrl,
		ImagePublicID: item.ImagePublicID,
		Color:         stringutils.GetString(item.Color),
		Style:         stringutils.GetString(item.Style),
		Material:      stringutils.GetString(item.Material),
		Pattern:       stringutils.GetString(item.Pattern),
		Fit:           stringutils.GetString(item.Fit),
		Seasonality:   stringutils.GetString(item.Seasonality),
		IsSystem:      item.ItemType == 1,
		// Description:   stringutils.GetString(item.Description),
	}
}
