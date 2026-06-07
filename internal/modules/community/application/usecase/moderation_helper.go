package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/contract"
	"smart-wardrobe-be/internal/shared/domain/constants/postitemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/transferstate"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

func syncWardrobeStatusByItem(
	ctx context.Context,
	postItemRepo repositories.IPostItemRepository,
	wardrobeCtr wardrobe_contract.IWardrobeContract,
	itemID uuid.UUID,
) error {
	postItems, err := postItemRepo.GetByItemID(ctx, itemID)
	if err != nil {
		return err
	}

	nextStatus := wardrobestatus.InWardrobe
	for _, item := range postItems {
		if item.TransferState == transferstate.Accepted || item.Status == postitemstatus.Sold {
			nextStatus = wardrobestatus.Sold
			break
		}
		if item.Status == postitemstatus.Available || item.TransferState == transferstate.Pending {
			nextStatus = wardrobestatus.Selling
		}
	}

	return wardrobeCtr.UpdateItemStatus(ctx, itemID, nextStatus)
}

func uniqueItemIDs(items []*entities.PostItem) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(items))
	result := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item.ItemID]; ok {
			continue
		}
		seen[item.ItemID] = struct{}{}
		result = append(result, item.ItemID)
	}
	return result
}
