package subscription

import (
	"context"
	"time"

	notificationContract "smart-wardrobe-be/internal/modules/notification/presentation/contract"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"

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
func (uc *SubscriptionUseCase) downgradeToFree(txCtx context.Context, lockedSub *entities.UserSubscription, freePlanID uuid.UUID, now time.Time) error {
	if lockedSub.FallbackPlanID != nil {
		subEvent, err := lockedSub.RestoreFallback("", now)
		if err != nil {
			return err
		}
		if err := uc.userSubRepo.Update(txCtx, lockedSub); err != nil {
			return err
		}
		if subEvent != nil {
			if err := uc.eventRepo.Create(txCtx, subEvent); err != nil {
				return err
			}
		}
		return nil
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

	if subEvent != nil {
		if err := uc.eventRepo.Create(txCtx, subEvent); err != nil {
			return err
		}
	}
	return nil
}
