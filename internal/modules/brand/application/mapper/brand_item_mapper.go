package mapper

import (
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

func MapToFashionItemRes(fi *entities.FashionItem) *dto.FashionItemRes {
	if fi == nil {
		return nil
	}
	var categoryName *string
	if fi.Category != nil {
		categoryName = &fi.Category.Name
	}
	return &dto.FashionItemRes{
		ID:              fi.ID,
		CategoryID:      fi.CategoryID,
		CategoryName:    categoryName,
		ImageUrl:        fi.ImageUrl,
		ImagePublicID:   fi.ImagePublicID,
		Color:           fi.Color,
		ColorHex:        fi.ColorHex,
		ColorHue:        fi.ColorHue,
		ColorSaturation: fi.ColorSaturation,
		ColorLightness:  fi.ColorLightness,
		Style:           fi.Style,
		Material:        fi.Material,
		Pattern:         fi.Pattern,
		Fit:             fi.Fit,
		Seasonality:     fi.Seasonality,
		Description:     fi.Description,
		CreatedAt:       fi.CreatedAt,
		UpdatedAt:       fi.UpdatedAt,
	}
}

func MapToBrandItemRes(item *entities.BrandItem) *dto.BrandItemRes {
	if item == nil {
		return nil
	}
	var fashionItemID uuid.UUID
	if item.FashionItemID != nil {
		fashionItemID = *item.FashionItemID
	}
	return &dto.BrandItemRes{
		ID:            item.ID,
		BrandID:       item.BrandID,
		FashionItemID: fashionItemID,
		ProductCode:   item.ProductCode,
		Name:          item.Name,
		Description:   item.Description,
		Price:         item.Price,
		ItemType:      item.ItemType,
		Status:        item.Status,
		FashionItem:   MapToFashionItemRes(item.FashionItem),
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
	var fashionItemID uuid.UUID
	if item.FashionItemID != nil {
		fashionItemID = *item.FashionItemID
	}
	return &dto.BrandItemStylingDTO{
		ID:            item.ID,
		BrandID:       item.BrandID,
		BrandName:     brandName,
		FashionItemID: fashionItemID,
		ProductCode:   item.ProductCode,
		Name:          item.Name,
		ItemType:      item.ItemType,
		Status:        item.Status,
		FashionItem:   MapToFashionItemRes(item.FashionItem),
	}
}
