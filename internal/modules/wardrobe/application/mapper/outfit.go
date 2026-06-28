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
	itemDTOs = make([]*dto.OutfitItemRes, len(items))
	for idx, item := range items {
		var brandItemDTO *dto.BrandItemBriefRes
		if item.BrandItem != nil {
			brandName := ""
			if item.BrandItem.Brand != nil {
				brandName = item.BrandItem.Brand.Name
			}
			brandItemDTO = &dto.BrandItemBriefRes{
				ID:        item.BrandItem.ID,
				BrandID:   item.BrandItem.BrandID,
				BrandName: brandName,
				ItemType:  string(item.BrandItem.ItemType),
				Name:      item.BrandItem.Name,
				Price:     item.BrandItem.Price,
			}
		}

		itemDTOs[idx] = &dto.OutfitItemRes{
			ID:            item.FashionItemID,
			FashionItemID: item.FashionItemID,
			ItemContext:   string(item.ItemContext),
			WardrobeItem:  MapToWardrobeItemRes(item.WardrobeItem),
			BrandItem:     brandItemDTO,
			PositionX:     item.PositionX,
			PositionY:     item.PositionY,
			Scale:         item.Scale,
			LayerOrder:    item.LayerOrder,
		}
	}

	return &dto.OutfitRes{
		ID:            outfit.ID,
		UserID:        outfit.UserID,
		Name:          outfit.Name,
		Description:   outfit.Description,
		CoverImageUrl: outfit.CoverImageUrl,
		CoverPublicID: outfit.CoverPublicID,
		Status:        outfit.Status,
		CreatedAt:     outfit.CreatedAt,
		UpdatedAt:     outfit.UpdatedAt,
		Items:         itemDTOs,
	}
}
