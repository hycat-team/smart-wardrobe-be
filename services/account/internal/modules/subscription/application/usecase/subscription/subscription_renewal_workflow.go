package subscription

import (
	"context"
	"fmt"
	"time"

	notificationContract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/subscriptionrenewalstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	sharedmoney "smart-wardrobe-be/shared/domain/money"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	renewalStatusRenewed    = subscriptionrenewalstatus.Renewed
	renewalStatusDowngraded = subscriptionrenewalstatus.Downgraded
	renewalStatusSkipped    = subscriptionrenewalstatus.Skipped
	renewalStatusFailed     = subscriptionrenewalstatus.Failed
)

const (
	renewalReasonRenewed                  = "renewed"
	renewalReasonDowngradedAutoRenewOff   = "downgraded_auto_renew_disabled"
	renewalReasonDowngradedInsufficient   = "downgraded_insufficient_balance"
	renewalReasonFailedMissingPlan        = "failed_missing_plan"
	renewalReasonFailedInvalidPrice       = "failed_invalid_price"
	renewalReasonFailedInvalidDuration    = "failed_invalid_duration"
	renewalReasonFailedInactivePlan       = "failed_inactive_plan"
	renewalReasonFailedWalletMissing      = "failed_wallet_missing"
	renewalReasonFailedLockSubscription   = "failed_lock_subscription"
	renewalReasonFailedLockWallet         = "failed_lock_wallet"
	renewalReasonFailedProcessing         = "failed_processing"
	renewalReasonSkippedNilExpiresAt      = "skipped_nil_expires_at"
	renewalReasonSkippedNotFoundAfterLock = "skipped_not_found_after_lock"
)

type RenewalExecutionResult struct {
	Status            subscriptionrenewalstatus.SubscriptionRenewalStatus
	Reason            string
	UserID            uuid.UUID
	ExpiresAt         *time.Time
	RenewalAttemptKey string
}

type RenewalBatchSummary struct {
	RenewedCount    int
	DowngradedCount int
	SkippedCount    int
	FailedCount     int
	OutcomeCounts   map[string]int
}

// ProcessScheduledRenewals processes expired subscriptions in batches and applies renew-or-downgrade decisions.
func (uc *SubscriptionUseCase) ProcessScheduledRenewals(ctx context.Context, run *workerlog.Run) error {
	now := timeutils.GetNow(uc.cfg.Database.TimeZone)

	freePlanID, err := uc.planUseCase.GetDefaultSubscriptionPlanID(ctx)
	if err != nil {
		return subscriptionerrors.ErrDefaultPlanLoadFailed()
	}

	var lastUserID uuid.UUID
	var lastExpiresAt time.Time
	limit := 100
	summary := NewRenewalBatchSummary()

	for {
		expiredSubs, err := uc.userSubRepo.GetActiveExpiredSubscriptionsBatch(ctx, now, lastUserID, lastExpiresAt, limit)
		if err != nil {
			return subscriptionerrors.ErrQueryExpiredSubscriptionsFailed()
		}
		if len(expiredSubs) == 0 {
			break
		}

		for _, sub := range expiredSubs {
			run.AddTotal(1)
			if sub.ExpiresAt == nil {
				summary.Record(RenewalExecutionResult{
					Status: renewalStatusSkipped,
					Reason: renewalReasonSkippedNilExpiresAt,
					UserID: sub.UserID,
				}, nil)
				run.AddSkipped(1)
				continue
			}

			lastUserID = sub.UserID
			lastExpiresAt = *sub.ExpiresAt

			result, err := uc.processSingleScheduledRenewal(ctx, sub, freePlanID, now)
			summary.Record(result, err)
			if err != nil {
				run.ChildError(uc.log, "Failed to process subscription renewal",
					zap.String("userId", sub.UserID.String()),
					zap.Time("expiresAt", *sub.ExpiresAt),
					zap.String("reason", result.Reason),
					zap.String("result", string(renewalStatusFailed)),
					zap.Error(err),
				)
				continue
			}

			switch result.Status {
			case renewalStatusRenewed:
				run.AddSuccess(1)
			case renewalStatusDowngraded:
				run.AddSuccess(1)
			case renewalStatusSkipped:
				run.AddSkipped(1)
			}
		}

		if len(expiredSubs) < limit {
			break
		}
	}

	run.AddSummaryFields(
		zap.Int("renewedCount", summary.RenewedCount),
		zap.Int("downgradedCount", summary.DowngradedCount),
		zap.Int("failedCount", summary.FailedCount),
	)
	if summary.FailedCount > 0 {
		return fmt.Errorf("scheduled renewal job completed with %d failed records", summary.FailedCount)
	}

	return nil
}

