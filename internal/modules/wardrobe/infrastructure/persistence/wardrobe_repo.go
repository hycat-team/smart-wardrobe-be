package persistence

import (
	"context"

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

func (r *WardrobeItemRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entities.WardrobeItem{}).Where("user_id = ?", userID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *WardrobeItemRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.WardrobeItem, error) {
	var items []*entities.WardrobeItem
	err := r.GetDB(ctx).Preload("Category").Where("user_id = ?", userID).Order("created_at DESC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WardrobeItemRepository) BulkCreate(ctx context.Context, items []*entities.WardrobeItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.GetDB(ctx).Create(&items).Error
}

func (r *WardrobeItemRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.WardrobeItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var items []*entities.WardrobeItem
	err := r.GetDB(ctx).Preload("Category").Where("id IN ?", ids).Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}
