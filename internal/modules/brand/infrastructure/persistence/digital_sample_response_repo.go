package persistence

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DigitalSampleResponseRepository struct {
	shared_persist.GenericRepository[entities.DigitalSampleResponse, uuid.UUID]
}

func NewDigitalSampleResponseRepository(db *gorm.DB) repositories.IDigitalSampleResponseRepository {
	relations := []string{"BrandItem"}
	return &DigitalSampleResponseRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.DigitalSampleResponse, uuid.UUID](db, relations),
	}
}

func (r *DigitalSampleResponseRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.DigitalSampleResponse, error) {
	var responses []*entities.DigitalSampleResponse
	err := r.GetDB(ctx).Preload("BrandItem").Where("user_id = ?", userID).Find(&responses).Error
	if err != nil {
		return nil, err
	}
	return responses, nil
}

func (r *DigitalSampleResponseRepository) GetByBrandItemID(ctx context.Context, brandItemID uuid.UUID) ([]*entities.DigitalSampleResponse, error) {
	var responses []*entities.DigitalSampleResponse
	err := r.GetDB(ctx).Where("brand_item_id = ?", brandItemID).Find(&responses).Error
	if err != nil {
		return nil, err
	}
	return responses, nil
}
