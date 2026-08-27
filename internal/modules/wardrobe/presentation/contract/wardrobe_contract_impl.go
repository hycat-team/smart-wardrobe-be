package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"

	"github.com/google/uuid"
)

type WardrobeContract struct {
	wardrobeContractUC uc_interfaces.IWardrobeContractUseCase
}

func NewWardrobeContract(wardrobeContractUC uc_interfaces.IWardrobeContractUseCase) IWardrobeContract {
	return &WardrobeContract{wardrobeContractUC: wardrobeContractUC}
}

func (c *WardrobeContract) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status wardrobestatus.WardrobeItemStatus) error {
	return c.wardrobeContractUC.UpdateItemStatus(ctx, itemID, status)
}

func (c *WardrobeContract) GetWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID) ([]*dto.WardrobeItemRes, error) {
	return c.wardrobeContractUC.GetWardrobeItemsByUserID(ctx, userID)
}

func (c *WardrobeContract) GetActiveWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.WardrobeItemRes, error) {
	return c.wardrobeContractUC.GetActiveWardrobeItemsByUserID(ctx, userID, limit)
}

func (c *WardrobeContract) SetFashionItemID(ctx context.Context, itemID uuid.UUID, fashionItemID uuid.UUID) error {
	return c.wardrobeContractUC.SetFashionItemID(ctx, itemID, fashionItemID)
}

func (c *WardrobeContract) GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*dto.WardrobeItemRes, error) {
	return c.wardrobeContractUC.GetFailedItemsForCleanup(ctx, limit)
}

func (c *WardrobeContract) DeleteWardrobeItem(ctx context.Context, itemID uuid.UUID) error {
	return c.wardrobeContractUC.DeleteWardrobeItem(ctx, itemID)
}

func (c *WardrobeContract) GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*dto.WardrobeItemRes, error) {
	return c.wardrobeContractUC.GetStaleProcessingItems(ctx, staleBefore, limit)
}

func (c *WardrobeContract) CountItemsByFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (int64, error) {
	return c.wardrobeContractUC.CountItemsByFashionIDs(ctx, fashionItemIDs)
}
