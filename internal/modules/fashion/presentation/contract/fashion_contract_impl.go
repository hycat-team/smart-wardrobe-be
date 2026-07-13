package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"

	"github.com/google/uuid"
)

type FashionContract struct {
	fashionItemUC uc_interfaces.IFashionItemUseCase
	categoryUC    uc_interfaces.ICategoryUseCase
}

func NewFashionContract(fashionItemUC uc_interfaces.IFashionItemUseCase, categoryUC uc_interfaces.ICategoryUseCase) IFashionContract {
	return &FashionContract{
		fashionItemUC: fashionItemUC,
		categoryUC:    categoryUC,
	}
}

func (c *FashionContract) GetFashionItem(ctx context.Context, id uuid.UUID) (*dto.FashionItemRes, error) {
	return c.fashionItemUC.GetFashionItem(ctx, id)
}

func (c *FashionContract) ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*dto.FashionItemRes, error) {
	return c.fashionItemUC.ListFashionItems(ctx, ids)
}

func (c *FashionContract) ListFashionItemsBrief(ctx context.Context, ids []uuid.UUID) ([]*dto.FashionItemBriefRes, error) {
	items, err := c.fashionItemUC.ListFashionItems(ctx, ids)
	if err != nil {
		return nil, err
	}
	res := make([]*dto.FashionItemBriefRes, len(items))
	for i, item := range items {
		res[i] = &dto.FashionItemBriefRes{
			ID:           item.ID,
			ImageUrl:     item.ImageUrl,
			ColorHex:     item.ColorHex,
			CategoryName: item.CategoryName,
			CategorySlug: item.CategorySlug,
		}
	}
	return res, nil
}

func (c *FashionContract) GetCategory(ctx context.Context, id uuid.UUID) (*dto.CategoryRes, error) {
	return c.categoryUC.GetCategory(ctx, id)
}

func (c *FashionContract) ListCategories(ctx context.Context, ids []uuid.UUID) ([]*dto.CategoryRes, error) {
	return c.categoryUC.ListCategories(ctx, ids)
}

func (c *FashionContract) GetAllCategories(ctx context.Context) ([]*dto.CategoryRes, error) {
	return c.categoryUC.GetAllCategories(ctx)
}

func (c *FashionContract) GetCategoryBySlug(ctx context.Context, slug string) (*dto.CategoryRes, error) {
	return c.categoryUC.GetCategoryBySlug(ctx, slug)
}

func (c *FashionContract) GetCategoryByName(ctx context.Context, name string) (*dto.CategoryRes, error) {
	return c.categoryUC.GetCategoryByName(ctx, name)
}

func (c *FashionContract) CreateCategory(ctx context.Context, input dto.CreateCategoryInput) (*dto.CategoryRes, error) {
	return c.categoryUC.CreateCategory(ctx, input)
}

func (c *FashionContract) UpdateCategory(ctx context.Context, id uuid.UUID, input dto.UpdateCategoryInput) (*dto.CategoryRes, error) {
	return c.categoryUC.UpdateCategory(ctx, id, input)
}

func (c *FashionContract) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return c.categoryUC.DeleteCategory(ctx, id)
}
