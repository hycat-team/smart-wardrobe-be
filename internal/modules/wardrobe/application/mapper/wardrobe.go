package mapper

import (
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"
)

func MapToFashionItemRes(fi *fashion_dto.FashionItemRes) *dto.FashionItemRes {
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
	return &dto.FashionItemRes{
		ID:          fi.ID,
		Category:    category,
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

func MapToWardrobeItemRes(item *entities.WardrobeItem, fi *fashion_dto.FashionItemRes) *dto.WardrobeItemRes {
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
		FashionItem: MapToFashionItemBrief(fiBrief),
	}
	return res
}

func MapToFashionItemBrief(fi *fashion_dto.FashionItemBriefRes) *dto.FashionItemBriefRes {
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
	return &dto.FashionItemBriefRes{
		ID:       fi.ID,
		ImageUrl: fi.ImageUrl,
		Color:    fi.Color,
		ColorHex: fi.ColorHex,
		Style:    fi.Style,
		Category: category,
	}
}

func MapToWardrobeItemBrief(item *entities.WardrobeItem, fiBrief *fashion_dto.FashionItemBriefRes, brandItem *dto.BrandItemBriefRes) *dto.WardrobeItemBriefRes {
	if item == nil {
		return nil
	}
	return &dto.WardrobeItemBriefRes{
		ID:            item.ID,
		Price:         item.Price,
		Status:        item.Status,
		LastUsedAt:    item.LastUsedAt,
		FashionItem:   MapToFashionItemBrief(fiBrief),
		FashionItemID: item.FashionItemID,
		BrandItem:     brandItem,
		TaskID:        item.TaskID,
	}
}
