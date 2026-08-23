package mapper

import (
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"

	"github.com/google/uuid"
)

func MapToOutfitRes(outfit *entities.Outfit, items []*entities.OutfitItem, fashionItemMap map[uuid.UUID]*fashion_dto.FashionItemBriefRes, availableFashionIDs map[uuid.UUID]struct{}) *dto.OutfitRes {
	if outfit == nil {
		return nil
	}

	var itemDTOs []*dto.OutfitItemRes
	itemDTOs = make([]*dto.OutfitItemRes, len(items))
	for idx, item := range items {
		fi := fashionItemMap[item.FashionItemID]
		var briefDTO *dto.FashionItemBriefRes
		if fi != nil {
			var category *dto.CategoryBriefRes
			if fi.Category != nil {
				category = &dto.CategoryBriefRes{
					ID:   fi.Category.ID,
					Name: fi.Category.Name,
					Slug: fi.Category.Slug,
				}
			}
			briefDTO = &dto.FashionItemBriefRes{
				ID:       fi.ID,
				ImageUrl: fi.ImageUrl,
				ColorHex: fi.ColorHex,
				Category: category,
			}
		}
		isAvailable := availableFashionIDs == nil
		if !isAvailable {
			_, isAvailable = availableFashionIDs[item.FashionItemID]
		}
		itemDTOs[idx] = &dto.OutfitItemRes{
			ID:          item.FashionItemID,
			ItemContext: string(item.ItemContext),
			FashionItem: briefDTO,
			PositionX:   item.PositionX,
			PositionY:   item.PositionY,
			Scale:       item.Scale,
			LayerOrder:  item.LayerOrder,
			IsAvailable: isAvailable,
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

func MapToOutfitBrief(outfit *entities.Outfit) *dto.OutfitBriefRes {
	if outfit == nil {
		return nil
	}
	return &dto.OutfitBriefRes{
		ID:            outfit.ID,
		Name:          outfit.Name,
		Description:   outfit.Description,
		CoverImageUrl: outfit.CoverImageUrl,
		Status:        outfit.Status,
		CreatedAt:     outfit.CreatedAt,
	}
}
