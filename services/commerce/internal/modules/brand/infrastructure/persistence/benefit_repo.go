package persistence

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefitstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/repositories"
	shared_persist "smart-wardrobe-be/shared/infrastructure/repositories"

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
	err := r.GetQueryWithPreload(ctx).Where("brand_id = ? AND status = ?", brandID, string(benefitstatus.Active)).Find(&benefits).Error
	if err != nil {
		return nil, err
	}
	return benefits, nil
}

func (r *BrandBenefitRepository) GetActiveByBrandIDAndTierID(ctx context.Context, brandID, tierID uuid.UUID) ([]*entities.BrandBenefit, error) {
	var benefits []*entities.BrandBenefit
	err := r.GetQueryWithPreload(ctx).
		Where("brand_id = ? AND status = ? AND unlock_type = ? AND required_tier_id = ?",
			brandID, string(benefitstatus.Active), "tier_privilege", tierID).
		Find(&benefits).Error
	if err != nil {
		return nil, err
	}
	return benefits, nil
}

func (r *BrandBenefitRepository) GetActiveByBrandIDPaginated(ctx context.Context, filter repositories.BrandBenefitFilter) (*repositories.BrandBenefitListResult, error) {
	db := r.GetQueryWithPreload(ctx).Where("brand_id = ? AND status = ?", filter.BrandID, string(benefitstatus.Active))

	if filter.UnlockType != nil {
		db = db.Where("unlock_type = ?", *filter.UnlockType)
	}
	if filter.BenefitType != nil {
		db = db.Where("benefit_type = ?", *filter.BenefitType)
	}

	var totalCount int64
	if err := db.Model(&entities.BrandBenefit{}).Count(&totalCount).Error; err != nil {
		return nil, err
	}

	if filter.Page > 0 && filter.Limit > 0 {
		offset := (filter.Page - 1) * filter.Limit
		db = db.Offset(offset).Limit(filter.Limit)
	}

	var benefits []*entities.BrandBenefit
	if err := db.Order("created_at DESC").Find(&benefits).Error; err != nil {
		return nil, err
	}

	return &repositories.BrandBenefitListResult{
		Benefits:   benefits,
		TotalCount: totalCount,
	}, nil
}
