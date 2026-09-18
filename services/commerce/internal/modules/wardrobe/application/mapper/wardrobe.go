package mapper

import (
	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/entities"
	shared_dto "smart-wardrobe-be/shared/application/dto"
)

func MapToFashionItemRes(fi *fashion_dto.FashionItemContractDTO) *dto.FashionItemRes {
	if fi == nil {
		return nil
	}
	return &dto.FashionItemRes{
		ID:          fi.ID,
		Category:    fi.Category,
		ImageUrl:    fi.ImageUrl,
		Color:       fi.Color,
		ColorHex:    fi.ColorHex,
		Style:       fi.Style,
		Material:    fi.Material,
		Pattern:     fi.Pattern,
		Fit:         fi.Fit,
		Seasonality: fi.Seasonality,
		Description: fi.Description,
		CreatedAt:   fi.CreatedAt,
		UpdatedAt:   fi.UpdatedAt,
	}
}

func MapToWardrobeItemRes(item *entities.WardrobeItem, fi *fashion_dto.FashionItemContractDTO) *dto.WardrobeItemRes {
	if item == nil {
		return nil
	}

	res := &dto.WardrobeItemRes{
		ID:            item.ID,
		UserID:        item.UserID,
		Price:         item.Price,
		Status:        item.Status,
		CreatedAt:     item.CreatedAt,
		LastUsedAt:    item.LastUsedAt,
		TaskID:        item.TaskID,
		FashionItem:   MapToFashionItemRes(fi),
		FashionItemID: item.FashionItemID,
	}
	return res
}

func MapToSearchWardrobeItemRes(item *entities.WardrobeItem, fiBrief *fashion_dto.FashionItemBriefRes) *dto.SearchWardrobeItemRes {
	if item == nil {
		return nil
	}

	res := &dto.SearchWardrobeItemRes{
		ID:          item.ID,
		Price:       item.Price,
		IsSystem:    item.ItemType == itemtype.SystemCatalogItem,
		FashionItem: fiBrief,
	}
	return res
}

func MapToWardrobeItemBrief(item *entities.WardrobeItem, fiBrief *fashion_dto.FashionItemBriefRes, brandItem *shared_dto.BrandItemBriefRes) *dto.WardrobeItemBriefRes {
	if item == nil {
		return nil
	}
	return &dto.WardrobeItemBriefRes{
		ID:            item.ID,
		Price:         item.Price,
		Status:        item.Status,
		IsLocked:      false,
		LastUsedAt:    item.LastUsedAt,
		FashionItem:   fiBrief,
		FashionItemID: item.FashionItemID,
		BrandItem:     brandItem,
		TaskID:        item.TaskID,
	}
}

func MapToWardrobeItemContractDTO(item *entities.WardrobeItem) *dto.WardrobeItemContractDTO {
	if item == nil {
		return nil
	}
	return &dto.WardrobeItemContractDTO{
		ID:            item.ID,
		UserID:        item.UserID,
		Price:         item.Price,
		Status:        item.Status,
		FashionItemID: item.FashionItemID,
		CreatedAt:     item.CreatedAt,
		LastUsedAt:    item.LastUsedAt,
		IsDeleted:     item.IsDeleted,
		TaskID:        item.TaskID,
	}
}

func MapToWardrobeItemContractDTOList(items []*entities.WardrobeItem) []*dto.WardrobeItemContractDTO {
	if items == nil {
		return []*dto.WardrobeItemContractDTO{}
	}
	result := make([]*dto.WardrobeItemContractDTO, 0, len(items))
	for _, item := range items {
		if contractDTO := MapToWardrobeItemContractDTO(item); contractDTO != nil {
			result = append(result, contractDTO)
		}
	}
	return result
}
