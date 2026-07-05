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

func (c *WardrobeContract) CopyItemToUser(ctx context.Context, sourceItemID uuid.UUID, targetUserID uuid.UUID) (*dto.WardrobeItemRes, error) {
	return c.wardrobeContractUC.CopyItemToUser(ctx, sourceItemID, targetUserID)
}

func (c *WardrobeContract) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status wardrobestatus.WardrobeItemStatus) error {
	return c.wardrobeContractUC.UpdateItemStatus(ctx, itemID, status)
}

func (c *WardrobeContract) VerifyItemsForPost(ctx context.Context, userID uuid.UUID, itemIDs []uuid.UUID) error {
	return c.wardrobeContractUC.VerifyItemsForPost(ctx, userID, itemIDs)
}

func (c *WardrobeContract) GetItemsByIDs(ctx context.Context, itemIDs []uuid.UUID) ([]*dto.WardrobeItemRes, error) {
	return c.wardrobeContractUC.GetItemsByIDs(ctx, itemIDs)
}

func (c *WardrobeContract) GetWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID) ([]*dto.WardrobeItemRes, error) {
	return c.wardrobeContractUC.GetWardrobeItemsByUserID(ctx, userID)
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

func (c *WardrobeContract) MarkProcessingFailed(ctx context.Context, itemID uuid.UUID, processingVersion int, reason string, reviewReason *string) (bool, error) {
	return c.wardrobeContractUC.MarkProcessingFailed(ctx, itemID, processingVersion, reason, reviewReason)
}

func (c *WardrobeContract) MarkProcessingNeedsReview(ctx context.Context, itemID uuid.UUID, processingVersion int, reviewReason string) (bool, error) {
	return c.wardrobeContractUC.MarkProcessingNeedsReview(ctx, itemID, processingVersion, reviewReason)
}

func (c *WardrobeContract) CompleteProcessingSuccess(ctx context.Context, itemID uuid.UUID, processingVersion int, updates map[string]any) (bool, error) {
	return c.wardrobeContractUC.CompleteProcessingSuccess(ctx, itemID, processingVersion, updates)
}
