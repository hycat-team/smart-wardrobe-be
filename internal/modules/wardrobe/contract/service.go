package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobe/wardrobestatus"

	"github.com/google/uuid"
)

type IWardrobeContract interface {
	CopyItemToUser(ctx context.Context, sourceItemID uuid.UUID, targetUserID uuid.UUID) (*dto.WardrobeItemRes, error)
	UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status wardrobestatus.WardrobeItemStatus) error
	VerifyItemsForPost(ctx context.Context, userID uuid.UUID, itemIDs []uuid.UUID) error
	GetItemsByIDs(ctx context.Context, itemIDs []uuid.UUID) ([]*dto.WardrobeItemRes, error)
}
