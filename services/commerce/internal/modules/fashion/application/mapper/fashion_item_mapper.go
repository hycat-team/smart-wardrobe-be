package mapper

import (
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
)

// MapToFashionItemContractDTO maps a fashion item entity to its cross-module
// contract DTO, including the processing/review reasons surfaced to brand portal
// (feature 023, US4).
func MapToFashionItemContractDTO(fi *entities.FashionItem) *dto.FashionItemContractDTO {
	if fi == nil {
		return nil
	}
	var category *dto.CategoryBriefRes
	if fi.Category != nil {
		category = &dto.CategoryBriefRes{
			ID:   fi.Category.ID,
			Name: fi.Category.Name,
			Slug: fi.Category.Slug,
		}
	}
	return &dto.FashionItemContractDTO{
		ID:                    fi.ID,
		Category:              category,
		ImageUrl:              fi.ImageUrl,
		ImagePublicID:         fi.ImagePublicID,
		Color:                 fi.Color,
		ColorHex:              fi.ColorHex,
		ColorHue:              fi.ColorHue,
		ColorSaturation:       fi.ColorSaturation,
		ColorLightness:        fi.ColorLightness,
		Style:                 fi.Style,
		Material:              fi.Material,
		Pattern:               fi.Pattern,
		Fit:                   fi.Fit,
		Seasonality:           fi.Seasonality,
		Description:           fi.Description,
		ProcessingErrorReason: fi.ProcessingErrorReason,
		ReviewReason:          fi.ReviewReason,
		CreatedAt:             fi.CreatedAt,
		UpdatedAt:             fi.UpdatedAt,
	}
}
