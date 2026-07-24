package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitredemptionstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

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
