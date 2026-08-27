package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

type IWardrobeContract interface {
	GetWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID) ([]*dto.WardrobeItemRes, error)
	GetActiveWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.WardrobeItemRes, error)
	GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*dto.WardrobeItemRes, error)
	GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*dto.WardrobeItemRes, error)
	CountItemsByFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (int64, error)
}
