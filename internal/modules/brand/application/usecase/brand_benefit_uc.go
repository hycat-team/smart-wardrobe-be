package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/benefit/benefitfeaturecode"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/benefit/benefitredemptionstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/benefit/benefitstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/benefit/benefittype"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/benefit/benefitunlocktype"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltytransactiontype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type BrandBenefitUseCase struct {
	brandRepo      repositories.IBrandRepository
	memberRepo     repositories.IBrandMemberRepository
	customerRepo   repositories.IBrandCustomerRepository
	accountRepo    repositories.ILoyaltyAccountRepository
	txRepo         repositories.ILoyaltyPointTransactionRepository
	lotRepo        repositories.ILoyaltyPointLotRepository
	benefitRepo    repositories.IBrandBenefitRepository
	redemptionRepo repositories.IBenefitRedemptionRepository
	tierRepo       repositories.ILoyaltyTierRepository
	uow            shared_repos.IUnitOfWork
}

func NewBrandBenefitUseCase(
	brandRepo repositories.IBrandRepository,
	memberRepo repositories.IBrandMemberRepository,
	customerRepo repositories.IBrandCustomerRepository,
	accountRepo repositories.ILoyaltyAccountRepository,
	txRepo repositories.ILoyaltyPointTransactionRepository,
	lotRepo repositories.ILoyaltyPointLotRepository,
	benefitRepo repositories.IBrandBenefitRepository,
	redemptionRepo repositories.IBenefitRedemptionRepository,
	tierRepo repositories.ILoyaltyTierRepository,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IBrandBenefitUseCase {
	return &BrandBenefitUseCase{
		brandRepo:      brandRepo,
		memberRepo:     memberRepo,
		customerRepo:   customerRepo,
		accountRepo:    accountRepo,
		txRepo:         txRepo,
		lotRepo:        lotRepo,
		benefitRepo:    benefitRepo,
		redemptionRepo: redemptionRepo,
		tierRepo:       tierRepo,
		uow:            uow,
	}
}

func (uc *BrandBenefitUseCase) CreateBrandBenefit(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateBrandBenefitReq) (*dto.BrandBenefitRes, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}

	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	bType := benefittype.BenefitType(strings.ToLower(input.BenefitType))
	uType := benefitunlocktype.BenefitUnlockType(strings.ToLower(input.UnlockType))

	benefit := &entities.BrandBenefit{
		BrandID:     brandID,
		Name:        strings.TrimSpace(input.Name),
		Description: input.Description,
		BenefitType: bType,
		UnlockType:  uType,
		Status:      benefitstatus.Active,
	}

	if uType == benefitunlocktype.PointRedemption {
		if input.RequiredPoints == nil || *input.RequiredPoints <= 0 {
			return nil, branderrors.ErrPurchaseAmountOrPointsRequired()
		}
		benefit.RequiredPoints = input.RequiredPoints
	} else if uType == benefitunlocktype.TierPrivilege {
		if input.RequiredTierID == nil {
			return nil, branderrors.ErrInvalidBrandMemberRole("Yeu cau phai co requiredTierId")
		}
		tier, err := uc.tierRepo.GetByID(ctx, *input.RequiredTierID)
		if err != nil {
			return nil, err
		}
		if tier == nil || tier.BrandID != brandID {
			return nil, branderrors.ErrBenefitInvalidStatus()
		}
		benefit.RequiredTierID = input.RequiredTierID
	}

	if input.FeatureCode != nil {
		fCode := benefitfeaturecode.BenefitFeatureCode(strings.ToLower(*input.FeatureCode))
		benefit.FeatureCode = &fCode
	}

	if input.FeatureConfig != nil {
		bytes, err := json.Marshal(input.FeatureConfig)
		if err != nil {
			return nil, branderrors.ErrBenefitInvalidStatus()
		}
		benefit.FeatureConfig = entities.JSONDocument(bytes)
	}

	if err := uc.benefitRepo.Create(ctx, benefit); err != nil {
		return nil, err
	}

	return mapper.MapBrandBenefit(benefit), nil
}

func (uc *BrandBenefitUseCase) ListBrandBenefitsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandBenefitRes, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}
	benefits, err := uc.benefitRepo.GetByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrandBenefits(benefits), nil
}

func (uc *BrandBenefitUseCase) ListActiveBenefitsForUser(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandBenefitRes, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	benefits, err := uc.benefitRepo.GetActiveByBrandID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	return mapper.MapBrandBenefits(benefits), nil
}

