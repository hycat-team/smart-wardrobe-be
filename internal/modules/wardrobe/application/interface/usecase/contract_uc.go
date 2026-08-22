package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"

	"github.com/google/uuid"
)

type IWardrobeContractUseCase interface {
	CopyItemToUser(ctx context.Context, sourceItemID uuid.UUID, targetUserID uuid.UUID) (*dto.WardrobeItemRes, error)
	UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status wardrobestatus.WardrobeItemStatus) error
	VerifyItemsForPost(ctx context.Context, userID uuid.UUID, itemIDs []uuid.UUID) error
	GetItemsByIDs(ctx context.Context, itemIDs []uuid.UUID) ([]*dto.WardrobeItemRes, error)
	GetWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID) ([]*dto.WardrobeItemRes, error)
	GetActiveWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.WardrobeItemRes, error)
	GetUserFashionItemIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	SetFashionItemID(ctx context.Context, itemID uuid.UUID, fashionItemID uuid.UUID) error
	GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*dto.WardrobeItemRes, error)
	DeleteWardrobeItem(ctx context.Context, itemID uuid.UUID) error
	GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*dto.WardrobeItemRes, error)
	MarkProcessingFailed(ctx context.Context, itemID uuid.UUID, processingVersion int, reason string, reviewReason *string) (bool, error)
	MarkProcessingNeedsReview(ctx context.Context, itemID uuid.UUID, processingVersion int, reviewReason string) (bool, error)
	CompleteProcessingSuccess(ctx context.Context, itemID uuid.UUID, processingVersion int, updates map[string]any) (bool, error)
	CountItemsByFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (int64, error)
}
