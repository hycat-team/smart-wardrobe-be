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
		res.FashionItem = &wardrobeDto.FashionItemRes{
			ID:              fi.ID,
			CategoryID:      fi.CategoryID,
			CategoryName:    fi.CategoryName,
			CategorySlug:    fi.CategorySlug,
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
		if fi.CategoryID != nil && fi.CategoryName != nil {
			var slug string
			if fi.CategorySlug != nil {
				slug = *fi.CategorySlug
			}
			res.Category = &wardrobeDto.CategoryRes{
				ID:   *fi.CategoryID,
				Name: *fi.CategoryName,
				Slug: slug,
			}
		}
	}
	return &res
}