func (uc *BrandBenefitUseCase) GetActiveBenefitForUser(ctx context.Context, userID uuid.UUID, benefitID uuid.UUID) (*dto.BrandBenefitRes, error) {
	benefit, err := uc.benefitRepo.GetByID(ctx, benefitID)
	if err != nil {
		return nil, err
	}
	if benefit == nil {
		return nil, branderrors.ErrBenefitNotFound()
	}
	// Verify customer status via user loyalty helper (local check)
	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, benefit.BrandID, userID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return nil, branderrors.ErrCustomerNotFound()
	}
	if benefit.Status != benefitstatus.Active {
		return nil, branderrors.ErrBenefitNotActive()
	}
	return mapper.MapBrandBenefit(benefit), nil
}

func (uc *BrandBenefitUseCase) ListBenefitRedemptionsForUser(ctx context.Context, userID uuid.UUID) ([]*dto.BenefitRedemptionRes, error) {
	customers, err := uc.customerRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	customerIDs := make([]uuid.UUID, 0, len(customers))
	for _, customer := range customers {
		if customer.Status != brandcustomerstatus.Active {
			continue
		}
		customerIDs = append(customerIDs, customer.ID)
	}
	redemptions, err := uc.redemptionRepo.GetByBrandCustomerIDs(ctx, customerIDs)
	if err != nil {
		return nil, err
	}
	res := make([]*dto.BenefitRedemptionRes, 0, len(redemptions))
	for _, redemption := range redemptions {
		res = append(res, mapper.MapBenefitRedemption(redemption))
	}
	return res, nil
}

func (uc *BrandBenefitUseCase) UpdateBenefitStatus(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, benefitID uuid.UUID, status string) (*dto.BrandBenefitRes, error) {
	if err := requireBrandRoleShared(ctx, uc.brandRepo, uc.memberRepo, staffUserID, brandID, brandmemberrole.Owner, brandmemberrole.Staff); err != nil {
		return nil, err
	}

	benefit, err := uc.benefitRepo.GetByID(ctx, benefitID)
	if err != nil {
		return nil, err
	}
	if benefit == nil || benefit.BrandID != brandID {
		return nil, branderrors.ErrBenefitNotFound()
	}

	bStatus := benefitstatus.BenefitStatus(strings.ToLower(status))
	if bStatus != benefitstatus.Active && bStatus != benefitstatus.Inactive && bStatus != benefitstatus.Archived {
		return nil, branderrors.ErrBenefitInvalidStatus()
	}

	benefit.Status = bStatus
	if err := uc.benefitRepo.Update(ctx, benefit); err != nil {
		return nil, err
	}

	return mapper.MapBrandBenefit(benefit), nil
}

func (uc *BrandBenefitUseCase) CheckBrandFeatureAccess(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, featureCode string) (bool, error) {
	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return false, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return false, nil
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return false, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return false, nil
	}

	benefits, err := uc.benefitRepo.GetActiveByBrandID(ctx, brandID)
	if err != nil {
		return false, err
	}

	now := time.Now().UTC()
	fCode := benefitfeaturecode.BenefitFeatureCode(strings.ToLower(featureCode))

	for _, benefit := range benefits {
		if benefit.BenefitType != benefittype.FeatureAccess || benefit.FeatureCode == nil || *benefit.FeatureCode != fCode {
			continue
		}

		if benefit.UnlockType == benefitunlocktype.TierPrivilege {
			account, err := uc.accountRepo.GetByBrandCustomerID(ctx, customer.ID)
			if err != nil || account == nil || account.CurrentTierID == nil {
				continue
			}

			userTier, err := uc.tierRepo.GetByID(ctx, *account.CurrentTierID)
			if err != nil || userTier == nil {
				continue
			}

			requiredTier, err := uc.tierRepo.GetByID(ctx, *benefit.RequiredTierID)
			if err != nil || requiredTier == nil {
				continue
			}

			if userTier.Rank >= requiredTier.Rank {
				return true, nil
			}
		} else if benefit.UnlockType == benefitunlocktype.PointRedemption || benefit.UnlockType == benefitunlocktype.ManualGrant {
			red, err := uc.redemptionRepo.GetActiveRedemptionByFeature(ctx, customer.ID, featureCode, now)
			if err != nil {
				return false, err
			}
			if red != nil {
				return true, nil
			}
		}
	}

	return false, nil
}

