package mapper

import (
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"
)

func MapToFashionItemRes(fi *fashion_dto.FashionItemRes) *dto.FashionItemRes {
	if fi == nil {
		return nil
	}
	return &dto.FashionItemRes{
		ID:              fi.ID,
		CategoryID:      fi.CategoryID,
		CategoryName:    fi.CategoryName,
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

func MapToWardrobeItemRes(item *entities.WardrobeItem, fi *fashion_dto.FashionItemRes) *dto.WardrobeItemRes {
	if item == nil {
		return nil
	}

	res := &dto.WardrobeItemRes{
		ID:         item.ID,
		UserID:     item.UserID,
		Price:      item.Price,
		Status:     item.Status,
		CreatedAt:  item.CreatedAt,
		LastUsedAt: item.LastUsedAt,
		IsDeleted:  item.IsDeleted,
	}
	if fi != nil {
		res.FashionItem = MapToFashionItemRes(fi)
		if fi.CategoryName != nil {
			res.Category = &dto.CategoryRes{
				ID:   *fi.CategoryID,
				Name: *fi.CategoryName,
			}
		}
	}
	return res
}

func MapToSearchWardrobeItemRes(item *entities.WardrobeItem, fi *fashion_dto.FashionItemRes) *dto.SearchWardrobeItemRes {
	if item == nil {
		return nil
	}

	res := &dto.SearchWardrobeItemRes{
		ID:       item.ID,
		Price:    item.Price,
		IsSystem: item.ItemType == 1,
	}
	if fi != nil {
		res.FashionItem = MapToFashionItemRes(fi)
		if fi.CategoryName != nil {
			res.Category = &dto.CategoryRes{
				ID:   *fi.CategoryID,
				Name: *fi.CategoryName,
			}
		}
	}
	return res
}
