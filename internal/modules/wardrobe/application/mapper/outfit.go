package mapper

import (
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

func MapToOutfitRes(outfit *entities.Outfit, items []*entities.OutfitItem) *dto.OutfitRes {
	if outfit == nil {
		return nil
	}

	var itemDTOs []*dto.OutfitItemRes
	if len(items) > 0 {
		itemDTOs = make([]*dto.OutfitItemRes, len(items))
		for idx, item := range items {
			itemDTOs[idx] = &dto.OutfitItemRes{
				ID:           item.ItemID,
				WardrobeItem: MapToWardrobeItemRes(item.Wardrobe),
				PositionX:    item.PositionX,
				PositionY:    item.PositionY,
				Scale:        item.Scale,
				LayerOrder:   item.LayerOrder,
			}
		}
	}

	return &dto.OutfitRes{
		ID:            outfit.ID,
		UserID:        outfit.UserID,
		Name:          outfit.Name,
		Description:   outfit.Description,
		CoverImageUrl: outfit.CoverImageUrl,
		Status:        outfit.Status,
		CreatedAt:     outfit.CreatedAt,
		UpdatedAt:     outfit.UpdatedAt,
		Items:         itemDTOs,
	}
}
