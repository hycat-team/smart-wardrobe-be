package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefitredemptionstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/repositories"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shared_persist "smart-wardrobe-be/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BenefitRedemptionRepository struct {
	shared_persist.GenericRepository[entities.BenefitRedemption, uuid.UUID]
}

func NewBenefitRedemptionRepository(db *gorm.DB) repositories.IBenefitRedemptionRepository {
	relations := []string{"Benefit", "Brand", "BrandCustomer"}
	return &BenefitRedemptionRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.BenefitRedemption, uuid.UUID](db, relations),
	}
}

func (r *BenefitRedemptionRepository) GetByBrandCustomerID(ctx context.Context, brandCustomerID uuid.UUID) ([]*entities.BenefitRedemption, error) {
	var redemptions []*entities.BenefitRedemption
	err := r.GetDB(ctx).Where("brand_customer_id = ?", brandCustomerID).Find(&redemptions).Error
	if err != nil {
		return nil, err
	}
	return redemptions, nil
}

func (r *BenefitRedemptionRepository) GetByBrandCustomerIDs(ctx context.Context, brandCustomerIDs []uuid.UUID) ([]*entities.BenefitRedemption, error) {
	if len(brandCustomerIDs) == 0 {
		return []*entities.BenefitRedemption{}, nil
	}
	var redemptions []*entities.BenefitRedemption
	err := r.GetDB(ctx).Where("brand_customer_id IN ?", brandCustomerIDs).Find(&redemptions).Error
	if err != nil {
		return nil, err
	}
	return redemptions, nil
}

func (r *BenefitRedemptionRepository) GetByFilter(ctx context.Context, filter repositories.BenefitRedemptionFilter) (*repositories.BenefitRedemptionListResult, error) {
	if len(filter.BrandCustomerIDs) == 0 {
		return &repositories.BenefitRedemptionListResult{}, nil
	}

	db := r.GetQueryWithPreload(ctx).Where("brand_customer_id IN ?", filter.BrandCustomerIDs)

	if filter.BrandID != nil {
		db = db.Where("brand_id = ?", *filter.BrandID)
	}

	var totalCount int64
	if err := db.Model(&entities.BenefitRedemption{}).Count(&totalCount).Error; err != nil {
		return nil, err
	}

	paginationQuery := shared_dto.PaginationQuery{Page: filter.Page, Limit: filter.Limit}
	db = shared_persist.ApplyPagination(db, paginationQuery)

	var items []*entities.BenefitRedemption
	if err := db.Order("redeemed_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}

	return &repositories.BenefitRedemptionListResult{
		Items:      items,
		TotalCount: totalCount,
	}, nil
}

func (r *BenefitRedemptionRepository) GetActiveRedemptionByFeature(ctx context.Context, brandCustomerID uuid.UUID, featureCode string, now time.Time) (*entities.BenefitRedemption, error) {
	var redemption entities.BenefitRedemption
	err := r.GetDB(ctx).
		Joins("JOIN brand_benefits ON brand_benefits.id = benefit_redemptions.benefit_id").
		Where("benefit_redemptions.brand_customer_id = ? AND benefit_redemptions.status = ? AND brand_benefits.feature_code = ? AND (benefit_redemptions.expires_at IS NULL OR benefit_redemptions.expires_at > ?)",
			brandCustomerID, string(benefitredemptionstatus.Redeemed), featureCode, now).
		First(&redemption).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &redemption, nil
}

func (r *BenefitRedemptionRepository) GetUserExpiringBenefitsCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entities.BenefitRedemption{}).
		Where("user_id = ? AND status = ? AND expires_at BETWEEN now() AND now() + interval '30 days'", userID, string(benefitredemptionstatus.Redeemed)).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r *BenefitRedemptionRepository) GetRedemptionAnalyticsByBrand(ctx context.Context, brandID uuid.UUID) ([]repositories.BenefitRedemptionAnalytics, error) {
	var rows []repositories.BenefitRedemptionAnalytics
	err := r.GetDB(ctx).
		Table("benefit_redemptions").
		Select("bb.id AS benefit_id, bb.name AS benefit_name, bb.benefit_type, "+
			"COUNT(*) AS redemption_count, DATE_TRUNC('month', benefit_redemptions.redeemed_at)::date AS month").
		Joins("JOIN brand_benefits bb ON bb.id = benefit_redemptions.benefit_id").
		Where("benefit_redemptions.brand_id = ?", brandID).
		Group("bb.id, bb.name, bb.benefit_type, DATE_TRUNC('month', benefit_redemptions.redeemed_at)").
		Order("redemption_count DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *BenefitRedemptionRepository) CountRedemptionsByBrand(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (int64, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entities.BenefitRedemption{}).
		Where("brand_id = ? AND redeemed_at >= ? AND redeemed_at <= ?", brandID, fromDate, toDate).
		Count(&count).Error
	return count, err
}
