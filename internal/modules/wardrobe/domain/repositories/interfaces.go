package repositories

import (
	"context"
	"time"

	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IWardrobeItemRepository interface {
	shared_repos.IGenericRepository[entities.WardrobeItem, uuid.UUID]
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	CountByUserIDAndCategory(ctx context.Context, userID uuid.UUID, categorySlug *string) (int64, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, categorySlug *string) ([]*entities.WardrobeItem, error)
	GetByUserIDPaginated(ctx context.Context, userID uuid.UUID, categorySlug *string, pagination shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error)
	BulkCreate(ctx context.Context, items []*entities.WardrobeItem) error
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.WardrobeItem, error)
	GetItems(ctx context.Context, query *string, categorySlug *string, itemType itemtype.ItemType) ([]*entities.WardrobeItem, error)
	GetItemsPaginated(ctx context.Context, query *string, categorySlug *string, itemType itemtype.ItemType, pagination shared_dto.PaginationQuery) ([]*entities.WardrobeItem, error)
	CountItems(ctx context.Context, query *string, categorySlug *string, itemType itemtype.ItemType) (int64, error)
	GetFailedItemsForCleanup(ctx context.Context, limit int) ([]*entities.WardrobeItem, error)
	TouchLastUsedAt(ctx context.Context, ids []uuid.UUID, usedAt time.Time) error
	GetSimilarItemsByVectorAndCategory(ctx context.Context, userID uuid.UUID, categoryID uuid.UUID, vector entities.Vector, limit int) ([]*entities.WardrobeItem, error)
	GetRecentlyActiveItemsByCategory(ctx context.Context, userID uuid.UUID, categoryID uuid.UUID, limit int) ([]*entities.WardrobeItem, error)
	GetHybridCandidates(ctx context.Context, userID uuid.UUID, semanticVector entities.Vector, keywords []string, limit int) ([]*entities.WardrobeItem, error)
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
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Outfit, error)
	GetByUserIDPaginated(ctx context.Context, userID uuid.UUID, pagination shared_dto.PaginationQuery) ([]*entities.Outfit, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
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
	GetByContextID(ctx context.Context, contextID uuid.UUID) ([]*entities.Message, error)
	GetByContextIDPaginated(ctx context.Context, contextID uuid.UUID, pagination shared_dto.PaginationQuery) ([]*entities.Message, error)
	GetRecentByContextID(ctx context.Context, contextID uuid.UUID, limit int) ([]*entities.Message, error)
	GetOldestByContextID(ctx context.Context, contextID uuid.UUID, limit int) ([]*entities.Message, error)
	DeleteByIDs(ctx context.Context, ids []uuid.UUID) error
	CountByContextID(ctx context.Context, contextID uuid.UUID) (int64, error)
}
