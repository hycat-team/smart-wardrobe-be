package strategies

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitredemptionstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"

	"github.com/google/uuid"
)

type ManualGrantStrategy struct{}

func NewManualGrantStrategy() *ManualGrantStrategy {
	return &ManualGrantStrategy{}
}

func (s *ManualGrantStrategy) CheckAccess(_ context.Context, _ *entities.BrandBenefit, _ *entities.BrandCustomer, _ *entities.LoyaltyAccount, _ time.Time) (bool, error) {
	return false, nil
}

func (s *ManualGrantStrategy) Redeem(_ context.Context, benefit *entities.BrandBenefit, customer *entities.BrandCustomer, userID uuid.UUID, now time.Time) (*entities.BenefitRedemption, error) {
	var expiresAt *time.Time
	durationDays := parseValidDurationDays(benefit.FeatureConfig)
	if durationDays > 0 {
		t := now.Add(time.Duration(durationDays) * 24 * time.Hour)
		expiresAt = &t
	}

	redemption := &entities.BenefitRedemption{
		BenefitID:       benefit.ID,
		BrandID:         benefit.BrandID,
		BrandCustomerID: customer.ID,
		UserID:          &userID,
		PointsSpent:     0,
		Status:          benefitredemptionstatus.Redeemed,
		RedeemedAt:      now,
		ExpiresAt:       expiresAt,
	}

	return redemption, nil
}
