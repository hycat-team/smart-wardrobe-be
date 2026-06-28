package persistence

import (
	"context"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/entities"
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

func (r *CategoryRepository) CountWardrobeItemsByCategoryAndItemType(ctx context.Context, categoryID uuid.UUID, kind itemtype.ItemType) (int64, error) {
	var count int64
	err := r.GetDB(ctx).
		Model(&entities.WardrobeItem{}).
		Joins("JOIN fashion_items ON fashion_items.id = wardrobe_items.fashion_item_id").
		Where("fashion_items.category_id = ? AND wardrobe_items.item_type = ? AND wardrobe_items.is_deleted = ?", categoryID, kind, false).
		Count(&count).Error
	return count, err
}

func (r *CategoryRepository) ReassignSystemCatalogItemsToCategory(ctx context.Context, fromCategoryID uuid.UUID, toCategoryID uuid.UUID) error {
	systemFashionItems := r.GetDB(ctx).
		Model(&entities.WardrobeItem{}).
		Select("wardrobe_items.fashion_item_id").
		Joins("JOIN fashion_items ON fashion_items.id = wardrobe_items.fashion_item_id").
		Where("fashion_items.category_id = ? AND wardrobe_items.item_type = ? AND wardrobe_items.is_deleted = ?", fromCategoryID, itemtype.SystemCatalogItem, false)

	return r.GetDB(ctx).
		Model(&entities.FashionItem{}).
		Where("id IN (?)", systemFashionItems).
		Update("category_id", toCategoryID).Error
}
