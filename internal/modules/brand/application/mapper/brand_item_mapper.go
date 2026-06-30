package mapper

import (
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

func MapToBrandItemRes(item *entities.BrandItem) *dto.BrandItemRes {
	if item == nil {
		return nil
	}
	return &dto.BrandItemRes{
		ID:            item.ID,
		BrandID:       item.BrandID,
		FashionItemID: item.FashionItemID,
		ProductCode:   item.ProductCode,
		Name:          item.Name,
		Description:   item.Description,
		Price:         item.Price,
		ItemType:      item.ItemType,
		Status:        item.Status,
		FashionItem:   item.FashionItem,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func MapToDigitalSampleResponseRes(res *entities.DigitalSampleResponse) *dto.DigitalSampleResponseRes {
	if res == nil {
		return nil
	}
	var voteStr *string
	if res.VoteType != nil {
		s := string(*res.VoteType)
		voteStr = &s
	}
	return &dto.DigitalSampleResponseRes{
		ID:           res.ID,
		BrandItemID:  res.BrandItemID,
		UserID:       res.UserID,
		OutfitID:     res.OutfitID,
		VoteType:     voteStr,
		Rating:       res.Rating,
		FeedbackText: res.FeedbackText,
		CreatedAt:    res.CreatedAt,
	}
}

func MapToBrandItemStylingDTO(item *entities.BrandItem) *dto.BrandItemStylingDTO {
	if item == nil {
		return nil
	}
	brandName := ""
	if item.Brand != nil {
		brandName = item.Brand.Name
	}
	return &dto.BrandItemStylingDTO{
		ID:            item.ID,
		BrandID:       item.BrandID,
		BrandName:     brandName,
		FashionItemID: item.FashionItemID,
		ProductCode:   item.ProductCode,
		Name:          item.Name,
		ItemType:      item.ItemType,
		Status:        item.Status,
		FashionItem:   item.FashionItem,
	}
}
