package subscription

import (
	"context"
	"math"
	"time"

	notificationContract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	sharedmoney "smart-wardrobe-be/shared/domain/money"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// upcomingExpiryBatchSize matches the expiry scan's page size: a reminder run
// touches every finite plan ending inside the window, which can be the whole
// customer base right after a launch, and loading that in one query is how the
// job turns into a memory spike at midnight.
const upcomingExpiryBatchSize = 100

// notifyUpcomingExpiryBatch sends the warnings for one page of expiring plans.
//
// The wallet and fallback-plan lookups are batched per page rather than per
// customer: the previous loop issued one wallet query per row, so a window with
// 20k customers meant 20k round trips in a single nightly run.
//
// Failures to load the batch's wallets or plans are logged and the page is
// skipped. A reminder is best-effort - it must never abort the renewal scan that
// follows it, and the next midnight retries.
func (uc *SubscriptionUseCase) notifyUpcomingExpiryBatch(
	ctx context.Context,
	batch []*entities.UserSubscription,
	now time.Time,
	defaultFreePlanName *string,
) {
	renewUserIDs := make([]uuid.UUID, 0, len(batch))
	fallbackPlanIDs := make([]uuid.UUID, 0, len(batch))
	for _, sub := range batch {
		if sub == nil || sub.SubscriptionPlan == nil || sub.ExpiresAt == nil {
			continue
		}
		if sub.IsAutoRenewEnabled {
			renewUserIDs = append(renewUserIDs, sub.UserID)
		}
		if sub.FallbackPlanID != nil {
			fallbackPlanIDs = append(fallbackPlanIDs, *sub.FallbackPlanID)
		}
	}

	walletByUserID := map[uuid.UUID]*entities.UserWallet{}
	if len(renewUserIDs) > 0 {
		wallets, err := uc.walletRepo.GetByUserIDs(ctx, renewUserIDs)
		if err != nil {
			uc.log.Warn("Expiry reminder skipped a batch: wallets could not be loaded",
				zap.Int("users", len(renewUserIDs)), zap.Error(err))
			return
		}
		for _, wallet := range wallets {
			walletByUserID[wallet.UserID] = wallet
		}
	}

	fallbackNameByPlanID := map[uuid.UUID]string{}
	if len(fallbackPlanIDs) > 0 {
		plans, err := uc.planRepo.GetByIDs(ctx, fallbackPlanIDs)
		if err != nil {
			uc.log.Warn("Expiry reminder could not load lifetime fallback plans",
				zap.Int("plans", len(fallbackPlanIDs)), zap.Error(err))
		}
		for _, plan := range plans {
			fallbackNameByPlanID[plan.ID] = plan.Name
		}
	}

	for _, sub := range batch {
		if sub == nil || sub.SubscriptionPlan == nil || sub.ExpiresAt == nil {
			continue
		}

		// Round UP: a plan ending in 2 days and 3 hours still has three days of
		// service in front of the customer, and truncating it to "2 ngày" made
		// the warning look later than the account's actual deadline.
		daysRemaining := int(math.Ceil(sub.ExpiresAt.Sub(now).Hours() / 24))
		if daysRemaining < 0 {
			daysRemaining = 0
		}

		expiryNotice := map[string]any{
			"PlanName":         sub.SubscriptionPlan.Name,
			"ExpiresAt":        timeutils.FormatDateVN(*sub.ExpiresAt),
			"DaysRemaining":    daysRemaining,
			"RestoredPlanName": uc.restoredPlanName(ctx, sub, fallbackNameByPlanID, defaultFreePlanName),
		}

		if !sub.IsAutoRenewEnabled {
			_ = uc.notifContract.SendByUserID(ctx, notificationContract.ChannelEmail, notificationContract.TemplateSubscriptionExpiring, expiryNotice, sub.UserID)
			continue
		}

		wallet := walletByUserID[sub.UserID]
		if wallet == nil || wallet.Balance.GreaterThanOrEqual(sub.SubscriptionPlan.Price) {
			continue
		}

		expiryNotice["Balance"] = sharedmoney.FormatVND(wallet.Balance)
		expiryNotice["RequiredAmount"] = sharedmoney.FormatVND(sub.SubscriptionPlan.Price)
		_ = uc.notifContract.SendByUserID(ctx, notificationContract.ChannelEmail, notificationContract.TemplateWalletWarning, expiryNotice, sub.UserID)
	}
}

// restoredPlanName reports the plan an account falls back to when its finite
// plan ends: its lifetime plan when one sits underneath, otherwise the free
// plan. Saying "gói Free" unconditionally is a false statement about what a
// lifetime customer still owns.
//
// defaultFreePlanName is a run-scoped cache, loaded at most once per run.
func (uc *SubscriptionUseCase) restoredPlanName(
	ctx context.Context,
	sub *entities.UserSubscription,
	fallbackNameByPlanID map[uuid.UUID]string,
	defaultFreePlanName *string,
) string {
	if sub.FallbackPlanID != nil {
		if name := fallbackNameByPlanID[*sub.FallbackPlanID]; name != "" {
			return name
		}
	}
	if *defaultFreePlanName == "" {
		if freePlan, err := uc.planRepo.GetDefaultPlan(ctx); err == nil && freePlan != nil {
			*defaultFreePlanName = freePlan.Name
		}
	}
	return *defaultFreePlanName
}
