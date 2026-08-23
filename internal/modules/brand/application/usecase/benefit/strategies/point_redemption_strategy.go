package strategies

import (
	"context"
	"math"
	"time"

	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitredemptionstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/loyaltytransactiontype"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"

	"github.com/google/uuid"
)

type PointRedemptionStrategy struct {
	accountRepo    repositories.ILoyaltyAccountRepository
	lotRepo        repositories.ILoyaltyPointLotRepository
	txRepo         repositories.ILoyaltyPointTransactionRepository
	redemptionRepo repositories.IBenefitRedemptionRepository
}

func NewPointRedemptionStrategy(
	accountRepo repositories.ILoyaltyAccountRepository,
	lotRepo repositories.ILoyaltyPointLotRepository,
	txRepo repositories.ILoyaltyPointTransactionRepository,
	redemptionRepo repositories.IBenefitRedemptionRepository,
) *PointRedemptionStrategy {
	return &PointRedemptionStrategy{
		accountRepo:    accountRepo,
		lotRepo:        lotRepo,
		txRepo:         txRepo,
		redemptionRepo: redemptionRepo,
	}
}

func (s *PointRedemptionStrategy) CheckAccess(ctx context.Context, benefit *entities.BrandBenefit, customer *entities.BrandCustomer, account *entities.LoyaltyAccount, now time.Time) (bool, error) {
	featureCode := ""
	if benefit.FeatureCode != nil {
		featureCode = string(*benefit.FeatureCode)
	}

	red, err := s.redemptionRepo.GetActiveRedemptionByFeature(ctx, customer.ID, featureCode, now)
	if err != nil {
		return false, err
	}
	return red != nil, nil
}

func (s *PointRedemptionStrategy) Redeem(ctx context.Context, benefit *entities.BrandBenefit, customer *entities.BrandCustomer, userID uuid.UUID, now time.Time) (*entities.BenefitRedemption, error) {
	featureCode := ""
	if benefit.FeatureCode != nil {
		featureCode = string(*benefit.FeatureCode)
	}

	existing, err := s.redemptionRepo.GetActiveRedemptionByFeature(ctx, customer.ID, featureCode, now)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, branderrors.ErrBenefitRedemptionExists()
	}

	reqPoints := 0
	if benefit.RequiredPoints != nil {
		reqPoints = *benefit.RequiredPoints
	}

	if reqPoints > 0 {
		account, err := s.accountRepo.GetByBrandCustomerIDForUpdate(ctx, customer.ID)
		if err != nil {
			return nil, err
		}
		if account == nil {
			return nil, branderrors.ErrInsufficientLoyaltyPoints()
		}

		if err := s.redeemLoyaltyPointsFromLots(ctx, account.ID, reqPoints, now, benefit.Name, userID); err != nil {
			return nil, err
		}
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
		PointsSpent:     reqPoints,
		Status:          benefitredemptionstatus.Redeemed,
		RedeemedAt:      now,
		ExpiresAt:       expiresAt,
	}

	return redemption, nil
}

func (s *PointRedemptionStrategy) redeemLoyaltyPointsFromLots(ctx context.Context, loyaltyAccountID uuid.UUID, requiredPoints int, now time.Time, benefitName string, userID uuid.UUID) error {
	account, err := s.accountRepo.GetByIDForUpdate(ctx, loyaltyAccountID)
	if err != nil {
		return err
	}
	if account == nil {
		return branderrors.ErrInsufficientLoyaltyPoints()
	}

	if err := s.expireDueLotsForAccount(ctx, loyaltyAccountID, now); err != nil {
		return err
	}

	account, err = s.accountRepo.GetByIDForUpdate(ctx, loyaltyAccountID)
	if err != nil {
		return err
	}
	if account == nil || account.CurrentPoints < requiredPoints {
		return branderrors.ErrInsufficientLoyaltyPoints()
	}

	lots, err := s.lotRepo.ListRedeemableLotsForUpdate(ctx, loyaltyAccountID, now)
	if err != nil {
		return err
	}

	remainingToRedeem := requiredPoints
	for _, lot := range lots {
		take := int(math.Min(float64(lot.RemainingPoints), float64(remainingToRedeem)))
		nextRemaining := lot.RemainingPoints - take
		nextStatus := loyaltypointlotstatus.Active
		if nextRemaining == 0 {
			nextStatus = loyaltypointlotstatus.Consumed
		}
		if err := s.lotRepo.UpdateLotRemainingAndStatus(ctx, lot.ID, nextRemaining, nextStatus); err != nil {
			return err
		}
		remainingToRedeem -= take
		if remainingToRedeem == 0 {
			break
		}
	}

	if remainingToRedeem > 0 {
		return branderrors.ErrInsufficientLoyaltyPoints()
	}

	account.CurrentPoints -= requiredPoints
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return err
	}

	reason := "Doi quyen loi: " + benefitName
	refType := "BENEFIT_REDEMPTION"
	tx := &entities.LoyaltyPointTransaction{
		LoyaltyAccountID: loyaltyAccountID,
		BrandID:          account.BrandID,
		BrandCustomerID:  account.BrandCustomerID,
		UserID:           account.UserID,
		PointsDelta:      -requiredPoints,
		BalanceAfter:     account.CurrentPoints,
		TransactionType:  loyaltytransactiontype.Redeem,
		Reason:           &reason,
		ReferenceType:    &refType,
		CreatedByUserID:  &userID,
	}
	return s.txRepo.Create(ctx, tx)
}

func (s *PointRedemptionStrategy) expireDueLotsForAccount(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time) error {
	account, err := s.accountRepo.GetByIDForUpdate(ctx, loyaltyAccountID)
	if err != nil {
		return err
	}
	if account == nil {
		return nil
	}

	expiredLots, err := s.lotRepo.ListExpiredLotsForUpdate(ctx, loyaltyAccountID, now)
	if err != nil {
		return err
	}

	expiredPoints := 0
	for _, lot := range expiredLots {
		expiredPoints += lot.RemainingPoints
	}
	if expiredPoints == 0 {
		return nil
	}

	if expiredPoints > account.CurrentPoints {
		expiredPoints = account.CurrentPoints
	}

	for _, lot := range expiredLots {
		if err := s.lotRepo.UpdateLotRemainingAndStatus(ctx, lot.ID, 0, loyaltypointlotstatus.Expired); err != nil {
			return err
		}
	}

	account.CurrentPoints -= expiredPoints
	if account.CurrentPoints < 0 {
		account.CurrentPoints = 0
	}
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return err
	}

	reason := "Expired loyalty points"
	referenceType := "POINT_EXPIRY"
	tx := &entities.LoyaltyPointTransaction{
		LoyaltyAccountID: loyaltyAccountID,
		BrandID:          account.BrandID,
		BrandCustomerID:  account.BrandCustomerID,
		UserID:           account.UserID,
		PointsDelta:      -expiredPoints,
		BalanceAfter:     account.CurrentPoints,
		TransactionType:  loyaltytransactiontype.Expire,
		Reason:           &reason,
		ReferenceType:    &referenceType,
	}
	return s.txRepo.Create(ctx, tx)
}
