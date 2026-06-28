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
	if item.FashionItem != nil && item.FashionItem.Category != nil {
		categoryRes = &dto.CategoryRes{
			ID:   item.FashionItem.Category.ID,
			Name: item.FashionItem.Category.Name,
			Slug: item.FashionItem.Category.Slug,
		}
	}
	fashion := item.FashionItem
	if fashion == nil {
		fashion = &entities.FashionItem{}
	}

	return &dto.WardrobeItemRes{
		ID:                    item.ID,
		UserID:                item.UserID,
		Category:              categoryRes,
		ImageUrl:              fashion.ImageUrl,
		ImagePublicID:         fashion.ImagePublicID,
		Color:                 stringutils.GetString(fashion.Color),
		ColorHex:              stringutils.GetString(fashion.ColorHex),
		ColorHue:              fashion.ColorHue,
		ColorSaturation:       fashion.ColorSaturation,
		ColorLightness:        fashion.ColorLightness,
		Style:                 stringutils.GetString(fashion.Style),
		Material:              stringutils.GetString(fashion.Material),
		Pattern:               stringutils.GetString(fashion.Pattern),
		Fit:                   stringutils.GetString(fashion.Fit),
		Seasonality:           stringutils.GetString(fashion.Seasonality),
		Price:                 item.Price,
		Status:                item.Status,
		ReviewReason:          fashion.ReviewReason,
		ProcessingErrorReason: fashion.ProcessingErrorReason,
		CreatedAt:             item.CreatedAt,
		// Description:   stringutils.GetString(item.Description),
	}
}

func MapToSearchWardrobeItemRes(item *entities.WardrobeItem) *dto.SearchWardrobeItemRes {
	if item == nil {
		return nil
	}

	var categoryRes *dto.CategoryRes
	if item.FashionItem != nil && item.FashionItem.Category != nil {
		categoryRes = &dto.CategoryRes{
			ID:   item.FashionItem.Category.ID,
			Name: item.FashionItem.Category.Name,
			Slug: item.FashionItem.Category.Slug,
		}
	}
	fashion := item.FashionItem
	if fashion == nil {
		fashion = &entities.FashionItem{}
	}

	return &dto.SearchWardrobeItemRes{
		ID:              item.ID,
		Category:        categoryRes,
		ImageUrl:        fashion.ImageUrl,
		ImagePublicID:   fashion.ImagePublicID,
		Color:           stringutils.GetString(fashion.Color),
		ColorHex:        stringutils.GetString(fashion.ColorHex),
		ColorHue:        fashion.ColorHue,
		ColorSaturation: fashion.ColorSaturation,
		ColorLightness:  fashion.ColorLightness,
		Style:           stringutils.GetString(fashion.Style),
		Material:        stringutils.GetString(fashion.Material),
		Pattern:         stringutils.GetString(fashion.Pattern),
		Fit:             stringutils.GetString(fashion.Fit),
		Seasonality:     stringutils.GetString(fashion.Seasonality),
		Price:           item.Price,
		IsSystem:        item.ItemType == 1,
		// Description:   stringutils.GetString(item.Description),
	}
}
