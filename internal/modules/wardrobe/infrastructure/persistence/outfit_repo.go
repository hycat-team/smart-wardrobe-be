package persistence

import (
	"context"
	"errors"
	"sort"

	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OutfitRepository struct {
	shared_persist.GenericRepository[entities.Outfit, uuid.UUID]
}

func NewOutfitRepository(db *gorm.DB) repositories.IOutfitRepository {
	relations := []string{"Items.WardrobeItem.Category"}
	return &OutfitRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.Outfit, uuid.UUID](db, relations),
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

func (r *OutfitRepository) GetByUserIDPaginated(ctx context.Context, userID uuid.UUID, pagination shared_dto.PaginationQuery) ([]*entities.Outfit, error) {
	var outfits []*entities.Outfit
	query := r.GetDB(ctx).Model(&entities.Outfit{}).Where("user_id = ?", userID)
	db := shared_persist.ApplyPagination(query, pagination)
	err := db.Order("created_at DESC").Find(&outfits).Error
	if err != nil {
		return nil, err
	}
	return outfits, nil
}

func (r *OutfitRepository) GetDetailByID(ctx context.Context, id uuid.UUID) (*entities.Outfit, []*entities.OutfitItem, error) {
	var outfit entities.Outfit
	err := r.GetQueryWithPreload(ctx).Where("id = ?", id).First(&outfit).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	sort.Slice(outfit.Items, func(i, j int) bool {
		return outfit.Items[i].LayerOrder < outfit.Items[j].LayerOrder
	})

	return &outfit, outfit.Items, nil
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

		// Delete related old items
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
		// Delete intermediate items first
		if err := tx.Where("outfit_id = ?", id).Delete(&entities.OutfitItem{}).Error; err != nil {
			return err
		}

		// Delete Outfit entity (Soft delete or Hard delete based on configuration)
		if err := tx.Where("id = ?", id).Delete(&entities.Outfit{}).Error; err != nil {
			return err
		}

		return nil
	})
}
