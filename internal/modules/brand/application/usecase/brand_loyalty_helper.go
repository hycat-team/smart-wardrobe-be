package usecase

import (
	"context"
	"math"
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	"smart-wardrobe-be/internal/modules/brand/application/mapper"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerjoinedsource"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltyroundingmode"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltytransactiontype"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/userstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/utils/stringutils"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (uc *BrandLoyaltyUseCase) expireDueLotsForAccount(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time, createdByUserID *uuid.UUID) (int, error) {
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

func (uc *BrandLoyaltyUseCase) listLoyaltyLots(ctx context.Context, loyaltyAccountID uuid.UUID, query dto.ListLoyaltyPointLotsQueryReq) ([]*dto.LoyaltyPointLotRes, error) {
	var status *loyaltypointlotstatus.LoyaltyPointLotStatus
	if query.Status != nil && strings.TrimSpace(*query.Status) != "" {
		value := loyaltypointlotstatus.LoyaltyPointLotStatus(strings.ToLower(strings.TrimSpace(*query.Status)))
		status = &value
	}
	lots, err := uc.lotRepo.ListByAccountID(ctx, loyaltyAccountID, status, query.ExpiresAt, query.Page, query.Limit)
	if err != nil {
		return nil, err
	}
	return mapper.MapLoyaltyPointLots(lots), nil
}

func (uc *BrandLoyaltyUseCase) ensureLoyaltyAccount(ctx context.Context, customer *entities.BrandCustomer) error {
	if customer == nil {
		return nil
	}
	existing, err := uc.accountRepo.GetByBrandCustomerID(ctx, customer.ID)
	if err != nil || existing != nil {
		return err
	}
	account := &entities.LoyaltyAccount{
		BrandID:         customer.BrandID,
		BrandCustomerID: customer.ID,
		UserID:          customer.UserID,
		CurrentPoints:   0,
		LifetimePoints:  0,
		TotalSpend:      0,
	}
	return uc.accountRepo.Create(ctx, account)
}

func (uc *BrandLoyaltyUseCase) resolveBrandCustomerForPoints(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.GrantLoyaltyPointsReq) (*entities.BrandCustomer, error) {
	if input.UserID != nil {
		user, err := uc.userRepo.GetByID(ctx, *input.UserID)
		if err != nil {
			return nil, err
		}
		if user == nil || user.IsDeleted || user.Status != userstatus.Active {
			return nil, branderrors.ErrUserNotFoundOrInactive()
		}
		existing, err := uc.customerRepo.GetByBrandAndUser(ctx, brandID, *input.UserID)
		if err != nil || existing != nil {
			return existing, err
		}
		customer := &entities.BrandCustomer{
			BrandID:      brandID,
			UserID:       input.UserID,
			CustomerName: input.CustomerName,
			JoinedSource: brandcustomerjoinedsource.SelfJoin,
			Status:       brandcustomerstatus.Active,
			JoinedAt:     time.Now().UTC(),
		}
		if err := uc.customerRepo.Create(ctx, customer); err != nil {
			return nil, err
		}
		return customer, nil
	}

	if input.Phone != nil && strings.TrimSpace(*input.Phone) != "" {
		phone := strings.TrimSpace(*input.Phone)
		phoneHash := stringutils.HashSHA256(phone)
		existing, err := uc.customerRepo.GetByBrandAndPhoneHash(ctx, brandID, phoneHash)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			if existing.CustomerName == nil && input.CustomerName != nil {
				existing.CustomerName = input.CustomerName
				if err := uc.customerRepo.Update(ctx, existing); err != nil {
					return nil, err
				}
			}
			return existing, nil
		}
		member, err := uc.memberRepo.GetByBrandAndUser(ctx, brandID, staffUserID)
		if err != nil {
			return nil, err
		}
		if member == nil {
			return nil, branderrors.ErrBrandPortalForbidden()
		}
		customer := &entities.BrandCustomer{
			BrandID:              brandID,
			CustomerName:         input.CustomerName,
			PhoneE164:            &phone,
			PhoneHash:            &phoneHash,
			ExternalCustomerCode: input.ExternalCustomerCode,
			JoinedSource:         brandcustomerjoinedsource.OfflinePurchase,
			Status:               brandcustomerstatus.Active,
			JoinedAt:             time.Now().UTC(),
			CreatedByMemberID:    &member.ID,
		}
		if err := uc.customerRepo.Create(ctx, customer); err != nil {
			return nil, err
		}
		return customer, nil
	}

	externalCode := strings.TrimSpace(stringutils.GetString(input.ExternalCustomerCode))
	existing, err := uc.customerRepo.GetByBrandAndExternalCode(ctx, brandID, externalCode)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	member, err := uc.memberRepo.GetByBrandAndUser(ctx, brandID, staffUserID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, branderrors.ErrBrandPortalForbidden()
	}
	customer := &entities.BrandCustomer{
		BrandID:              brandID,
		CustomerName:         input.CustomerName,
		ExternalCustomerCode: input.ExternalCustomerCode,
		JoinedSource:         brandcustomerjoinedsource.OfflinePurchase,
		Status:               brandcustomerstatus.Active,
		JoinedAt:             time.Now().UTC(),
		CreatedByMemberID:    &member.ID,
	}
	if err := uc.customerRepo.Create(ctx, customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (uc *BrandLoyaltyUseCase) resolvePointsDelta(input dto.GrantLoyaltyPointsReq, program *entities.LoyaltyProgram) (int, error) {
	if input.PointsDelta != nil {
		return *input.PointsDelta, nil
	}
	if input.PurchaseAmount == nil || input.TransactionType != loyaltytransactiontype.Earn {
		return 0, branderrors.ErrPurchaseAmountOrPointsRequired()
	}
	if program == nil || program.AmountPerPoint <= 0 {
		return 0, branderrors.ErrActiveLoyaltyProgramRequired()
	}
	points := *input.PurchaseAmount / program.AmountPerPoint
	switch program.RoundingMode {
	case loyaltyroundingmode.Ceil:
		return int(math.Ceil(points)), nil
	case loyaltyroundingmode.Round:
		return int(math.Round(points)), nil
	default:
		return int(math.Floor(points)), nil
	}
}

func validateGrantLoyaltyPointsInput(input dto.GrantLoyaltyPointsReq) error {
	hasUserID := input.UserID != nil && *input.UserID != uuid.Nil
	hasPhone := input.Phone != nil && strings.TrimSpace(*input.Phone) != ""
	hasExternalCode := input.ExternalCustomerCode != nil && strings.TrimSpace(*input.ExternalCustomerCode) != ""
	if !hasUserID && !hasPhone && !hasExternalCode {
		return branderrors.ErrCustomerIdentifierRequired()
	}
	if input.PurchaseAmount == nil && input.PointsDelta == nil {
		return branderrors.ErrPurchaseAmountOrPointsRequired()
	}
	if input.PurchaseAmount != nil && *input.PurchaseAmount < 0 {
		return branderrors.ErrPurchaseAmountOrPointsRequired()
	}
	if input.PointsDelta != nil && *input.PointsDelta == 0 {
		return branderrors.ErrPointsDeltaZero()
	}
	switch input.TransactionType {
	case loyaltytransactiontype.Earn:
		return nil
	default:
		return branderrors.ErrInvalidLoyaltyTransactionType()
	}
}

func calculateTransactionExpiry(transactionType loyaltytransactiontype.LoyaltyTransactionType, program *entities.LoyaltyProgram) *time.Time {
	if transactionType != loyaltytransactiontype.Earn || program == nil || program.PointExpiryDays == nil {
		return nil
	}
	expiresAt := time.Now().UTC().AddDate(0, 0, *program.PointExpiryDays)
	return &expiresAt
}
