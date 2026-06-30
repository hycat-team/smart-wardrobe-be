package mapper

import (
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/utils/stringutils"
)

func MapToSearchDocumentDTO(item *entities.WardrobeItem) *dto.SearchDocumentDTO {
	if item == nil {
		return nil
	}
	fashion := item.FashionItem
	if fashion == nil {
		fashion = &entities.FashionItem{}
	}

	var categoryIDStr string
	if fashion.CategoryID != nil {
		categoryIDStr = fashion.CategoryID.String()
	}

	categoryDTO := dto.SearchDocumentCategoryDTO{
		ID: categoryIDStr,
	}
	if fashion.Category != nil {
		categoryDTO.Name = fashion.Category.Name
		categoryDTO.Slug = fashion.Category.Slug
	}

	return &dto.SearchDocumentDTO{
		ID:              item.ID,
		UserID:          item.UserID,
		FashionItemID:   item.FashionItemID,
		ItemType:        item.ItemType,
		ImageUrl:        fashion.ImageUrl,
		ImagePublicID:   fashion.ImagePublicID,
		Color:           stringutils.GetString(fashion.Color),
		Style:           stringutils.GetString(fashion.Style),
		Material:        stringutils.GetString(fashion.Material),
		Pattern:         stringutils.GetString(fashion.Pattern),
		Fit:             stringutils.GetString(fashion.Fit),
		Seasonality:     stringutils.GetString(fashion.Seasonality),
		Description:     stringutils.GetString(fashion.Description),
		Status:          item.Status,
		CreatedAt:       item.CreatedAt,
		Category:        categoryDTO,
		Price:           item.Price,
		ColorHex:        fashion.ColorHex,
		ColorHue:        fashion.ColorHue,
		ColorSaturation: fashion.ColorSaturation,
		ColorLightness:  fashion.ColorLightness,
	}
}
