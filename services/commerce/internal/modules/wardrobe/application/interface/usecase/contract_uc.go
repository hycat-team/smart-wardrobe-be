package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

type IWardrobeContractUseCase interface {
	GetWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID) ([]*dto.WardrobeItemContractDTO, error)
	GetActiveWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.WardrobeItemContractDTO, error)
	GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*dto.WardrobeItemContractDTO, error)
	GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*dto.WardrobeItemContractDTO, error)
	CountFashionItemOwnersByFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID, excludeWardrobeItemIDs []uuid.UUID) (map[uuid.UUID]int64, error)
	GetWardrobeItemsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*dto.WardrobeItemContractDTO, error)

	GetSystemCatalogFashionItemIDs(ctx context.Context, limit int) ([]uuid.UUID, error)
	GetOwnedFashionItemIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetStaleActiveWardrobeItemsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*dto.WardrobeItemContractDTO, error)
	TouchLastRecommendedAt(ctx context.Context, wardrobeItemIDs []uuid.UUID, at time.Time) error
}