// ProcessUpcomingExpiryNotifications sends wallet balance warnings for subscriptions expiring within N days.
func (uc *SubscriptionUseCase) ProcessUpcomingExpiryNotifications(ctx context.Context) error {
	warningDays := uc.cfg.Subscription.WalletBalanceWarningDays
	if warningDays <= 0 {
		warningDays = 3
	}
	now := timeutils.GetNow(uc.cfg.Database.TimeZone)
	deadline := now.AddDate(0, 0, warningDays)

	expiringSubs, err := uc.userSubRepo.GetSubscriptionsExpiringWithin(ctx, now, deadline)
	if err != nil {
		return subscriptionerrors.ErrQueryExpiredSubscriptionsFailed()
	}

	for _, sub := range expiringSubs {
		if sub == nil || !sub.IsAutoRenewEnabled || sub.SubscriptionPlan == nil {
			continue
		}

		wallet, err := uc.walletRepo.GetByUserID(ctx, sub.UserID)
		if err != nil || wallet == nil {
			continue
		}

		if wallet.Balance.GreaterThanOrEqual(sub.SubscriptionPlan.Price) {
			continue
		}

		daysRemaining := int(sub.ExpiresAt.Sub(now).Hours() / 24)
		if daysRemaining < 0 {
			daysRemaining = 0
		}

		_ = uc.notifContract.SendByUserID(ctx, notificationContract.ChannelEmail, notificationContract.TemplateWalletWarning, map[string]any{
			"DaysRemaining":  daysRemaining,
			"Balance":        sharedmoney.ToFloat(wallet.Balance),
			"RequiredAmount": sharedmoney.ToFloat(sub.SubscriptionPlan.Price),
		}, sub.UserID)
	}
	return nil
}

// NewRenewalBatchSummary creates the accumulator used for scheduled renewal reporting.
func NewRenewalBatchSummary() *RenewalBatchSummary {
	return &RenewalBatchSummary{
		OutcomeCounts: map[string]int{},
	}
}

// Record merges a single renewal execution result into the batch summary.
func (s *RenewalBatchSummary) Record(result RenewalExecutionResult, err error) {
	if err != nil {
		s.FailedCount++
		s.OutcomeCounts[result.Reason]++
		return
	}

	s.OutcomeCounts[result.Reason]++
	switch result.Status {
	case renewalStatusRenewed:
		s.RenewedCount++
	case renewalStatusDowngraded:
		s.DowngradedCount++
	case renewalStatusSkipped:
		s.SkippedCount++
	}
}

// processSingleScheduledRenewal handles one expired subscription inside its own transaction boundary.
func (uc *SubscriptionUseCase) processSingleScheduledRenewal(
	ctx context.Context,
	sub *entities.UserSubscription,
	freePlanID uuid.UUID,
	now time.Time,
) (RenewalExecutionResult, error) {
	result := RenewalExecutionResult{
		Status:    renewalStatusSkipped,
		Reason:    renewalReasonFailedProcessing,
		UserID:    sub.UserID,
		ExpiresAt: sub.ExpiresAt,
	}

	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		return uc.executeScheduledRenewal(txCtx, sub.UserID, freePlanID, now, &result)
	})
	if err == nil && (result.Status == renewalStatusRenewed || result.Status == renewalStatusDowngraded) {
		// The overview changed (extension or downgrade) - drop the cached entry
		// after the transaction commits (spec 019 FR-008).
		//
		// 024 T154: the nil-cache guard moved into the helper. This was a fourth
		// copy of the post-commit defect T151 fixed on the grant path - `_ = ...` on
		// the caller's context, so a cron run whose context had lapsed left the
		// overview serving the pre-renewal expiry until its TTL. A background job is
		// exactly the case where the caller's context is least likely to be alive.
		uc.invalidateOverviewCache(ctx, sub.UserID)
	}
	return result, err
}

