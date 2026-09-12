package purchase

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	"smart-wardrobe-be/internal/modules/subscription/application/ordercode"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/deposittransactiontype"
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

	now := timeutils.GetNow(uc.cfg.Database.TimeZone)
	reuseUntil := now.Add(linkReuseMinTTL)
	if active, err := uc.depositTxRepo.GetActiveDirectPurchase(ctx, userID); err != nil {
		return nil, err
	} else if isReusableDirectPurchaseLink(active, reuseUntil) {
		return mapDepositTransactionToPaymentLinkDTO(active), nil
	}

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
	var (
		tx              = &entities.DepositTransaction{}
		reusable        *entities.DepositTransaction
		paidMeanwhile   *entities.DepositTransaction
		rotateOrderCode int64
		needRotate      bool
		staleCodes      []int64
	)
	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		sub, err := uc.userSubRepo.GetByUserIDWithLock(txCtx, userID)
		if err != nil {
			return err
		}
		if err := uc.validatePurchaseWithPendingCheck(txCtx, userID, plan, sub, now, false); err != nil {
			return err
		}
		active, err := uc.depositTxRepo.GetActiveDirectPurchase(txCtx, userID)
		if err != nil {
			return err
		}
		if active == nil {
			return nil
		}
		if isReusableDirectPurchaseLink(active, reuseUntil) {
			reusable = active
			return nil
		}
		if isExpiredDirectPurchaseLink(active, now) {
			active.Status = depositstatus.Expired
			active.ExpiredAt = &now
			if err := uc.depositTxRepo.Update(txCtx, active); err != nil {
				return err
			}
			staleCodes = append(staleCodes, active.OrderCode)
			return nil
		}
		if active.PaymentUrl == nil && active.PaymentLinkID == nil {
			return subscriptionerrors.ErrPendingPaymentExists()
		}
		rotateOrderCode, needRotate = active.OrderCode, true
		return nil
	}); err != nil {
		return nil, err
	}
	if reusable != nil {
		return mapDepositTransactionToPaymentLinkDTO(reusable), nil
	}
	if needRotate {
		if cerr := uc.paymentGateway.CancelPaymentLink(ctx, rotateOrderCode, cancelReasonSuperseded); cerr != nil {
			old, gerr := uc.depositTxRepo.GetByOrderCode(ctx, rotateOrderCode)
			if gerr != nil || old == nil {
				return nil, subscriptionerrors.ErrPendingPaymentExists()
			}
			return mapDepositTransactionToPaymentLinkDTO(old), nil
		}
	}
	for _, code := range staleCodes {
		_ = uc.paymentGateway.CancelPaymentLink(ctx, code, cancelReasonStale)
	}
	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if needRotate {
			locked, err := uc.depositTxRepo.GetByOrderCodeWithLock(txCtx, rotateOrderCode)
			if err != nil || locked == nil {
				return subscriptionerrors.ErrLockTransactionFailed()
			}
			if locked.Status == depositstatus.Success {
				paidMeanwhile = locked
				return nil
			}
			locked.Status = depositstatus.Expired
			locked.ExpiredAt = &now
			if err := uc.depositTxRepo.Update(txCtx, locked); err != nil {
				return err
			}
		}
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
			OrderCode:             ordercode.Generate(),
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
	if paidMeanwhile != nil {
		return mapDepositTransactionToPaymentLinkDTO(paidMeanwhile), nil
	}

	returnURL := req.ReturnUrl
	if returnURL == "" {
		returnURL = uc.cfg.PayOS.ReturnUrl
	}
	cancelURL := req.CancelUrl
	if cancelURL == "" {
		cancelURL = uc.cfg.PayOS.CancelUrl
	}
	result, gatewayErr := uc.requestDirectPurchaseCheckoutWithRetry(ctx, tx, returnURL, cancelURL, expiresAt)
	if result == nil && gatewayErr != nil {
		return nil, gatewayErr
	}

	if applyErr := uc.applyDirectPurchaseCheckoutOutcome(ctx, tx, result, gatewayErr, now); applyErr != nil {
		return nil, applyErr
	}
	if gatewayErr != nil && payment.IsDuplicateOrderCode(result) {
		return nil, subscriptionerrors.ErrPaymentOrderConflict()
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
