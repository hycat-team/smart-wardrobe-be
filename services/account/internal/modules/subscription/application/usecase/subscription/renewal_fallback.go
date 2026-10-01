package subscription

import (
	"context"
	"time"

	notificationContract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// executeRenewalDowngrade downgrades an expired subscription to the free plan for a specific reason.
func (uc *SubscriptionUseCase) executeRenewalDowngrade(
	txCtx context.Context,
	lockedSub *entities.UserSubscription,
	freePlanID uuid.UUID,
	now time.Time,
	result *RenewalExecutionResult,
	reason string,
) error {
	result.Reason = reason
	if err := uc.downgradeToFree(txCtx, lockedSub, freePlanID, now); err != nil {
		return err
	}

	uc.log.Info("Downgraded subscription during scheduled renewal",
		zap.String("user_id", lockedSub.UserID.String()),
		zap.String("plan_id", lockedSub.SubscriptionPlanID.String()),
		zap.String("result", string(renewalStatusDowngraded)),
		zap.String("reason", reason),
	)

	planName := ""
	if lockedSub.SubscriptionPlan != nil {
		planName = lockedSub.SubscriptionPlan.Name
	}
	_ = uc.notifContract.SendByUserID(txCtx, notificationContract.ChannelEmail, notificationContract.TemplateRenewalFailed, map[string]any{
		"PlanName": planName,
		"Reason":   reason,
	}, lockedSub.UserID)

	result.Status = renewalStatusDowngraded
	return nil
}

// downgradeToFree updates the subscription record to the default free plan.
//
// 024 T051: both branches now publish. Expiry is the single most important
// transition in the whole feature — it is what removes a customer's paid
// entitlements — and it was writing a history row with no event, so the AI
// service kept honouring a plan that had already ended.
func (uc *SubscriptionUseCase) downgradeToFree(txCtx context.Context, lockedSub *entities.UserSubscription, freePlanID uuid.UUID, now time.Time) error {
	if lockedSub.FallbackPlanID != nil {
		// The lifetime fallback plan is the one the account is returning to, so
		// the payload has to name THAT plan, not the free plan.
		fallbackPlan, err := uc.planRepo.GetByID(txCtx, *lockedSub.FallbackPlanID)
		if err != nil {
			return err
		}
		if fallbackPlan == nil {
			return subscriptionerrors.ErrSubscriptionPlanNotFound()
		}
		subEvent, err := lockedSub.RestoreFallback("", now)
		if err != nil {
			return err
		}
		if err := uc.userSubRepo.Update(txCtx, lockedSub); err != nil {
			return err
		}
		_, err = uc.planChangePublisher.PublishPlanChange(txCtx, subEvent, lockedSub, fallbackPlan)
		return err
	}

	freePlan, err := uc.planRepo.GetByID(txCtx, freePlanID)
	if err != nil || freePlan == nil {
		return subscriptionerrors.ErrDefaultPlanLoadFailed()
	}

	subEvent, err := lockedSub.DowngradeToFree(freePlan, "", now)
	if err != nil {
		return err
	}

	if err := uc.userSubRepo.Update(txCtx, lockedSub); err != nil {
		return err
	}

	_, err = uc.planChangePublisher.PublishPlanChange(txCtx, subEvent, lockedSub, freePlan)
	return err
}
