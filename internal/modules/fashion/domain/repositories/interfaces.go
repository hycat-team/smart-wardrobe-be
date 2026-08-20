package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/domain/entities"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

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
	GetCategoryIDsByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]dto.DashboardFashionItemCategoryDTO, error)
	UpdateCategory(ctx context.Context, fashionItemID uuid.UUID, categoryID uuid.UUID) error
	ClaimManualAnalysisRetry(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, now time.Time) (*entities.FashionItem, bool, error)
	ClaimStaleProcessingRetry(ctx context.Context, itemID uuid.UUID, processingVersion int, staleBefore time.Time, now time.Time) (*entities.FashionItem, bool, error)
	IncrementProcessingRetryCount(ctx context.Context, itemID uuid.UUID) error
	MarkProcessingFailed(ctx context.Context, itemID uuid.UUID, processingVersion int, reason string, reviewReason *string) (bool, error)
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
