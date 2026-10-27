package repositories

import (
	"context"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/entities"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type IWardrobeItemRepository interface {
	shared_repos.IGenericRepository[entities.WardrobeItem, uuid.UUID]
	// LockUserWardrobe takes a transaction-scoped advisory lock for one user so
	// that a wardrobe quota check and the insert it guards cannot interleave
	// with another request of the same user. It MUST be called on a context
	// that carries a unit-of-work transaction.
	LockUserWardrobe(ctx context.Context, userID uuid.UUID) error
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	CountByUserIDAndFilters(ctx context.Context, userID uuid.UUID, fashionItemIDs []uuid.UUID, statuses []wardrobestatus.WardrobeItemStatus, filterByFashionItems bool) (int64, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, fashionItemIDs []uuid.UUID) ([]*entities.WardrobeItem, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*entities.WardrobeItem, error)
	GetByUserIDAndFiltersPaginated(ctx context.Context, userID uuid.UUID, fashionItemIDs []uuid.UUID, statuses []wardrobestatus.WardrobeItemStatus, filterByFashionItems bool, pagination shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error)
	BulkCreate(ctx context.Context, items []*entities.WardrobeItem) error
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.WardrobeItem, error)
	GetByUserIDAndFashionItemIDs(ctx context.Context, userID uuid.UUID, fashionItemIDs []uuid.UUID) ([]*entities.WardrobeItem, error)
	GetByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*entities.WardrobeItem, error)
	CountItemsGroupedByFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID, excludeItemIDs []uuid.UUID) (map[uuid.UUID]int64, error)
	GetUserFashionItemCounts(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]int64, error)
	CountItems(ctx context.Context, fashionItemIDs []uuid.UUID, filterByFashionItems bool, itemType itemtype.ItemType) (int64, error)
	GetItems(ctx context.Context, fashionItemIDs []uuid.UUID, filterByFashionItems bool, itemType itemtype.ItemType) ([]*entities.WardrobeItem, error)
	GetItemsPaginated(ctx context.Context, fashionItemIDs []uuid.UUID, filterByFashionItems bool, itemType itemtype.ItemType, pagination shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error)
	GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*entities.WardrobeItem, error)
	GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*entities.WardrobeItem, error)
	ClaimManualAnalysisRetry(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, now time.Time) (*entities.WardrobeItem, bool, error)
	ClaimStaleProcessingRetry(ctx context.Context, itemID uuid.UUID, processingVersion int, staleBefore time.Time, now time.Time) (*entities.WardrobeItem, bool, error)
	MarkProcessingFailed(ctx context.Context, itemID uuid.UUID, processingVersion int, reason string, reviewReason *string) (bool, error)
	MarkProcessingNeedsReview(ctx context.Context, itemID uuid.UUID, processingVersion int, reviewReason string) (bool, error)
	CompleteProcessingSuccess(ctx context.Context, itemID uuid.UUID, processingVersion int, updates map[string]any) (bool, error)
	TouchLastUsedAt(ctx context.Context, ids []uuid.UUID, usedAt time.Time) error
	// TouchLastRecommendedAt marks wardrobe items as recently recommended
	// (feature 033). Best-effort: a failed write is tolerated and retried by the
	// next recommendation.
	TouchLastRecommendedAt(ctx context.Context, ids []uuid.UUID, recommendedAt time.Time) error
	// GetStaleActiveItems returns the user's own active items ordered by the two
	// recency signals so the oldest ones surface first.
	GetStaleActiveItems(ctx context.Context, userID uuid.UUID, limit int) ([]*entities.WardrobeItem, error)
	// GetActiveSystemCatalogItemsByIDs returns only valid catalog rows: item_type
	// SystemCatalogItem, status InWardrobe, not deleted.
	GetActiveSystemCatalogItemsByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.WardrobeItem, error)
	// GetSystemCatalogFashionItemIDs lists fashion item ids of the system
	// catalog rows available for outfit completion.
	GetSystemCatalogFashionItemIDs(ctx context.Context, limit int) ([]uuid.UUID, error)
	// GetOwnedFashionItemIDs lists every non-deleted fashion item id the user
	// owns, regardless of item status (ownership for dedupe/clone).
	GetOwnedFashionItemIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetUserWardrobeSummaryStats(ctx context.Context, userID uuid.UUID) (*UserWardrobeSummaryStatsDTO, error)
	GetUnderutilizedItems(ctx context.Context, userID uuid.UUID, limit int) ([]*entities.WardrobeItem, error)
}

type IOutfitRepository interface {
	shared_repos.IGenericRepository[entities.Outfit, uuid.UUID]
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Outfit, error)
	GetByUserIDPaginated(ctx context.Context, userID uuid.UUID, pagination shared_dto.PaginationQuery) ([]*entities.Outfit, error)
	GetDetailByID(ctx context.Context, id uuid.UUID) (*entities.Outfit, []*entities.OutfitItem, error)
	CreateWithItems(ctx context.Context, outfit *entities.Outfit, items []*entities.OutfitItem) error
	UpdateWithItems(ctx context.Context, outfit *entities.Outfit, items []*entities.OutfitItem) error
	DeleteOutfit(ctx context.Context, id uuid.UUID) error
	GetFashionItemIDsByOutfitIDs(ctx context.Context, outfitIDs []uuid.UUID) ([]uuid.UUID, error)
	CountOutfitItemsByFashionIDs(ctx context.Context, fashionItemIDs []uuid.UUID) (map[uuid.UUID]int64, error)
}
