package usecase

import (
	"context"
	"encoding/json"
	"math"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/loyaltytransactiontype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"time"

	"github.com/google/uuid"
)

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
		take := int(math.Min(float64(lot.RemainingPoints), float64(remainingToRedeem)))
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
	var config map[string]any
	if err := json.Unmarshal(doc, &config); err != nil {
		return 0
	}
	if val, ok := config["validDurationDays"]; ok {
		if floatVal, ok := val.(float64); ok {
			return int(floatVal)
		}
	}
	return 0
}
