package mapper

import (
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"

	"github.com/google/uuid"
)

func MapToOutfitRes(outfit *entities.Outfit, items []*entities.OutfitItem, fashionItemMap map[uuid.UUID]*fashion_dto.FashionItemBriefRes) *dto.OutfitRes {
	if outfit == nil {
		return nil
	}

	var itemDTOs []*dto.OutfitItemRes
	itemDTOs = make([]*dto.OutfitItemRes, len(items))
	for idx, item := range items {
		fi := fashionItemMap[item.FashionItemID]
		var briefDTO *dto.FashionItemBriefRes
		if fi != nil {
			briefDTO = &dto.FashionItemBriefRes{
				ID:           fi.ID,
				ImageUrl:     fi.ImageUrl,
				ColorHex:     fi.ColorHex,
				CategoryName: fi.CategoryName,
				CategorySlug: fi.CategorySlug,
			}
		}
		itemDTOs[idx] = &dto.OutfitItemRes{
			ID:          item.FashionItemID,
			ItemContext: string(item.ItemContext),
			FashionItem: briefDTO,
			PositionX:   item.PositionX,
			PositionY:   item.PositionY,
			Scale:       item.Scale,
			LayerOrder:  item.LayerOrder,
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
