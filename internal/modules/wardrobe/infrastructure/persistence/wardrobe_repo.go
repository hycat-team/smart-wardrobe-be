package persistence

import (
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WardrobeItemRepository struct {
	shared_persist.GenericRepository[entities.WardrobeItem, uuid.UUID]
}

func NewWardrobeItemRepository(db *gorm.DB) repositories.IWardrobeItemRepository {
	return &WardrobeItemRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.WardrobeItem, uuid.UUID](db),
	}
}
