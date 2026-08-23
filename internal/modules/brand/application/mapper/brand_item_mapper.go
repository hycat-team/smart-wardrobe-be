package mapper

import (
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"

	"github.com/google/uuid"
)

func MapToBrandItemRes(item *entities.BrandItem, fi *fashion_dto.FashionItemRes) *dto.BrandItemRes {
	if item == nil {
		return nil
	}
	brandName := ""
	if item.Brand != nil {
		brandName = item.Brand.Name
	}
	var fashionItemID uuid.UUID
	if item.FashionItemID != nil {
		fashionItemID = *item.FashionItemID
	}
	return &dto.BrandItemRes{
		ID:            item.ID,
		BrandID:       item.BrandID,
		BrandName:     brandName,
		FashionItemID: fashionItemID,
		ProductCode:   item.ProductCode,
		Name:          item.Name,
		Description:   item.Description,
		Price:         item.Price,
		ItemType:      item.ItemType,
		Status:        item.Status,
		FashionItem:   fi,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
		TaskID:        item.TaskID,
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

func MapBrandItemBrief(item *entities.BrandItem, brand *entities.Brand, fi *fashion_dto.FashionItemBriefRes) *dto.BrandItemBriefRes {
	if item == nil {
		return nil
	}
	return &dto.BrandItemBriefRes{
		ID:          item.ID,
		Brand:       MapBrandBrief(brand),
		Name:        item.Name,
		Price:       item.Price,
		ItemType:    item.ItemType,
		Status:      item.Status,
		FashionItem: fi,
	}
}

func MapDigitalSampleResponseWithUser(res *entities.DigitalSampleResponse, userBrief *identity_dto.UserBriefRes) *dto.DigitalSampleResponseRes {
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
