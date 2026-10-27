package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/interface/usecase"

	"github.com/google/uuid"
)

type WardrobeContract struct {
	wardrobeContractUC uc_interfaces.IWardrobeContractUseCase
}

func NewWardrobeContract(wardrobeContractUC uc_interfaces.IWardrobeContractUseCase) IWardrobeContract {
	return &WardrobeContract{wardrobeContractUC: wardrobeContractUC}
}

func (c *WardrobeContract) GetWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID) ([]*dto.WardrobeItemContractDTO, error) {
	return c.wardrobeContractUC.GetWardrobeItemsByUserID(ctx, userID)
}

func (c *WardrobeContract) GetActiveWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.WardrobeItemContractDTO, error) {
	return c.wardrobeContractUC.GetActiveWardrobeItemsByUserID(ctx, userID, limit)
}

func (c *WardrobeContract) GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*dto.WardrobeItemContractDTO, error) {
	return c.wardrobeContractUC.GetFailedItemsForCleanup(ctx, limit)
}

func (c *WardrobeContract) GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*dto.WardrobeItemContractDTO, error) {
	return c.wardrobeContractUC.GetStaleProcessingItems(ctx, staleBefore, limit)
}

func (c *WardrobeContract) CountFashionItemOwnersByFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID, excludeWardrobeItemIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	return c.wardrobeContractUC.CountFashionItemOwnersByFashionIDs(ctx, fashionItemIDs, excludeWardrobeItemIDs)
}

func (c *WardrobeContract) GetWardrobeItemsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*dto.WardrobeItemContractDTO, error) {
	return c.wardrobeContractUC.GetWardrobeItemsByFashionItemIDs(ctx, fashionItemIDs)
}

func (c *WardrobeContract) GetSystemCatalogFashionItemIDs(ctx context.Context, limit int) ([]uuid.UUID, error) {
	return c.wardrobeContractUC.GetSystemCatalogFashionItemIDs(ctx, limit)
}

func (c *WardrobeContract) GetOwnedFashionItemIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return c.wardrobeContractUC.GetOwnedFashionItemIDs(ctx, userID)
}

func (c *WardrobeContract) GetStaleActiveWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.WardrobeItemContractDTO, error) {
	return c.wardrobeContractUC.GetStaleActiveWardrobeItemsByUserID(ctx, userID, limit)
}

func (c *WardrobeContract) TouchLastRecommendedAt(ctx context.Context, wardrobeItemIDs []uuid.UUID, at time.Time) error {
	return c.wardrobeContractUC.TouchLastRecommendedAt(ctx, wardrobeItemIDs, at)
}
