package mapper

import (
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	fashion_entities "smart-wardrobe-be/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"
)

func MapToFashionItemRes(fi *fashion_entities.FashionItem) *dto.FashionItemRes {
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

	return &dto.WardrobeItemRes{
		ID:          item.ID,
		UserID:      item.UserID,
		Category:    nil, // TODO(Phase1): fetch category via fashion item by item.FashionItemID
		Price:       item.Price,
		Status:      item.Status,
		FashionItem: nil, // TODO(Phase1): fetch fashion item by item.FashionItemID
		CreatedAt:   item.CreatedAt,
	}
}

func MapToSearchWardrobeItemRes(item *entities.WardrobeItem) *dto.SearchWardrobeItemRes {
	if item == nil {
		return nil
	}

	return &dto.SearchWardrobeItemRes{
		ID:          item.ID,
		Category:    nil, // TODO(Phase1): fetch category via fashion item by item.FashionItemID
		Price:       item.Price,
		IsSystem:    item.ItemType == 1,
		FashionItem: nil, // TODO(Phase1): fetch fashion item by item.FashionItemID
	}
}