// executeScheduledRenewal performs the locked subscription renewal workflow for one user.
func (uc *SubscriptionUseCase) executeScheduledRenewal(
	txCtx context.Context,
	userID uuid.UUID,
	freePlanID uuid.UUID,
	now time.Time,
	result *RenewalExecutionResult,
) error {
	result.Reason = renewalReasonFailedLockSubscription

	lockedSub, err := uc.userSubRepo.GetActiveExpiredSubscriptionByUserIDWithLock(txCtx, userID, now)
	if err != nil {
		return err
	}
	if lockedSub == nil {
		result.Status = renewalStatusSkipped
		result.Reason = renewalReasonSkippedNotFoundAfterLock
		return nil
	}

	plan, days, validationReason, err := uc.validateRenewalPlan(lockedSub)
	if err != nil {
		result.Status = renewalStatusFailed
		result.Reason = validationReason
		return err
	}

	if !lockedSub.IsAutoRenewEnabled {
		return uc.executeRenewalDowngrade(txCtx, lockedSub, freePlanID, now, result, renewalReasonDowngradedAutoRenewOff)
	}

	result.Reason = renewalReasonFailedLockWallet
	wallet, err := uc.walletRepo.GetByUserIDWithLock(txCtx, lockedSub.UserID)
	if err != nil {
		return err
	}
	if wallet == nil {
		result.Status = renewalStatusFailed
		result.Reason = renewalReasonFailedWalletMissing
		return fmt.Errorf("missing wallet for renewal")
	}
	if err := sharedmoney.ValidateSupportedCurrency(wallet.Currency); err != nil {
		result.Status = renewalStatusFailed
		result.Reason = renewalReasonFailedProcessing
		return err
	}

	if wallet.Balance.LessThan(plan.Price) {
		return uc.executeRenewalDowngrade(txCtx, lockedSub, freePlanID, now, result, renewalReasonDowngradedInsufficient)
	}
	key := fmt.Sprintf("SUB_RENEWAL:%s:%s:%s", lockedSub.UserID, plan.ID, lockedSub.ExpiresAt.UTC().Truncate(time.Second).Format(time.RFC3339))
	existing, err := uc.renewalAttemptRepo.GetByKey(txCtx, key)
	if err != nil {
		return err
	}
	if existing != nil && existing.Status == subscriptionrenewalstatus.Succeeded {
		result.Status = renewalStatusRenewed
		result.Reason = renewalReasonRenewed
		return nil
	}
	if existing == nil {
		existing = &entities.SubscriptionRenewalAttempt{RenewalAttemptKey: key, UserID: lockedSub.UserID, ExpectedPlanID: plan.ID, ExpectedExpiresAt: *lockedSub.ExpiresAt, ExpectedSubscriptionVersion: lockedSub.Version, Status: subscriptionrenewalstatus.Processing, AttemptCount: 1}
		if err := uc.renewalAttemptRepo.Create(txCtx, existing); err != nil {
			return err
		}
	} else {
		existing.Status = subscriptionrenewalstatus.Processing
		existing.AttemptCount++
		if err := uc.renewalAttemptRepo.Update(txCtx, existing); err != nil {
			return err
		}
	}
	result.RenewalAttemptKey = key

	return uc.executePaidRenewal(txCtx, lockedSub, wallet, plan, days, now, result)
}
