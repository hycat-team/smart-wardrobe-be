package persistence

import (
	"context"
	"strings"

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

func (r *BrandBenefitRepository) GetByBrandIDPaginated(ctx context.Context, filter repositories.BrandBenefitFilter) (*repositories.BrandBenefitListResult, error) {
	db := r.GetQueryWithPreload(ctx).Where("brand_id = ?", filter.BrandID)

	if filter.Status != nil && *filter.Status != "" {
		db = db.Where("LOWER(status) = ?", strings.ToLower(string(*filter.Status)))
	}

	if filter.UnlockType != nil {
		db = db.Where("unlock_type = ?", *filter.UnlockType)
	}
	if filter.BenefitType != nil {
		db = db.Where("benefit_type = ?", *filter.BenefitType)
	}
	if filter.RequiredTierID != nil {
		db = db.Where("required_tier_id = ?", *filter.RequiredTierID)
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

func (r *BrandBenefitRepository) CountActiveByTierIDs(ctx context.Context, brandID uuid.UUID, tierIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	counts := make(map[uuid.UUID]int64, len(tierIDs))
	if len(tierIDs) == 0 {
		return counts, nil
	}

	type tierCountRow struct {
		RequiredTierID uuid.UUID `gorm:"column:required_tier_id"`
		Count          int64     `gorm:"column:count"`
	}

	var rows []tierCountRow
	err := r.GetDB(ctx).
		Model(&entities.BrandBenefit{}).
		Select("required_tier_id, COUNT(*) AS count").
		Where("brand_id = ? AND LOWER(status) = ? AND required_tier_id IN ?", brandID, strings.ToLower(string(benefitstatus.Active)), tierIDs).
		Group("required_tier_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		counts[row.RequiredTierID] = row.Count
	}
	return counts, nil
}
