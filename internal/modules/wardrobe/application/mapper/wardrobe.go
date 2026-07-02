package mapper

import (
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

func MapToFashionItemRes(fi *entities.FashionItem) *dto.FashionItemRes {
	if fi == nil {
		return nil
	}
	var categoryName *string
	if fi.Category != nil {
		categoryName = &fi.Category.Name
	}
	return &dto.FashionItemRes{
		ID:              fi.ID,
		CategoryID:      fi.CategoryID,
		CategoryName:    categoryName,
		ImageUrl:        fi.ImageUrl,
		ImagePublicID:   fi.ImagePublicID,
		Color:           fi.Color,
		ColorHex:        fi.ColorHex,
		ColorHue:        fi.ColorHue,
		ColorSaturation: fi.ColorSaturation,
		ColorLightness:  fi.ColorLightness,
		Style:           fi.Style,
		Material:        fi.Material,
		Pattern:         fi.Pattern,
		Fit:             fi.Fit,
		Seasonality:     fi.Seasonality,
		Description:     fi.Description,
		CreatedAt:       fi.CreatedAt,
		UpdatedAt:       fi.UpdatedAt,
	}
}

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

	return &dto.WardrobeItemRes{
		ID:          item.ID,
		UserID:      item.UserID,
		Category:    categoryRes,
		Price:       item.Price,
		Status:      item.Status,
		FashionItem: MapToFashionItemRes(item.FashionItem),
		CreatedAt:   item.CreatedAt,
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

	return &dto.SearchWardrobeItemRes{
		ID:          item.ID,
		Category:    categoryRes,
		Price:       item.Price,
		IsSystem:    item.ItemType == 1,
		FashionItem: MapToFashionItemRes(item.FashionItem),
	}
}
