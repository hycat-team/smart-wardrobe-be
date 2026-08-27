package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"

	"github.com/google/uuid"
)

type IWardrobeContract interface {
	UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status wardrobestatus.WardrobeItemStatus) error
	GetWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID) ([]*dto.WardrobeItemRes, error)
	GetActiveWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.WardrobeItemRes, error)
	SetFashionItemID(ctx context.Context, itemID uuid.UUID, fashionItemID uuid.UUID) error
	GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*dto.WardrobeItemRes, error)
	DeleteWardrobeItem(ctx context.Context, itemID uuid.UUID) error
	GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*dto.WardrobeItemRes, error)
	CountItemsByFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (int64, error)
}
