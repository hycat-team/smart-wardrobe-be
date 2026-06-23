package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/wallet"
	"smart-wardrobe-be/internal/shared/domain/constants/walletstatementtype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/pkg/utils/timeutils"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	renewalStatusRenewed    = "renewed"
	renewalStatusDowngraded = "downgraded"
	renewalStatusSkipped    = "skipped"
	renewalStatusFailed     = "failed"
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
	Status            string
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

	freePlanID, err := uc.planContract.GetDefaultSubscriptionPlanID(ctx)
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
					zap.String("result", renewalStatusFailed),
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
	if existing != nil && existing.Status == "Succeeded" {
		result.Status = renewalStatusRenewed
		result.Reason = renewalReasonRenewed
		return nil
	}
	if existing == nil {
		existing = &entities.SubscriptionRenewalAttempt{RenewalAttemptKey: key, UserID: lockedSub.UserID, ExpectedPlanID: plan.ID, ExpectedExpiresAt: *lockedSub.ExpiresAt, ExpectedSubscriptionVersion: lockedSub.Version, Status: "Processing", AttemptCount: 1}
		if err := uc.renewalAttemptRepo.Create(txCtx, existing); err != nil {
			return err
		}
	} else {
		existing.Status = "Processing"
		existing.AttemptCount++
		if err := uc.renewalAttemptRepo.Update(txCtx, existing); err != nil {
			return err
		}
	}
	result.RenewalAttemptKey = key

	return uc.executePaidRenewal(txCtx, lockedSub, wallet, plan, days, now, result)
}

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
		zap.String("result", renewalStatusDowngraded),
		zap.String("reason", reason),
	)
	result.Status = renewalStatusDowngraded
	return nil
}

// executePaidRenewal debits the wallet, extends expiry, and records the renewal ledger entry.
func (uc *SubscriptionUseCase) executePaidRenewal(
	txCtx context.Context,
	lockedSub *entities.UserSubscription,
	userWallet *entities.UserWallet,
	plan *entities.SubscriptionPlan,
	days int,
	now time.Time,
	result *RenewalExecutionResult,
) error {
	result.Reason = renewalReasonRenewed

	snapshotBytes, err := json.Marshal(map[string]any{
		"maxWardrobeItems":   plan.MaxWardrobeItems,
		"maxOutfits":         plan.MaxOutfits,
		"aiOutfitDailyQuota": plan.AiOutfitDailyQuota,
		"aiChatDailyQuota":   plan.AiChatDailyQuota,
	})
	if err != nil {
		return err
	}
	snapshot := entities.JSONDocument(snapshotBytes)

	// Call ProcessWalletTransaction for debit
	metadata := wallet.WalletStatementMetadata{
		SourcePlanCode:             &plan.Slug,
		SourceTierRank:             &plan.TierRank,
		ActiveTierRankAtCompletion: &plan.TierRank,
		RenewalAttemptKey:          &result.RenewalAttemptKey,
	}
	desc := fmt.Sprintf("Auto-renewed subscription plan %s", plan.Name)
	if err := wallet.ProcessWalletTransaction(
		txCtx,
		uc.walletRepo,
		uc.statementRepo,
		lockedSub.UserID,
		plan.Price.Neg(),
		walletstatementtype.SubscriptionRenewal,
		desc,
		nil,
		metadata,
		now,
	); err != nil {
		return err
	}

	// Mutate subscription via Extend domain method
	subEvent, err := lockedSub.Extend(plan, snapshot, "", now)
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

	key := result.RenewalAttemptKey
	if attempt, err := uc.renewalAttemptRepo.GetByKey(txCtx, key); err != nil {
		return err
	} else if attempt != nil {
		attempt.Status = "Succeeded"
		attempt.CompletedAt = &now
		if err := uc.renewalAttemptRepo.Update(txCtx, attempt); err != nil {
			return err
		}
	}

	uc.log.Info("Renewed subscription automatically successfully",
		zap.String("user_id", lockedSub.UserID.String()),
		zap.String("plan_id", lockedSub.SubscriptionPlanID.String()),
		zap.Float64("amount", sharedmoney.ToFloat(plan.Price)),
		zap.String("currency", string(userWallet.Currency)),
		zap.String("transaction_type", string(walletstatementtype.SubscriptionRenewal)),
		zap.String("result", renewalStatusRenewed),
		zap.String("reason", result.Reason),
	)
	result.Status = renewalStatusRenewed
	return nil
}

// validateRenewalPlan ensures that the target plan is valid for renewal.
func (uc *SubscriptionUseCase) validateRenewalPlan(lockedSub *entities.UserSubscription) (*entities.SubscriptionPlan, int, string, error) {
	if lockedSub.SubscriptionPlan == nil {
		return nil, 0, renewalReasonFailedMissingPlan, fmt.Errorf("subscription plan is missing")
	}

	plan := lockedSub.SubscriptionPlan
	if !plan.IsActive {
		return nil, 0, renewalReasonFailedInactivePlan, fmt.Errorf("subscription plan is inactive")
	}
	if !plan.Price.GreaterThan(sharedmoney.Zero) {
		return nil, 0, renewalReasonFailedInvalidPrice, fmt.Errorf("subscription plan price must be greater than zero")
	}

	days := 30
	if plan.DurationDays != nil {
		if *plan.DurationDays <= 0 {
			return nil, 0, renewalReasonFailedInvalidDuration, fmt.Errorf("subscription plan duration must be greater than zero")
		}
		days = *plan.DurationDays
	}

	return plan, days, "", nil
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
