package mapper

import (
	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
)

func MapToWardrobeItemResWithFashion(item *wardrobeDto.WardrobeItemRes, fi *dto.FashionItemRes) *wardrobeDto.WardrobeItemRes {
	if item == nil {
		return nil
	}
	res := *item
	if fi != nil {
		var category *wardrobeDto.CategoryBriefRes
		if fi.Category != nil {
			category = &wardrobeDto.CategoryBriefRes{
				ID:   fi.Category.ID,
				Name: fi.Category.Name,
				Slug: fi.Category.Slug,
			}
		}
		res.FashionItem = &wardrobeDto.FashionItemRes{
			ID:              fi.ID,
			Category:        category,
			ImageUrl:        fi.ImageUrl,
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
	return &res
}
