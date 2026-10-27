package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type ICategoryRepository interface {
	shared_repos.IGenericRepository[entities.Category, uuid.UUID]
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.Category, error)
	GetBySlug(ctx context.Context, slug string) (*entities.Category, error)
	GetByName(ctx context.Context, name string) (*entities.Category, error)
}

type RecommendationHardFilters struct {
	Seasonality   []string
	CategorySlugs []string
}

const (
	HybridCandidateSourceHybrid   = "hybrid"
	HybridCandidateSourceVector   = "vector"
	HybridCandidateSourceLexical  = "lexical"
	HybridCandidateSourceFallback = "fallback"
)

type HybridCandidate struct {
	ID              uuid.UUID
	FashionItemID   *uuid.UUID
	VectorScore     float64
	LexicalScore    float64
	RetrievalScore  float64
	RetrievalRank   int
	RetrievalSource string
}

type IFashionItemRepository interface {
	shared_repos.IGenericRepository[entities.FashionItem, uuid.UUID]
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.FashionItem, error)
	GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*entities.FashionItem, error)
	GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*entities.FashionItem, error)
	SearchFashionItemIDs(ctx context.Context, query *string, categorySlug *string) ([]uuid.UUID, error)
	GetFashionItemIDsByCategory(ctx context.Context, categoryID uuid.UUID) ([]uuid.UUID, error)
	GetCategoryIDsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]DashboardFashionItemCategoryDTO, error)
	UpdateCategory(ctx context.Context, fashionItemID uuid.UUID, categoryID uuid.UUID) error
	ClaimManualAnalysisRetry(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, now time.Time) (*entities.FashionItem, bool, error)
	ClaimStaleProcessingRetry(ctx context.Context, itemID uuid.UUID, processingVersion int, staleBefore time.Time, now time.Time) (*entities.FashionItem, bool, error)
	IncrementProcessingRetryCount(ctx context.Context, itemID uuid.UUID) error
	MarkProcessingStarted(ctx context.Context, itemID uuid.UUID, processingVersion int, now time.Time) error
	MarkProcessingFailed(ctx context.Context, itemID uuid.UUID, processingVersion int, reason string, reviewReason *string) (bool, error)
	SetProcessingErrorReason(ctx context.Context, itemID uuid.UUID, reason string) error
	MarkProcessingNeedsReview(ctx context.Context, itemID uuid.UUID, processingVersion int, reviewReason string) (bool, error)
	CompleteProcessingSuccess(ctx context.Context, itemID uuid.UUID, processingVersion int, updates map[string]any) (bool, error)
	ReassignCategory(ctx context.Context, fashionItemIDs []uuid.UUID, fromCategoryID uuid.UUID, toCategoryID uuid.UUID) error
	ResetProcessingState(ctx context.Context, fashionItemID uuid.UUID) error
	GetHybridCandidates(ctx context.Context, fashionItemIDs []uuid.UUID, semanticVector []float32, lexicalTerms []string, excludedTerms []string, hardFilters RecommendationHardFilters, limit int, rrfK int) ([]HybridCandidate, error)
}

type IConversationalContextRepository interface {
	shared_repos.IGenericRepository[entities.ConversationalContext, uuid.UUID]
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.ConversationalContext, error)
}

// IUserStyleProfileRepository owns fashion.user_style_profiles (one row per
// user). It deliberately does not embed the generic repository: the primary key
// is user_id, not id, so GetByID would query a column that does not exist.
type IUserStyleProfileRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserStyleProfile, error)
	// Upsert inserts the profile or replaces the mutable columns of an existing
	// row (user_id and created_at are preserved).
	Upsert(ctx context.Context, profile *entities.UserStyleProfile) error
	Delete(ctx context.Context, userID uuid.UUID) error
	// UpdateLearnedWithLock locks the profile row with SELECT ... FOR UPDATE,
	// creates a default row when the user has none yet, applies mutate under the
	// lock and persists only the learned_* columns. Explicit preferences
	// (styles/excluded_styles/enabled) can never be overwritten by this path.
	UpdateLearnedWithLock(
		ctx context.Context,
		userID uuid.UUID,
		mutate func(profile *entities.UserStyleProfile) error,
	) (*entities.UserStyleProfile, error)
}

// IStyleSignalRepository owns fashion.style_signals.
type IStyleSignalRepository interface {
	// InsertIfAbsent inserts the signal with ON CONFLICT (id) DO NOTHING and
	// reports whether a new row was written. It is the idempotency boundary for
	// at-least-once outbox delivery.
	InsertIfAbsent(ctx context.Context, signal *entities.StyleSignal) (bool, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
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
