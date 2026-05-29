package repositories

import (
	"context"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type IWardrobeItemRepository interface {
	Create(ctx context.Context, item *entities.WardrobeItem) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.WardrobeItem, error)
}

type ICategoryRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Category, error)
}