func (uc *BrandBenefitUseCase) RedeemBenefit(ctx context.Context, userID uuid.UUID, benefitID uuid.UUID) (*dto.BenefitRedemptionRes, error) {
	benefit, err := uc.benefitRepo.GetByID(ctx, benefitID)
	if err != nil {
		return nil, err
	}
	if benefit == nil {
		return nil, branderrors.ErrBenefitNotFound()
	}
	brandID := benefit.BrandID

	brand, err := uc.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil || brand.Status != brandstatus.Active {
		return nil, branderrors.ErrBrandNotActive()
	}

	customer, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, userID)
	if err != nil {
		return nil, err
	}
	if customer == nil || customer.Status != brandcustomerstatus.Active {
		return nil, branderrors.ErrUserNotFoundOrInactive()
	}

	if benefit.Status != benefitstatus.Active {
		return nil, branderrors.ErrBenefitNotActive()
	}

	now := time.Now().UTC()
	var redemption *entities.BenefitRedemption

	if benefit.UnlockType == benefitunlocktype.PointRedemption {
		err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
			account, err := uc.accountRepo.GetByBrandCustomerIDForUpdate(txCtx, customer.ID)
			if err != nil {
				return err
			}
			if account == nil {
				return branderrors.ErrInsufficientLoyaltyPoints()
			}

			var featStr string
			if benefit.FeatureCode != nil {
				featStr = string(*benefit.FeatureCode)
			}
			existing, err := uc.redemptionRepo.GetActiveRedemptionByFeature(txCtx, customer.ID, featStr, now)
			if err != nil {
				return err
			}
			if existing != nil {
				return branderrors.ErrBenefitRedemptionExists()
			}

			reqPoints := 0
			if benefit.RequiredPoints != nil {
				reqPoints = *benefit.RequiredPoints
			}

			reason := "Doi quyen loi: " + benefit.Name
			refType := "BENEFIT_REDEMPTION"
			refID := benefit.ID
			_, err = uc.redeemLoyaltyPointsFromLots(txCtx, account.ID, reqPoints, now, &reason, &refType, &refID, &userID)
			if err != nil {
				return err
			}

			var expiresAt *time.Time
			durationDays := parseValidDurationDays(benefit.FeatureConfig)
			if durationDays > 0 {
				t := now.Add(time.Duration(durationDays) * 24 * time.Hour)
				expiresAt = &t
			}

			redemption = &entities.BenefitRedemption{
				BenefitID:       benefit.ID,
				BrandID:         brandID,
				BrandCustomerID: customer.ID,
				UserID:          &userID,
				PointsSpent:     reqPoints,
				Status:          benefitredemptionstatus.Redeemed,
				RedeemedAt:      now,
				ExpiresAt:       expiresAt,
			}

			return uc.redemptionRepo.Create(txCtx, redemption)
		})
		if err != nil {
			return nil, err
		}
	} else if benefit.UnlockType == benefitunlocktype.TierPrivilege {
		account, err := uc.accountRepo.GetByBrandCustomerID(ctx, customer.ID)
		if err != nil {
			return nil, err
		}
		if account == nil || account.CurrentTierID == nil {
			return nil, branderrors.ErrBenefitRequiredTierNotMet()
		}

		userTier, err := uc.tierRepo.GetByID(ctx, *account.CurrentTierID)
		if err != nil || userTier == nil {
			return nil, branderrors.ErrBenefitRequiredTierNotMet()
		}

		requiredTier, err := uc.tierRepo.GetByID(ctx, *benefit.RequiredTierID)
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

		redemption = &entities.BenefitRedemption{
			BenefitID:       benefit.ID,
			BrandID:         brandID,
			BrandCustomerID: customer.ID,
			UserID:          &userID,
			PointsSpent:     0,
			Status:          benefitredemptionstatus.Redeemed,
			RedeemedAt:      now,
			ExpiresAt:       expiresAt,
		}

		if err := uc.redemptionRepo.Create(ctx, redemption); err != nil {
			return nil, err
		}
	} else {
		return nil, branderrors.ErrBenefitUnlockTypeNotSupported()
	}

	return mapper.MapBenefitRedemption(redemption), nil
}

