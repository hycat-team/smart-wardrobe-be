package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/event"

	"github.com/google/uuid"
)

type FashionContract struct {
	fashionItemRepo repositories.IFashionItemRepository
	categoryRepo    repositories.ICategoryRepository
	eventPublisher  event.IEventPublisher
}

func NewFashionContract(fashionItemRepo repositories.IFashionItemRepository, categoryRepo repositories.ICategoryRepository, eventPublisher event.IEventPublisher) IFashionContract {
	return &FashionContract{
		fashionItemRepo: fashionItemRepo,
		categoryRepo:    categoryRepo,
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

func mapToCategoryRes(cat *entities.Category) *CategoryRes {
	if cat == nil {
		return nil
	}
	return &CategoryRes{
		ID:        cat.ID,
		Name:      cat.Name,
		Slug:      cat.Slug,
		SortOrder: cat.SortOrder,
	}
}

func (c *FashionContract) GetCategory(ctx context.Context, id uuid.UUID) (*CategoryRes, error) {
	cat, err := c.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapToCategoryRes(cat), nil
}

func (c *FashionContract) ListCategories(ctx context.Context, ids []uuid.UUID) ([]*CategoryRes, error) {
	items, err := c.categoryRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	res := make([]*CategoryRes, len(items))
	for i, item := range items {
		res[i] = mapToCategoryRes(item)
	}
	return res, nil
}
