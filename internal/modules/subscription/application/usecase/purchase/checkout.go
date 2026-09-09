package purchase

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/payosdescription"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/internal/pkg/utils/errorutils"
	"smart-wardrobe-be/internal/pkg/utils/timeutils"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/currency"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"

	"github.com/google/uuid"
)

func (uc *SubscriptionPurchaseUseCase) CreateDirectPurchase(ctx context.Context, userID uuid.UUID, req *dto.DirectPurchaseReq) (*dto.PaymentLinkDTO, error) {
	plan, err := uc.planRepo.GetBySlug(ctx, req.PlanSlug)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, subscriptionerrors.ErrRequestedPlanNotFound()
	}

	if !plan.Price.GreaterThan(sharedmoney.Zero) {
		return nil, subscriptionerrors.ErrFreePlanDirectPurchase()
	}

	if active, err := uc.depositTxRepo.GetActiveDirectPurchase(ctx, userID); err != nil {
		return nil, err
	} else if active != nil {
		url := ""
		if active.PaymentUrl != nil {
			url = *active.PaymentUrl
		}
		return &dto.PaymentLinkDTO{
			PaymentUrl:           url,
			OrderCode:            active.OrderCode,
			PaymentStatus:        active.Status,
			ExpiresAt:            active.ExpiresAt,
			NextReconciliationAt: active.NextReconciliationAt,
		}, nil
	}

	now := timeutils.GetNow(uc.cfg.Database.TimeZone)
	snapshotBytes, err := json.Marshal(map[string]any{
		"maxWardrobeItems":   plan.MaxWardrobeItems,
		"maxOutfits":         plan.MaxOutfits,
		"aiOutfitDailyQuota": plan.AiOutfitDailyQuota,
		"aiChatDailyQuota":   plan.AiChatDailyQuota,
	})
	if err != nil {
		return nil, subscriptionerrors.ErrDirectPurchaseCreateFailed()
	}
	expiresAt := now.Add(time.Duration(uc.cfg.PayOS.ExpiredMinutes) * time.Minute)
	nextReconcile := now.Add(30 * time.Second)
	tx := &entities.DepositTransaction{}
	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		sub, err := uc.userSubRepo.GetByUserIDWithLock(txCtx, userID)
		if err != nil {
			return err
		}
		if err := uc.validatePurchase(txCtx, userID, plan, sub, now); err != nil {
			return err
		}
		planCode, planName, tierRank, kind, pricingVersion := plan.Slug, plan.Name, plan.TierRank, plan.PlanKind, plan.PricingVersion
		tx = &entities.DepositTransaction{
			UserID:                userID,
			Amount:                plan.Price,
			ExpectedAmount:        plan.Price,
			Currency:              currency.VND,
			Status:                depositstatus.Creating,
			TransactionType:       deposittransactiontype.DirectPurchase,
			SubscriptionPlanID:    &plan.ID,
			PlanCode:              &planCode,
			PlanName:              &planName,
			TierRank:              &tierRank,
			PlanKind:              &kind,
			PurchasedDurationDays: plan.DurationDays,
			BenefitSnapshot:       entities.JSONDocument(snapshotBytes),
			PricingVersion:        &pricingVersion,
			ExpiresAt:             &expiresAt,
			NextReconciliationAt:  &nextReconcile,
		}
		if err := uc.depositTxRepo.Create(txCtx, tx); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "ux_active_direct_purchase_per_user") || strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
				return subscriptionerrors.ErrPendingPaymentExists()
			}
			return subscriptionerrors.ErrDirectPurchaseCreateFailed()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	returnURL := req.ReturnUrl
	if returnURL == "" {
		returnURL = uc.cfg.PayOS.ReturnUrl
	}
	cancelURL := req.CancelUrl
	if cancelURL == "" {
		cancelURL = uc.cfg.PayOS.CancelUrl
	}
	description := payosdescription.FormatTransactionDescription(tx.OrderCode)
	result, gatewayErr := uc.paymentGateway.CreateCheckoutSession(ctx, &payment.CheckoutSessionReq{
		OrderCode:   tx.OrderCode,
		Amount:      tx.Amount,
		Description: description,
		ReturnUrl:   returnURL,
		CancelUrl:   cancelURL,
		ExpiresAt:   expiresAt,
	})

	applyErr := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		locked, err := uc.depositTxRepo.GetByOrderCodeWithLock(txCtx, tx.OrderCode)
		if err != nil || locked == nil {
			return subscriptionerrors.ErrLockTransactionFailed()
		}
		if locked.Status != depositstatus.Creating {
			return nil
		}
		if gatewayErr == nil && result != nil && result.Outcome == payment.OutcomeSucceeded {
			locked.Status = depositstatus.Pending
			locked.PaymentUrl = &result.CheckoutURL
			if result.PaymentLinkID != "" {
				locked.PaymentLinkID = &result.PaymentLinkID
			}
		} else if result != nil && result.Outcome == payment.OutcomeKnownFailure && !result.Retryable {
			locked.Status = depositstatus.CreationFailed
			locked.FailureReason = &result.ErrorCode
		} else {
			locked.Status = depositstatus.ReconciliationRequired
			locked.ReconciliationAttempts++
			if result != nil {
				locked.LastProviderErrorCode = &result.ErrorCode
			}
			t := now
			locked.LastProviderErrorAt = &t
		}
		return uc.depositTxRepo.Update(txCtx, locked)
	})
	if applyErr != nil {
		return nil, applyErr
	}
	if gatewayErr != nil && result != nil && result.Outcome == payment.OutcomeKnownFailure {
		return nil, errorutils.WrapError(gatewayErr, "Không thể khởi tạo liên kết thanh toán với cổng ngân hàng")
	}
	url := ""
	responseStatus := depositstatus.ReconciliationRequired
	if result != nil {
		url = result.CheckoutURL
		if result.Outcome == payment.OutcomeSucceeded {
			responseStatus = depositstatus.Pending
		} else if result.Outcome == payment.OutcomeKnownFailure && !result.Retryable {
			responseStatus = depositstatus.CreationFailed
		}
	}
	return &dto.PaymentLinkDTO{
		PaymentUrl:           url,
		OrderCode:            tx.OrderCode,
		PaymentStatus:        responseStatus,
		ExpiresAt:            &expiresAt,
		NextReconciliationAt: &nextReconcile,
	}, nil
}
