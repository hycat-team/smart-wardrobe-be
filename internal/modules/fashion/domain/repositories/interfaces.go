package repositories

import (
	"context"
	"time"

	brand_repos "smart-wardrobe-be/internal/modules/brand/domain/repositories"
	wardrobe_repos "smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IWardrobeItemRepository = wardrobe_repos.IWardrobeItemRepository
type ICategoryRepository = wardrobe_repos.ICategoryRepository
type IOutfitRepository = wardrobe_repos.IOutfitRepository
type IBrandItemRepository = brand_repos.IBrandItemRepository
type RecommendationHardFilters = wardrobe_repos.RecommendationHardFilters
type HybridCandidate = wardrobe_repos.HybridCandidate

const (
	HybridCandidateSourceHybrid   = wardrobe_repos.HybridCandidateSourceHybrid
	HybridCandidateSourceVector   = wardrobe_repos.HybridCandidateSourceVector
	HybridCandidateSourceLexical  = wardrobe_repos.HybridCandidateSourceLexical
	HybridCandidateSourceFallback = wardrobe_repos.HybridCandidateSourceFallback
)

type IFashionItemRepository interface {
	shared_repos.IGenericRepository[entities.FashionItem, uuid.UUID]
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.FashionItem, error)
	GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*entities.FashionItem, error)
	GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*entities.FashionItem, error)
	ClaimManualAnalysisRetry(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, now time.Time) (*entities.FashionItem, bool, error)
	ClaimStaleProcessingRetry(ctx context.Context, itemID uuid.UUID, processingVersion int, staleBefore time.Time, now time.Time) (*entities.FashionItem, bool, error)
	MarkProcessingFailed(ctx context.Context, itemID uuid.UUID, processingVersion int, reason string, reviewReason *string) (bool, error)
	MarkProcessingNeedsReview(ctx context.Context, itemID uuid.UUID, processingVersion int, reviewReason string) (bool, error)
	CompleteProcessingSuccess(ctx context.Context, itemID uuid.UUID, processingVersion int, updates map[string]any) (bool, error)
}

type IConversationalContextRepository interface {
	shared_repos.IGenericRepository[entities.ConversationalContext, uuid.UUID]
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.ConversationalContext, error)
}

type IMessageRepository interface {
	shared_repos.IGenericRepository[entities.Message, uuid.UUID]
	CountByContextID(ctx context.Context, contextID uuid.UUID) (int64, error)
	GetByContextID(ctx context.Context, contextID uuid.UUID) ([]*entities.Message, error)
	GetByContextIDPaginated(ctx context.Context, contextID uuid.UUID, pagination shared_dto.PaginationQuery) ([]*entities.Message, error)
	GetRecentByContextID(ctx context.Context, contextID uuid.UUID, limit int) ([]*entities.Message, error)
	GetOldestByContextID(ctx context.Context, contextID uuid.UUID, limit int) ([]*entities.Message, error)
	DeleteByIDs(ctx context.Context, ids []uuid.UUID) error
	CountUnsummarizedByContextID(ctx context.Context, contextID uuid.UUID) (int64, error)
	GetOldestUnsummarizedByContextID(ctx context.Context, contextID uuid.UUID, limit int) ([]*entities.Message, error)
	MarkAsSummarized(ctx context.Context, ids []uuid.UUID) error
}
