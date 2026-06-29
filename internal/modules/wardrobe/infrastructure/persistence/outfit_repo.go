package persistence

import (
	"context"
	"errors"
	"sort"

	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/branditem/branditemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobe/outfititemcontext"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobe/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OutfitRepository struct {
	shared_persist.GenericRepository[entities.Outfit, uuid.UUID]
}

func NewOutfitRepository(db *gorm.DB) repositories.IOutfitRepository {
	relations := []string{"Items.FashionItem", "Items.FashionItem.Category"}
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

func (r *OutfitRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entities.Outfit{}).Where("user_id = ?", userID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
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

	if err := r.attachUserWardrobeItems(ctx, &outfit); err != nil {
		return nil, nil, err
	}
	if err := r.attachBrandItems(ctx, &outfit); err != nil {
		return nil, nil, err
	}

	var validItems []*entities.OutfitItem
	for _, item := range outfit.Items {
		if item.ItemContext == outfititemcontext.UserWardrobe {
			if item.WardrobeItem != nil &&
				!item.WardrobeItem.IsDeleted &&
				item.WardrobeItem.Status == wardrobestatus.InWardrobe &&
				item.WardrobeItem.UserID == outfit.UserID {
				validItems = append(validItems, item)
			}
		} else if item.ItemContext == outfititemcontext.BrandItem {
			if item.BrandItem != nil &&
				item.BrandItem.Status == branditemstatus.Active {
				validItems = append(validItems, item)
			}
		}
	}
	outfit.Items = validItems

	sort.Slice(outfit.Items, func(i, j int) bool {
		return outfit.Items[i].LayerOrder < outfit.Items[j].LayerOrder
	})

	return &outfit, outfit.Items, nil
}

func (r *OutfitRepository) attachUserWardrobeItems(ctx context.Context, outfit *entities.Outfit) error {
	if outfit == nil || len(outfit.Items) == 0 {
		return nil
	}

	fashionIDs := make([]uuid.UUID, 0, len(outfit.Items))
	for _, item := range outfit.Items {
		if item.ItemContext == outfititemcontext.UserWardrobe {
			fashionIDs = append(fashionIDs, item.FashionItemID)
		}
	}
	if len(fashionIDs) == 0 {
		return nil
	}

	var wardrobeItems []*entities.WardrobeItem
	if err := r.GetDB(ctx).
		Preload("FashionItem").
		Preload("FashionItem.Category").
		Where("user_id = ? AND fashion_item_id IN ?", outfit.UserID, fashionIDs).
		Find(&wardrobeItems).Error; err != nil {
		return err
	}

	wardrobeByFashionID := make(map[uuid.UUID]*entities.WardrobeItem, len(wardrobeItems))
	for _, wardrobeItem := range wardrobeItems {
		wardrobeByFashionID[wardrobeItem.FashionItemID] = wardrobeItem
	}
	for _, item := range outfit.Items {
		if item.ItemContext == outfititemcontext.UserWardrobe {
			item.WardrobeItem = wardrobeByFashionID[item.FashionItemID]
		}
	}
	return nil
}

func (r *OutfitRepository) attachBrandItems(ctx context.Context, outfit *entities.Outfit) error {
	if outfit == nil || len(outfit.Items) == 0 {
		return nil
	}

	fashionIDs := make([]uuid.UUID, 0, len(outfit.Items))
	for _, item := range outfit.Items {
		if item.ItemContext == outfititemcontext.BrandItem {
			fashionIDs = append(fashionIDs, item.FashionItemID)
		}
	}
	if len(fashionIDs) == 0 {
		return nil
	}

	var brandItems []*entities.BrandItem
	if err := r.GetDB(ctx).
		Preload("Brand").
		Preload("FashionItem").
		Preload("FashionItem.Category").
		Where("fashion_item_id IN ?", fashionIDs).
		Find(&brandItems).Error; err != nil {
		return err
	}

	brandByFashionID := make(map[uuid.UUID]*entities.BrandItem, len(brandItems))
	for _, brandItem := range brandItems {
		brandByFashionID[brandItem.FashionItemID] = brandItem
	}
	for _, item := range outfit.Items {
		if item.ItemContext == outfititemcontext.BrandItem {
			item.BrandItem = brandByFashionID[item.FashionItemID]
		}
	}
	return nil
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
			if err := tx.Omit("Outfit", "FashionItem", "WardrobeItem").Create(&items).Error; err != nil {
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
			if err := tx.Omit("Outfit", "FashionItem", "WardrobeItem").Create(&items).Error; err != nil {
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
