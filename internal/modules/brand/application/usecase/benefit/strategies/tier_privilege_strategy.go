package strategies

import (
	"context"
	"time"

	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitredemptionstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"

	"github.com/google/uuid"
)

type TierPrivilegeStrategy struct {
	accountRepo    repositories.ILoyaltyAccountRepository
	tierRepo       repositories.ILoyaltyTierRepository
	redemptionRepo repositories.IBenefitRedemptionRepository
}

func NewTierPrivilegeStrategy(
	accountRepo repositories.ILoyaltyAccountRepository,
	tierRepo repositories.ILoyaltyTierRepository,
	redemptionRepo repositories.IBenefitRedemptionRepository,
) *TierPrivilegeStrategy {
	return &TierPrivilegeStrategy{
		accountRepo:    accountRepo,
		tierRepo:       tierRepo,
		redemptionRepo: redemptionRepo,
	}
}

func (s *TierPrivilegeStrategy) CheckAccess(ctx context.Context, benefit *entities.BrandBenefit, customer *entities.BrandCustomer, account *entities.LoyaltyAccount, now time.Time) (bool, error) {
	if benefit.RequiredTierID == nil {
		return false, nil
	}

	if account == nil || account.CurrentTierID == nil {
		return false, nil
	}

	userTier, err := s.tierRepo.GetByID(ctx, *account.CurrentTierID)
	if err != nil || userTier == nil {
		return false, nil
	}

	requiredTier, err := s.tierRepo.GetByID(ctx, *benefit.RequiredTierID)
	if err != nil || requiredTier == nil {
		return false, nil
	}

	return userTier.Rank >= requiredTier.Rank, nil
}

func (s *TierPrivilegeStrategy) Redeem(ctx context.Context, benefit *entities.BrandBenefit, customer *entities.BrandCustomer, userID uuid.UUID, now time.Time) (*entities.BenefitRedemption, error) {
	if benefit.RequiredTierID == nil {
		return nil, branderrors.ErrBenefitRequiredTierNotMet()
	}

	account, err := s.accountRepo.GetByBrandCustomerID(ctx, customer.ID)
	if err != nil {
		return nil, err
	}
	if account == nil || account.CurrentTierID == nil {
		return nil, branderrors.ErrBenefitRequiredTierNotMet()
	}

	userTier, err := s.tierRepo.GetByID(ctx, *account.CurrentTierID)
	if err != nil || userTier == nil {
		return nil, branderrors.ErrBenefitRequiredTierNotMet()
	}

	requiredTier, err := s.tierRepo.GetByID(ctx, *benefit.RequiredTierID)
	if err != nil || requiredTier == nil {
		return nil, branderrors.ErrBenefitRequiredTierNotMet()
	}

	if userTier.Rank < requiredTier.Rank {
		return nil, branderrors.ErrBenefitRequiredTierNotMet()
	}

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
