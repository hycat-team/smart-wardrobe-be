package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
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
	err := r.GetDB(ctx).Where("brand_id = ?", brandID).Find(&benefits).Error
	if err != nil {
		return nil, err
	}
	return benefits, nil
}

func (r *BrandBenefitRepository) GetActiveByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandBenefit, error) {
	var benefits []*entities.BrandBenefit
	err := r.GetDB(ctx).Where("brand_id = ? AND status = ?", brandID, "active").Find(&benefits).Error
	if err != nil {
		return nil, err
	}
	return benefits, nil
}

type BenefitRedemptionRepository struct {
	shared_persist.GenericRepository[entities.BenefitRedemption, uuid.UUID]
}

func NewBenefitRedemptionRepository(db *gorm.DB) repositories.IBenefitRedemptionRepository {
	relations := []string{"Benefit", "Brand", "BrandCustomer", "User"}
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

func (r *BenefitRedemptionRepository) GetActiveRedemptionByFeature(ctx context.Context, brandCustomerID uuid.UUID, featureCode string, now time.Time) (*entities.BenefitRedemption, error) {
	var redemption entities.BenefitRedemption
	// Join brand_benefits to filter by feature_code and check if redemption status is REDEEMED
	// and expires_at is either null or in the future
	err := r.GetDB(ctx).
		Joins("JOIN brand_benefits ON brand_benefits.id = benefit_redemptions.benefit_id").
		Where("benefit_redemptions.brand_customer_id = ? AND benefit_redemptions.status = ? AND brand_benefits.feature_code = ? AND (benefit_redemptions.expires_at IS NULL OR benefit_redemptions.expires_at > ?)",
			brandCustomerID, "redeemed", featureCode, now).
		First(&redemption).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &redemption, nil
}