func (uc *BrandBenefitUseCase) redeemLoyaltyPointsFromLots(ctx context.Context, loyaltyAccountID uuid.UUID, requiredPoints int, now time.Time, reason *string, referenceType *string, referenceID *uuid.UUID, createdByUserID *uuid.UUID) (*entities.LoyaltyPointTransaction, error) {
	if requiredPoints <= 0 {
		return nil, branderrors.ErrPointsDeltaZero()
	}
	account, err := uc.accountRepo.GetByIDForUpdate(ctx, loyaltyAccountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, branderrors.ErrInsufficientLoyaltyPoints()
	}
	if _, err := uc.expireDueLotsForAccount(ctx, loyaltyAccountID, now, nil); err != nil {
		return nil, err
	}
	account, err = uc.accountRepo.GetByIDForUpdate(ctx, loyaltyAccountID)
	if err != nil {
		return nil, err
	}
	if account == nil || account.CurrentPoints < requiredPoints {
		return nil, branderrors.ErrInsufficientLoyaltyPoints()
	}
	lots, err := uc.lotRepo.ListRedeemableLotsForUpdate(ctx, loyaltyAccountID, now)
	if err != nil {
		return nil, err
	}
	remainingToRedeem := requiredPoints
	for _, lot := range lots {
		take := minInt(lot.RemainingPoints, remainingToRedeem)
		nextRemaining := lot.RemainingPoints - take
		nextStatus := loyaltypointlotstatus.Active
		if nextRemaining == 0 {
			nextStatus = loyaltypointlotstatus.Consumed
		}
		if err := uc.lotRepo.UpdateLotRemainingAndStatus(ctx, lot.ID, nextRemaining, nextStatus); err != nil {
			return nil, err
		}
		remainingToRedeem -= take
		if remainingToRedeem == 0 {
			break
		}
	}
	if remainingToRedeem > 0 {
		return nil, branderrors.ErrInsufficientLoyaltyPoints()
	}
	account.CurrentPoints -= requiredPoints
	if err := uc.accountRepo.Update(ctx, account); err != nil {
		return nil, err
	}
	tx := &entities.LoyaltyPointTransaction{
		LoyaltyAccountID: loyaltyAccountID,
		BrandID:          account.BrandID,
		BrandCustomerID:  account.BrandCustomerID,
		UserID:           account.UserID,
		PointsDelta:      -requiredPoints,
		BalanceAfter:     account.CurrentPoints,
		TransactionType:  loyaltytransactiontype.Redeem,
		Reason:           reason,
		ReferenceType:    referenceType,
		ReferenceID:      referenceID,
		CreatedByUserID:  createdByUserID,
	}
	if err := uc.txRepo.Create(ctx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (uc *BrandBenefitUseCase) expireDueLotsForAccount(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time, createdByUserID *uuid.UUID) (int, error) {
	account, err := uc.accountRepo.GetByIDForUpdate(ctx, loyaltyAccountID)
	if err != nil {
		return 0, err
	}
	if account == nil {
		return 0, nil
	}
	expiredLots, err := uc.lotRepo.ListExpiredLotsForUpdate(ctx, loyaltyAccountID, now)
	if err != nil {
		return 0, err
	}
	expiredPoints := 0
	for _, lot := range expiredLots {
		expiredPoints += lot.RemainingPoints
	}
	if expiredPoints == 0 {
		return 0, nil
	}
	if expiredPoints > account.CurrentPoints {
		expiredPoints = account.CurrentPoints
	}
	for _, lot := range expiredLots {
		if err := uc.lotRepo.UpdateLotRemainingAndStatus(ctx, lot.ID, 0, loyaltypointlotstatus.Expired); err != nil {
			return 0, err
		}
	}
	account.CurrentPoints -= expiredPoints
	if account.CurrentPoints < 0 {
		account.CurrentPoints = 0
	}
	if err := uc.accountRepo.Update(ctx, account); err != nil {
		return 0, err
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
		CreatedByUserID:  createdByUserID,
	}
	if err := uc.txRepo.Create(ctx, tx); err != nil {
		return 0, err
	}
	return expiredPoints, nil
}

func parseValidDurationDays(doc entities.JSONDocument) int {
	if len(doc) == 0 {
		return 0
	}
	var config map[string]interface{}
	if err := json.Unmarshal(doc, &config); err != nil {
		return 0
	}
	if val, ok := config["valid_duration_days"]; ok {
		if floatVal, ok := val.(float64); ok {
			return int(floatVal)
		}
	}
	return 0
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
