package subscription

import (
	"context"
	"time"

	notificationContract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// downgradeWalletNotice carries the already-formatted wallet figures for the
// insufficient-balance notice. nil means the downgrade happened for another
// reason (auto-renew switched off), where quoting a shortfall would invent one.
type downgradeWalletNotice struct {
	RequiredAmount string
	Balance        string
}

// executeRenewalDowngrade downgrades an expired subscription to the free plan for a specific reason.
func (uc *SubscriptionUseCase) executeRenewalDowngrade(
	txCtx context.Context,
	lockedSub *entities.UserSubscription,
	freePlanID uuid.UUID,
	now time.Time,
	result *RenewalExecutionResult,
	reason string,
	walletNotice *downgradeWalletNotice,
) error {
	result.Reason = reason

	// Capture what the customer is LOSING before downgradeToFree rewrites the
	// subscription in place. DowngradeToFree reassigns SubscriptionPlan to the
	// free plan, so reading the name afterwards reported the plan they were moved
	// TO - which is how a customer whose Premium had just ended was told
	// "Gói dịch vụ: Free Plan". The lifetime-fallback branch does not reassign
	// the field, so the two paths disagreed with each other as well.
	planName := ""
	if lockedSub.SubscriptionPlan != nil {
		planName = lockedSub.SubscriptionPlan.Name
	}
	expiredAt := ""
	if lockedSub.ExpiresAt != nil {
		expiredAt = timeutils.FormatDateVN(*lockedSub.ExpiresAt)
	}

	restoredPlanName, err := uc.downgradeToFree(txCtx, lockedSub, freePlanID, now)
	if err != nil {
		return err
	}

	// The log keeps the raw reason code: it is a technical field, and the
	// operator-facing line is where `downgraded_auto_renew_disabled` belongs.
	// The EMAIL must never show it - the template renders a Vietnamese sentence
	// chosen from NeedsTopup instead.
	uc.log.Info("Downgraded subscription during scheduled renewal",
		zap.String("user_id", lockedSub.UserID.String()),
		zap.String("plan_id", lockedSub.SubscriptionPlanID.String()),
		zap.String("result", string(renewalStatusDowngraded)),
		zap.String("reason", reason),
	)

	params := map[string]any{
		"PlanName":         planName,
		"ExpiredAt":        expiredAt,
		"RestoredPlanName": restoredPlanName,
		"NeedsTopup":       walletNotice != nil,
	}
	if walletNotice != nil {
		params["RequiredAmount"] = walletNotice.RequiredAmount
		params["Balance"] = walletNotice.Balance
	}

	_ = uc.notifContract.SendByUserID(txCtx, notificationContract.ChannelEmail, notificationContract.TemplateSubscriptionExpired, params, lockedSub.UserID)

	result.Status = renewalStatusDowngraded
	return nil
}

// downgradeToFree updates the subscription record to the plan the account falls
// back to, and returns THAT plan's name.
//
// The name is returned rather than looked up by the caller because the answer is
// only known here: an account holding a lifetime plan keeps it underneath a
// finite one, so expiry restores the lifetime plan instead of dropping to Free.
// The expiry email told every customer "trở về gói Free", which was false for
// exactly those accounts.
//
// 024 T051: both branches now publish. Expiry is the single most important
// transition in the whole feature — it is what removes a customer's paid
// entitlements — and it was writing a history row with no event, so the AI
// service kept honouring a plan that had already ended.
func (uc *SubscriptionUseCase) downgradeToFree(txCtx context.Context, lockedSub *entities.UserSubscription, freePlanID uuid.UUID, now time.Time) (string, error) {
	if lockedSub.FallbackPlanID != nil {
		// The lifetime fallback plan is the one the account is returning to, so
		// the payload has to name THAT plan, not the free plan.
		fallbackPlan, err := uc.planRepo.GetByID(txCtx, *lockedSub.FallbackPlanID)
		if err != nil {
			return "", err
		}
		if fallbackPlan == nil {
			return "", subscriptionerrors.ErrSubscriptionPlanNotFound()
		}
		subEvent, err := lockedSub.RestoreFallback("", now)
		if err != nil {
			return "", err
		}
		if err := uc.userSubRepo.Update(txCtx, lockedSub); err != nil {
			return "", err
		}
		if _, err := uc.planChangePublisher.PublishPlanChange(txCtx, subEvent, lockedSub, fallbackPlan); err != nil {
			return "", err
		}
		return fallbackPlan.Name, nil
	}

	freePlan, err := uc.planRepo.GetByID(txCtx, freePlanID)
	if err != nil || freePlan == nil {
		return "", subscriptionerrors.ErrDefaultPlanLoadFailed()
	}

	subEvent, err := lockedSub.DowngradeToFree(freePlan, "", now)
	if err != nil {
		return "", err
	}

	if err := uc.userSubRepo.Update(txCtx, lockedSub); err != nil {
		return "", err
	}

	if _, err := uc.planChangePublisher.PublishPlanChange(txCtx, subEvent, lockedSub, freePlan); err != nil {
		return "", err
	}
	return freePlan.Name, nil
}
