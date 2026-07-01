package persistence

import (
	"context"
	"errors"

	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BrandItemRepository struct {
	shared_persist.GenericRepository[entities.BrandItem, uuid.UUID]
}

func NewBrandItemRepository(db *gorm.DB) repositories.IBrandItemRepository {
	relations := []string{"Brand", "FashionItem", "FashionItem.Category"}
	return &BrandItemRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.BrandItem, uuid.UUID](db, relations),
	}
}

func (r *BrandItemRepository) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandItem, error) {
	var items []*entities.BrandItem
	err := r.GetDB(ctx).Preload("FashionItem").Preload("FashionItem.Category").Where("brand_id = ?", brandID).Find(&items).Error
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
		Preload("FashionItem").
		Preload("FashionItem.Category").
		Where("brand_id IN ?", brandIDs).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *BrandItemRepository) GetByProductCode(ctx context.Context, brandID uuid.UUID, code string) (*entities.BrandItem, error) {
	var item entities.BrandItem
	err := r.GetDB(ctx).Preload("FashionItem").Preload("FashionItem.Category").Where("brand_id = ? AND product_code = ?", brandID, code).First(&item).Error
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
	err := r.GetDB(ctx).Preload("FashionItem").Preload("FashionItem.Category").Preload("Brand").Where("fashion_item_id = ?", fashionItemID).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

type DigitalSampleResponseRepository struct {
	shared_persist.GenericRepository[entities.DigitalSampleResponse, uuid.UUID]
}

func NewDigitalSampleResponseRepository(db *gorm.DB) repositories.IDigitalSampleResponseRepository {
	relations := []string{"BrandItem", "User", "Outfit"}
	return &DigitalSampleResponseRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.DigitalSampleResponse, uuid.UUID](db, relations),
	}
}

func (r *DigitalSampleResponseRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.DigitalSampleResponse, error) {
	var responses []*entities.DigitalSampleResponse
	err := r.GetDB(ctx).Preload("BrandItem").Preload("BrandItem.FashionItem").Where("user_id = ?", userID).Find(&responses).Error
	if err != nil {
		return nil, err
	}
	return responses, nil
}

func (r *DigitalSampleResponseRepository) GetByBrandItemID(ctx context.Context, brandItemID uuid.UUID) ([]*entities.DigitalSampleResponse, error) {
	var responses []*entities.DigitalSampleResponse
	err := r.GetDB(ctx).Preload("User").Where("brand_item_id = ?", brandItemID).Find(&responses).Error
	if err != nil {
		return nil, err
	}
	return responses, nil
}
