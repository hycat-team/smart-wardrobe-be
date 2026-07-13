package persistence

import (
	"context"
	"smart-wardrobe-be/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	shared_persist.GenericRepository[entities.Category, uuid.UUID]
}

func NewCategoryRepository(db *gorm.DB) repositories.ICategoryRepository {
	relations := []string{}
	return &CategoryRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.Category, uuid.UUID](
			db, relations,
		),
	}
}

func (r *CategoryRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.Category, error) {
	var items []*entities.Category
	err := r.GetDB(ctx).Where("id IN ?", ids).Find(&items).Error
	return items, err
}

func (r *CategoryRepository) GetBySlug(ctx context.Context, slug string) (*entities.Category, error) {
	var category entities.Category
	err := r.GetDB(ctx).Where("slug = ?", slug).First(&category).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

func (r *CategoryRepository) GetByName(ctx context.Context, name string) (*entities.Category, error) {
	var category entities.Category
	err := r.GetDB(ctx).Where("name = ?", name).First(&category).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

func (r *CategoryRepository) GetAll(ctx context.Context) ([]*entities.Category, error) {
	var categories []*entities.Category
	err := r.GetDB(ctx).
		Order("sort_order ASC").
		Order("name ASC").
		Find(&categories).Error
	if err != nil {
		return nil, err
	}
	return categories, nil
}
