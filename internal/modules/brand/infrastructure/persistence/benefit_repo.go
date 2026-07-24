package persistence

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BrandBenefitRepository struct {
	shared_persist.GenericRepository[entities.BrandBenefit, uuid.UUID]
}

func NewBrandBenefitRepository(db *gorm.DB) repositories.IBrandBenefitRepository {
	relations := []string{"Brand", "RequiredTier"}
	return &BrandBenefitRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.BrandBenefit, uuid.UUID](db, relations),
	}
}

func (r *BrandBenefitRepository) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandBenefit, error) {
	var benefits []*entities.BrandBenefit
	err := r.GetQueryWithPreload(ctx).Where("brand_id = ?", brandID).Find(&benefits).Error
	if err != nil {
		return nil, err
	}
	return benefits, nil
}

func (r *BrandBenefitRepository) GetActiveByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandBenefit, error) {
	var benefits []*entities.BrandBenefit
	err := r.GetQueryWithPreload(ctx).Where("brand_id = ? AND status = ?", brandID, "active").Find(&benefits).Error
	if err != nil {
		return nil, err
	}
	return benefits, nil
}


