package repositories

import (
	"context"

	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IWardrobeItemRepository interface {
	shared_repos.IGenericRepository[entities.WardrobeItem, uuid.UUID]
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.WardrobeItem, error)
	BulkCreate(ctx context.Context, items []*entities.WardrobeItem) error
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.WardrobeItem, error)
	GetItems(ctx context.Context, query *string, itemType itemtype.ItemType) ([]*entities.WardrobeItem, error)
}

type ICategoryRepository interface {
	shared_repos.IGenericRepository[entities.Category, uuid.UUID]
}

type IOutfitRepository interface {
	shared_repos.IGenericRepository[entities.Outfit, uuid.UUID]
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Outfit, error)
	GetDetailByID(ctx context.Context, id uuid.UUID) (*entities.Outfit, []*entities.OutfitItem, error)
	CreateWithItems(ctx context.Context, outfit *entities.Outfit, items []*entities.OutfitItem) error
	UpdateWithItems(ctx context.Context, outfit *entities.Outfit, items []*entities.OutfitItem) error
	DeleteOutfit(ctx context.Context, id uuid.UUID) error
}
