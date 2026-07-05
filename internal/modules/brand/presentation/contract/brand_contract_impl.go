package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitfeaturecode"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"

	"github.com/google/uuid"
)

type BrandContract struct {
	benefitUC uc_interfaces.IBrandBenefitUseCase
	itemUC    uc_interfaces.IBrandItemUseCase
}

func NewBrandContract(
	benefitUC uc_interfaces.IBrandBenefitUseCase,
	itemUC uc_interfaces.IBrandItemUseCase,
) IBrandContract {
	return &BrandContract{
		benefitUC: benefitUC,
		itemUC:    itemUC,
	}
}

func (c *BrandContract) CheckBrandFeatureAccess(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, featureCode benefitfeaturecode.BenefitFeatureCode) (bool, error) {
	return c.benefitUC.CheckBrandFeatureAccess(ctx, userID, brandID, featureCode)
}

func (c *BrandContract) ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, req *dto.ListEligibleBrandItemsReq) ([]*dto.BrandItemRes, []*fashion_dto.FashionItemRes, error) {
	return c.itemUC.ListEligibleBrandItemsForStyling(ctx, userID, req)
}

func (c *BrandContract) CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *dto.BrandItemRes, *fashion_dto.FashionItemRes, error) {
	return c.itemUC.CheckBrandItemEligibility(ctx, userID, fashionItemID)
}

func (c *BrandContract) SetFashionItemID(ctx context.Context, brandItemID uuid.UUID, fashionItemID uuid.UUID) error {
	return c.itemUC.SetFashionItemID(ctx, brandItemID, fashionItemID)
}

func (c *BrandContract) UpdateBrandItemStatus(ctx context.Context, brandItemID uuid.UUID, status string) error {
	return c.itemUC.UpdateBrandItemStatusDirect(ctx, brandItemID, status)
}

func (c *BrandContract) GetBrandItemByID(ctx context.Context, brandItemID uuid.UUID) (*dto.BrandItemRes, error) {
	return c.itemUC.GetBrandItemByIDDirect(ctx, brandItemID)
}
