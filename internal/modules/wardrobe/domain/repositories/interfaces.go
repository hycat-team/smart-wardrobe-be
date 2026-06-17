package repositories

import (
	"context"
	"time"

	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

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
	Item            *entities.WardrobeItem
	VectorScore     float64
	LexicalScore    float64
	RetrievalScore  float64
	RetrievalRank   int
	RetrievalSource string
}

type IWardrobeItemRepository interface {
	shared_repos.IGenericRepository[entities.WardrobeItem, uuid.UUID]
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	CountByUserIDAndFilters(ctx context.Context, userID uuid.UUID, categorySlug *string, statuses []wardrobestatus.WardrobeItemStatus) (int64, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, categorySlug *string) ([]*entities.WardrobeItem, error)
	GetByUserIDAndFiltersPaginated(ctx context.Context, userID uuid.UUID, categorySlug *string, statuses []wardrobestatus.WardrobeItemStatus, pagination shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error)
	BulkCreate(ctx context.Context, items []*entities.WardrobeItem) error
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.WardrobeItem, error)
	CountItems(ctx context.Context, query *string, categorySlug *string, itemType itemtype.ItemType) (int64, error)
	GetItems(ctx context.Context, query *string, categorySlug *string, itemType itemtype.ItemType) ([]*entities.WardrobeItem, error)
	GetItemsPaginated(ctx context.Context, query *string, categorySlug *string, itemType itemtype.ItemType, pagination shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error)
	GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*entities.WardrobeItem, error)
	GetStaleProcessingItems(ctx context.Context, staleBefore time.Time, limit int) ([]*entities.WardrobeItem, error)
	ClaimManualAnalysisRetry(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, now time.Time) (*entities.WardrobeItem, bool, error)
	ClaimStaleProcessingRetry(ctx context.Context, itemID uuid.UUID, processingVersion int, staleBefore time.Time, now time.Time) (*entities.WardrobeItem, bool, error)
	MarkProcessingFailed(ctx context.Context, itemID uuid.UUID, processingVersion int, reason string, reviewReason *string) (bool, error)
	MarkProcessingNeedsReview(ctx context.Context, itemID uuid.UUID, processingVersion int, reviewReason string) (bool, error)
	CompleteProcessingSuccess(ctx context.Context, itemID uuid.UUID, processingVersion int, updates map[string]any) (bool, error)
	TouchLastUsedAt(ctx context.Context, ids []uuid.UUID, usedAt time.Time) error
	// GetHybridCandidates truy vấn danh sách ứng viên gợi ý trang phục kết hợp (hybrid search) bằng cách thực hiện
	// tìm kiếm ngữ nghĩa qua Vector Embedding (Cosine Similarity) và tìm kiếm từ khóa (Full-text Search tsquery).
	// Danh sách kết quả được tổng hợp điểm và xếp hạng lại sử dụng thuật toán Reciprocal Rank Fusion (RRF).
	//
	// Hành vi:
	// 	- 1. Nếu [semanticVector] không rỗng, thực hiện tìm kiếm Vector.
	// 	- 2. Nếu [lexicalTerms] không rỗng, thực hiện tìm kiếm Lexical thô và mở rộng (FTS).
	// 	- 3. Loại trừ các món đồ khớp với [excludedTerms] hoặc không thỏa mãn [hardFilters] (như mùa vụ, danh mục).
	// 	- 4. Kết hợp hai nguồn kết quả và tính điểm RRF dựa trên tham số [rrfK].
	// 	- 5. Giới hạn số lượng kết quả tối đa bằng [limit].
	//
	// Đầu vào mẫu:
	//   userID: "8e05c317-062e-4b47-ba21-12f5a04f21db"
	//   semanticVector: entities.Vector{0.1, -0.2, ...}
	//   lexicalTerms: []string{"ao", "khoac"}
	//   excludedTerms: []string{"len"}
	//   hardFilters: RecommendationHardFilters{Seasonality: []string{"winter"}}
	//   limit: 20
	//   rrfK: 60
	//
	// Đầu ra mẫu:
	//   ([]HybridCandidate, nil)
	GetHybridCandidates(ctx context.Context, userID uuid.UUID, semanticVector entities.Vector, lexicalTerms []string, excludedTerms []string, hardFilters RecommendationHardFilters, limit int, rrfK int) ([]HybridCandidate, error)
}

type ICategoryRepository interface {
	shared_repos.IGenericRepository[entities.Category, uuid.UUID]
	GetBySlug(ctx context.Context, slug string) (*entities.Category, error)
	GetByName(ctx context.Context, name string) (*entities.Category, error)
	CountWardrobeItemsByCategoryAndItemType(ctx context.Context, categoryID uuid.UUID, itemType itemtype.ItemType) (int64, error)
	ReassignSystemCatalogItemsToCategory(ctx context.Context, fromCategoryID uuid.UUID, toCategoryID uuid.UUID) error
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
}
