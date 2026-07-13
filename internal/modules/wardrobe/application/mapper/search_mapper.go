package mapper

import (
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"

	"github.com/google/uuid"
)

func MapToSearchDocumentDTO(item *entities.WardrobeItem) *dto.SearchDocumentDTO {
	if item == nil {
		return nil
	}

	var fashionItemID uuid.UUID
	if item.FashionItemID != nil {
		fashionItemID = *item.FashionItemID
	}
	return &dto.SearchDocumentDTO{
		ID:               item.ID,
		UserID:           item.UserID,
		FashionItemID:    fashionItemID,
		ItemType:         item.ItemType,
		ImageUrl:         "",
		ImagePublicID:    "",
		Color:            "",
		Style:            "",
		Material:         "",
		Pattern:          "",
		Fit:              "",
		Seasonality:      "",
		Description:      "",
		Status:           item.Status,
		CreatedAt:        item.CreatedAt,
		FashionCreatedAt: time.Time{},
		FashionUpdatedAt: time.Time{},
		Category:         dto.SearchDocumentCategoryDTO{},
		Price:            item.Price,
		ColorHex:         nil,
		ColorHue:         nil,
		ColorSaturation:  nil,
		ColorLightness:   nil,
	}
}
