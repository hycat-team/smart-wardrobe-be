package mapper

import (
	"time"

	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"

	"github.com/google/uuid"
)

func MapToSearchDocumentDTO(item *entities.WardrobeItem, fashionItem *fashion_dto.FashionItemRes) *dto.SearchDocumentDTO {
	if item == nil {
		return nil
	}

	var fashionItemID uuid.UUID
	if item.FashionItemID != nil {
		fashionItemID = *item.FashionItemID
	}

	var imageUrl string
	var imagePublicID string
	var color string
	var style string
	var material string
	var pattern string
	var fit string
	var seasonality string
	var description string
	var fashionCreatedAt time.Time
	var fashionUpdatedAt time.Time
	var category dto.SearchDocumentCategoryDTO
	var colorHex *string
	var colorHue *float64
	var colorSaturation *float64
	var colorLightness *float64

	if fashionItem != nil {
		fashionItemID = fashionItem.ID
		imageUrl = fashionItem.ImageUrl
		imagePublicID = fashionItem.ImagePublicID
		if fashionItem.Color != nil {
			color = *fashionItem.Color
		}
		colorHex = fashionItem.ColorHex
		colorHue = fashionItem.ColorHue
		colorSaturation = fashionItem.ColorSaturation
		colorLightness = fashionItem.ColorLightness
		if fashionItem.Style != nil {
			style = *fashionItem.Style
		}
		if fashionItem.Material != nil {
			material = *fashionItem.Material
		}
		if fashionItem.Pattern != nil {
			pattern = *fashionItem.Pattern
		}
		if fashionItem.Fit != nil {
			fit = *fashionItem.Fit
		}
		if fashionItem.Seasonality != nil {
			seasonality = *fashionItem.Seasonality
		}
		if fashionItem.Description != nil {
			description = *fashionItem.Description
		}
		fashionCreatedAt = fashionItem.CreatedAt
		fashionUpdatedAt = fashionItem.UpdatedAt
		if fashionItem.Category != nil {
			category.ID = fashionItem.Category.ID.String()
			category.Name = fashionItem.Category.Name
			category.Slug = fashionItem.Category.Slug
		}
	}

	return &dto.SearchDocumentDTO{
		ID:               item.ID,
		UserID:           item.UserID,
		FashionItemID:    fashionItemID,
		ItemType:         item.ItemType,
		ImageUrl:         imageUrl,
		ImagePublicID:    imagePublicID,
		Color:            color,
		Style:            style,
		Material:         material,
		Pattern:          pattern,
		Fit:              fit,
		Seasonality:      seasonality,
		Description:      description,
		Status:           item.Status,
		CreatedAt:        item.CreatedAt,
		FashionCreatedAt: fashionCreatedAt,
		FashionUpdatedAt: fashionUpdatedAt,
		Category:         category,
		Price:            item.Price,
		ColorHex:         colorHex,
		ColorHue:         colorHue,
		ColorSaturation:  colorSaturation,
		ColorLightness:   colorLightness,
	}
}
