package contract

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/mapper"

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

func (c *FashionContract) GetFashionItem(ctx context.Context, id uuid.UUID) (*dto.FashionItemContractDTO, error) {
	return c.fashionItemUC.GetFashionItem(ctx, id)
}

func (c *FashionContract) ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*dto.FashionItemContractDTO, error) {
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
			ID:                    item.ID,
			ImageUrl:              item.ImageUrl,
			Color:                 item.Color,
			ColorHex:              item.ColorHex,
			Category:              item.Category,
			Style:                 item.Style,
			ReviewReason:          item.ReviewReason,
			ProcessingErrorReason: item.ProcessingErrorReason,
		}
	}
	return res, nil
}

func (c *FashionContract) CreateFashionItem(ctx context.Context, input dto.CreateFashionItemInput) error {
	return c.fashionItemUC.CreateFashionItem(ctx, input)
}

func (c *FashionContract) BulkCreateFashionItems(ctx context.Context, inputs []dto.CreateFashionItemInput) error {
	return c.fashionItemUC.BulkCreateFashionItems(ctx, inputs)
}

func (c *FashionContract) UpdateFashionItemCategory(ctx context.Context, fashionItemID uuid.UUID, categoryID uuid.UUID) error {
	return c.fashionItemUC.UpdateFashionItemCategory(ctx, fashionItemID, categoryID)
}

func (c *FashionContract) GetStaleProcessingInfo(ctx context.Context, fashionItemIDs []uuid.UUID) ([]dto.FashionProcessingInfoDTO, error) {
	return c.fashionItemUC.GetStaleProcessingInfo(ctx, fashionItemIDs)
}

func (c *FashionContract) SearchFashionItemIDs(ctx context.Context, query *string, categorySlug *string) ([]uuid.UUID, error) {
	return c.fashionItemUC.SearchFashionItemIDs(ctx, query, categorySlug)
}

func (c *FashionContract) ResetProcessingState(ctx context.Context, fashionItemID uuid.UUID) error {
	return c.fashionItemUC.ResetProcessingState(ctx, fashionItemID)
}

func (c *FashionContract) SetProcessingErrorReason(ctx context.Context, fashionItemID uuid.UUID, reason string) error {
	return c.fashionItemUC.SetProcessingErrorReason(ctx, fashionItemID, reason)
}

func (c *FashionContract) IsWearableCategory(ctx context.Context, categoryID uuid.UUID) (bool, error) {
	category, err := c.categoryUC.GetCategory(ctx, categoryID)
	if err != nil {
		return false, err
	}
	if category == nil {
		return false, nil
	}
	return mapper.CategorySlugToWearingRole(category.Slug) != "other", nil
}
