package mapper

import (
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"

	"github.com/google/uuid"
)

func MapToBrandItemRes(item *entities.BrandItem, fi *fashion_dto.FashionItemContractDTO) *dto.BrandItemRes {
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
	var fiID uuid.UUID
	if item.FashionItemID != nil {
		fiID = *item.FashionItemID
	}
	return &dto.BrandItemBriefRes{
		ID:            item.ID,
		FashionItemID: fiID,
		Brand:         MapBrandBrief(brand),
		ProductCode:   item.ProductCode,
		Name:          item.Name,
		Price:         item.Price,
		ItemType:      item.ItemType,
		Status:        item.Status,
		FashionItem:   fi,
	}
}

func MapDigitalSampleResponseWithUser(res *entities.DigitalSampleResponse, userBrief *identity_dto.UserBriefContractDTO) *dto.DigitalSampleResponseRes {
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
		User:         MapIdentityUserBrief(userBrief),
		CreatedAt:    res.CreatedAt,
	}
}

func MapToBrandItemContractDTO(item *entities.BrandItem, brandName string) *dto.BrandItemContractDTO {
	if item == nil {
		return nil
	}
	var fashionItemID uuid.UUID
	if item.FashionItemID != nil {
		fashionItemID = *item.FashionItemID
	}
	name := brandName
	if name == "" && item.Brand != nil {
		name = item.Brand.Name
	}
	return &dto.BrandItemContractDTO{
		ID:            item.ID,
		BrandID:       item.BrandID,
		BrandName:     name,
		FashionItemID: fashionItemID,
		ProductCode:   item.ProductCode,
		Name:          item.Name,
		Description:   item.Description,
		Price:         item.Price,
		ItemType:      item.ItemType,
		Status:        item.Status,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
		TaskID:        item.TaskID,
	}
}

func MapToBrandItemContractDTOList(items []*entities.BrandItem, brandNameMap map[uuid.UUID]string) []*dto.BrandItemContractDTO {
	if items == nil {
		return []*dto.BrandItemContractDTO{}
	}
	result := make([]*dto.BrandItemContractDTO, 0, len(items))
	for _, item := range items {
		brandName := ""
		if brandNameMap != nil {
			brandName = brandNameMap[item.BrandID]
		}
		if contractDTO := MapToBrandItemContractDTO(item, brandName); contractDTO != nil {
			result = append(result, contractDTO)
		}
	}
	return result
}
