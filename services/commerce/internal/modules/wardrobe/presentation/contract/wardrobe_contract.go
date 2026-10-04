package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

type IWardrobeContract interface {
	GetWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID) ([]*dto.WardrobeItemContractDTO, error)
	GetActiveWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.WardrobeItemContractDTO, error)
	GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*dto.WardrobeItemContractDTO, error)
	GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*dto.WardrobeItemContractDTO, error)
	CountFashionItemOwnersByFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID, excludeWardrobeItemIDs []uuid.UUID) (map[uuid.UUID]int64, error)
	GetWardrobeItemsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*dto.WardrobeItemContractDTO, error)
}
