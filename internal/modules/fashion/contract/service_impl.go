package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/modules/fashion/domain/entities"

	"github.com/google/uuid"
)

type FashionContract struct {
	fashionItemRepo repositories.IFashionItemRepository
	eventPublisher  event.IEventPublisher
}

func NewFashionContract(fashionItemRepo repositories.IFashionItemRepository, eventPublisher event.IEventPublisher) IFashionContract {
	return &FashionContract{
		fashionItemRepo: fashionItemRepo,
		eventPublisher:  eventPublisher,
	}
}

func mapToFashionItemRes(fi *entities.FashionItem) *FashionItemRes {
	if fi == nil {
		return nil
	}
	var categoryName *string
	if fi.Category != nil {
		categoryName = &fi.Category.Name
	}
	return &FashionItemRes{
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

func (c *FashionContract) GetFashionItem(ctx context.Context, id uuid.UUID) (*FashionItemRes, error) {
	item, err := c.fashionItemRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapToFashionItemRes(item), nil
}

func (c *FashionContract) ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*FashionItemRes, error) {
	items, err := c.fashionItemRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	res := make([]*FashionItemRes, len(items))
	for i, item := range items {
		res[i] = mapToFashionItemRes(item)
	}
	return res, nil
}
