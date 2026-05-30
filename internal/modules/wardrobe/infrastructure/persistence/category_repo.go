package persistence

import (
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	shared_persist.GenericRepository[entities.Category, uuid.UUID]
}

func NewCategoryRepository(db *gorm.DB) repositories.ICategoryRepository {
	return &CategoryRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.Category, uuid.UUID](db),
	}
}
