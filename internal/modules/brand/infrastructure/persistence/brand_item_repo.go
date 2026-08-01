package persistence

import (
	"context"
	"errors"

	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BrandItemRepository struct {
	shared_persist.GenericRepository[entities.BrandItem, uuid.UUID]
}

func NewBrandItemRepository(db *gorm.DB) repositories.IBrandItemRepository {
	relations := []string{"Brand"}
	return &BrandItemRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.BrandItem, uuid.UUID](db, relations),
	}
}

func (r *BrandItemRepository) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandItem, error) {
	var items []*entities.BrandItem
	err := r.GetDB(ctx).Where("brand_id = ?", brandID).Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *BrandItemRepository) GetByBrandIDs(ctx context.Context, brandIDs []uuid.UUID) ([]*entities.BrandItem, error) {
	if len(brandIDs) == 0 {
		return []*entities.BrandItem{}, nil
	}
	var items []*entities.BrandItem
	err := r.GetDB(ctx).
		Where("brand_id IN ?", brandIDs).
		Order("created_at DESC, item_type DESC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *BrandItemRepository) GetByProductCode(ctx context.Context, brandID uuid.UUID, code string) (*entities.BrandItem, error) {
	var item entities.BrandItem
	err := r.GetDB(ctx).Where("brand_id = ? AND product_code = ?", brandID, code).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *BrandItemRepository) GetByFashionItemID(ctx context.Context, fashionItemID uuid.UUID) (*entities.BrandItem, error) {
	var item entities.BrandItem
	err := r.GetDB(ctx).Preload("Brand").Where("fashion_item_id = ?", fashionItemID).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *BrandItemRepository) GetByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*entities.BrandItem, error) {
	if len(fashionItemIDs) == 0 {
		return nil, nil
	}
	var items []*entities.BrandItem
	err := r.GetDB(ctx).Where("fashion_item_id IN ?", fashionItemIDs).Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *BrandItemRepository) GetByBrandIDPaginated(ctx context.Context, filter repositories.BrandItemFilter) (*repositories.BrandItemListResult, error) {
	db := r.GetDB(ctx).Where("brand_id = ?", filter.BrandID)

	if filter.ItemType != nil && *filter.ItemType != "" {
		db = db.Where("item_type = ?", *filter.ItemType)
	}
	if filter.Status != nil && *filter.Status != "" {
		db = db.Where("status = ?", *filter.Status)
	}

	var totalCount int64
	if err := db.Model(&entities.BrandItem{}).Count(&totalCount).Error; err != nil {
		return nil, err
	}

	paginationQuery := shared_dto.PaginationQuery{
		Page:  filter.Page,
		Limit: filter.Limit,
	}
	db = shared_persist.ApplyPagination(db, paginationQuery)

	var items []*entities.BrandItem
	if err := db.Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}

	return &repositories.BrandItemListResult{
		Items:      items,
		TotalCount: totalCount,
	}, nil
}

func (r *BrandItemRepository) CountByStatus(ctx context.Context, brandID uuid.UUID) ([]repositories.BrandItemStatusCount, error) {
	var rows []repositories.BrandItemStatusCount
	err := r.GetDB(ctx).Model(&entities.BrandItem{}).
		Select("status AS status, COUNT(*) AS count").
		Where("brand_id = ?", brandID).
		Group("status").
		Order("count DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *BrandItemRepository) CountByType(ctx context.Context, brandID uuid.UUID) ([]repositories.BrandItemTypeCount, error) {
	var rows []repositories.BrandItemTypeCount
	err := r.GetDB(ctx).Model(&entities.BrandItem{}).
		Select("item_type AS item_type, COUNT(*) AS count").
		Where("brand_id = ?", brandID).
		Group("item_type").
		Order("count DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *BrandItemRepository) CountByBrandID(ctx context.Context, brandID uuid.UUID) (int64, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entities.BrandItem{}).
		Where("brand_id = ?", brandID).
		Count(&count).Error
	return count, err
}
