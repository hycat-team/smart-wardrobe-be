package contract

import (
	"context"

	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/shared/domain/entities"

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

func (c *BrandContract) CheckBrandFeatureAccess(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, featureCode string) (bool, error) {
	return c.benefitUC.CheckBrandFeatureAccess(ctx, userID, brandID, featureCode)
}

func (c *BrandContract) ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, filter interface{}) (interface{}, error) {
	return c.itemUC.ListEligibleBrandItemsForStyling(ctx, userID, filter)
}

func (c *BrandContract) CheckBrandItemEligibility(ctx context.Context, userID uuid.UUID, fashionItemID uuid.UUID) (bool, *entities.BrandItem, error) {
	return c.itemUC.CheckBrandItemEligibility(ctx, userID, fashionItemID)
}
