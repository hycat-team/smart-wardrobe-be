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

func (c *WardrobeContract) CountItemsByFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (int64, error) {
	return c.wardrobeContractUC.CountItemsByFashionIDs(ctx, fashionItemIDs)
}
