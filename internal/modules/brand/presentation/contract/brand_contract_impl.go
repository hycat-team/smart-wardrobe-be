package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"

	"github.com/google/uuid"
)

type BrandContract struct {
	benefitUC uc_interfaces.IBrandBenefitUseCase
	itemUC    uc_interfaces.IBrandItemContractUseCase
}

func NewBrandContract(
	benefitUC uc_interfaces.IBrandBenefitUseCase,
	itemUC uc_interfaces.IBrandItemContractUseCase,
) IBrandContract {
	return &BrandContract{
		benefitUC: benefitUC,
		itemUC:    itemUC,
	}
}

func (c *BrandContract) ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, req *dto.ListEligibleBrandItemsReq) ([]*dto.BrandItemRes, []*fashion_dto.FashionItemRes, error) {
	items, err := c.itemUC.ListEligibleBrandItemsForStyling(ctx, userID, req)
	if err != nil {
		return nil, nil, err
	}
	return items, nil, nil
}

func (c *BrandContract) CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *dto.BrandItemRes, *fashion_dto.FashionItemRes, error) {
	eligible, item, err := c.itemUC.CheckBrandItemEligibility(ctx, userID, fashionItemID)
	if err != nil {
		return false, nil, nil, err
	}
	return eligible, item, nil, nil
}

func (c *BrandContract) ListActiveBrandItemFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	return c.itemUC.ListActiveBrandItemFashionIDs(ctx, fashionItemIDs)
}

func (c *BrandContract) GetBrandItemByID(ctx context.Context, brandItemID uuid.UUID) (*dto.BrandItemRes, error) {
	return c.itemUC.GetBrandItemByIDDirect(ctx, brandItemID)
}

func (c *BrandContract) GetBrandItemsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*dto.BrandItemBriefRes, error) {
	return c.itemUC.GetBrandItemsByFashionItemIDs(ctx, fashionItemIDs)
}

func (c *BrandContract) GetBrandItemsByIDs(ctx context.Context, brandItemIDs []uuid.UUID) ([]*dto.BrandItemRes, error) {
	return c.itemUC.GetBrandItemsByIDs(ctx, brandItemIDs)
}
