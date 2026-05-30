package persistence

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OutfitRepository struct {
	shared_persist.GenericRepository[entities.Outfit, uuid.UUID]
}

func NewOutfitRepository(db *gorm.DB) repositories.IOutfitRepository {
	return &OutfitRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.Outfit, uuid.UUID](db),
	}
}

func (r *OutfitRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Outfit, error) {
	var outfits []*entities.Outfit
	err := r.GetDB(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&outfits).Error
	if err != nil {
		return nil, err
	}
	return outfits, nil
}

func (r *OutfitRepository) GetDetailByID(ctx context.Context, id uuid.UUID) (*entities.Outfit, []*entities.OutfitItem, error) {
	var outfit entities.Outfit
	err := r.GetDB(ctx).Where("id = ?", id).First(&outfit).Error
	if err != nil {
		return nil, nil, err
	}

	var items []*entities.OutfitItem
	err = r.GetDB(ctx).Preload("Wardrobe.Category").Where("outfit_id = ?", id).Order("layer_order ASC").Find(&items).Error
	if err != nil {
		return nil, nil, err
	}

	return &outfit, items, nil
}

func (r *OutfitRepository) CreateWithItems(ctx context.Context, outfit *entities.Outfit, items []*entities.OutfitItem) error {
	return r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(outfit).Error; err != nil {
			return err
		}

		for _, item := range items {
			item.OutfitID = outfit.ID
		}

		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *OutfitRepository) UpdateWithItems(ctx context.Context, outfit *entities.Outfit, items []*entities.OutfitItem) error {
	return r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(outfit).Error; err != nil {
			return err
		}

		// Xóa các items cũ liên quan
		if err := tx.Where("outfit_id = ?", outfit.ID).Delete(&entities.OutfitItem{}).Error; err != nil {
			return err
		}

		for _, item := range items {
			item.OutfitID = outfit.ID
		}

		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *OutfitRepository) DeleteOutfit(ctx context.Context, id uuid.UUID) error {
	return r.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		// Xóa các items trung gian trước
		if err := tx.Where("outfit_id = ?", id).Delete(&entities.OutfitItem{}).Error; err != nil {
			return err
		}

		// Xóa thực thể Outfit (Soft delete hoặc Hard delete dựa trên cấu hình)
		if err := tx.Where("id = ?", id).Delete(&entities.Outfit{}).Error; err != nil {
			return err
		}

		return nil
	})
}
