package usecase

import (
	"context"
	"fmt"
	"time"

	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/shared/domain/constants/walletstatementtype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"
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

func (uc *SubscriptionUseCase) ProcessScheduledRenewals(ctx context.Context) error {
	// Get the current timestamp based on database timezone settings.
	now := timeutils.GetNow(uc.cfg.Database.TimeZone)

	// Fetch the configured default subscription plan ID (usually the Free Plan) for downgrades.
	freePlanID, err := uc.planContract.GetDefaultSubscriptionPlanID(ctx)
	if err != nil {
		return subscriptionerrors.ErrDefaultPlanLoadFailed
	}

	// Initialize pagination state for cursor-based batch processing to avoid memory bloat.
	var lastUserID uuid.UUID
	var lastExpiresAt time.Time
	limit := 100

	// Track statistics for reporting and logs.
	var renewedCount, downgradedCount, skippedCount, failedCount int
	outcomeCounts := map[string]int{}

	// Start batch processing loop.
	for {
		// Query a batch of expired, active subscriptions using a cursor (lastUserID & lastExpiresAt).
		expiredSubs, err := uc.userSubRepo.GetActiveExpiredSubscriptionsBatch(ctx, now, lastUserID, lastExpiresAt, limit)
		if err != nil {
			return subscriptionerrors.ErrQueryExpiredSubscriptionsFailed
		}

		// If no expired subscriptions are returned, processing is complete.
		if len(expiredSubs) == 0 {
			break
		}

		// Process each expired subscription in the current batch.
		for _, sub := range expiredSubs {
			// If ExpiresAt is nil, this is an unlimited subscription and should not expire.
			if sub.ExpiresAt == nil {
				skippedCount++
				outcomeCounts[renewalReasonSkippedNilExpiresAt]++
				continue
			}

			// Capture cursor values for the next query pagination block.
			cursorUserID := sub.UserID
			cursorExpiresAt := *sub.ExpiresAt

			lastUserID = cursorUserID
			lastExpiresAt = cursorExpiresAt

			resultStatus := renewalStatusSkipped
			resultReason := renewalReasonFailedProcessing

			// Define individual renewal transaction scope.
			processSubFn := func(txCtx context.Context) error {
				resultReason = renewalReasonFailedLockSubscription

				// Re-fetch the subscription with a row lock (FOR UPDATE) to prevent concurrent modifications.
				lockedSub, err := uc.userSubRepo.GetActiveExpiredSubscriptionByUserIDWithLock(txCtx, sub.UserID, now)
				if err != nil {
					return err
				}
				// If the subscription is no longer expired or was modified/removed by another process, skip it.
				if lockedSub == nil {
					resultStatus = renewalStatusSkipped
					resultReason = renewalReasonSkippedNotFoundAfterLock
					return nil
				}

				// Validate plan configurations (active status, pricing, duration).
				plan, days, validationReason, err := uc.validateRenewalPlan(lockedSub)
				if err != nil {
					resultStatus = renewalStatusFailed
					resultReason = validationReason
					return err
				}

				// If auto-renewal is turned off, downgrade the user immediately to the Free plan.
				if !lockedSub.IsAutoRenewEnabled {
					resultReason = renewalReasonDowngradedAutoRenewOff
					if err := uc.downgradeToFree(txCtx, lockedSub, freePlanID, now); err != nil {
						return err
					}
					uc.log.Info("Downgraded subscription because auto-renew is disabled",
						zap.String("user_id", lockedSub.UserID.String()),
						zap.String("plan_id", lockedSub.SubscriptionPlanID.String()),
						zap.String("result", renewalStatusDowngraded),
						zap.String("reason", resultReason),
					)
					resultStatus = renewalStatusDowngraded
					return nil
				}

				resultReason = renewalReasonFailedLockWallet

				// Fetch and lock the user's wallet (FOR UPDATE) to perform debit checks.
				wallet, err := uc.walletRepo.GetByUserIDWithLock(txCtx, lockedSub.UserID)
				if err != nil {
					return err
				}
				if wallet == nil {
					resultStatus = renewalStatusFailed
					resultReason = renewalReasonFailedWalletMissing
					return fmt.Errorf("missing wallet for renewal")
				}
				if err := sharedmoney.ValidateSupportedCurrency(wallet.Currency); err != nil {
					resultStatus = renewalStatusFailed
					resultReason = renewalReasonFailedProcessing
					return err
				}

				// If the user has insufficient funds, automatically downgrade them to the Free plan.
				if wallet.Balance.LessThan(plan.Price) {
					resultReason = renewalReasonDowngradedInsufficient
					if err := uc.downgradeToFree(txCtx, lockedSub, freePlanID, now); err != nil {
						return err
					}
					uc.log.Info("Downgraded subscription because wallet balance is insufficient",
						zap.String("user_id", lockedSub.UserID.String()),
						zap.String("plan_id", lockedSub.SubscriptionPlanID.String()),
						zap.String("currency", string(wallet.Currency)),
						zap.Float64("amount", sharedmoney.ToFloat(plan.Price)),
						zap.String("result", renewalStatusDowngraded),
						zap.String("reason", resultReason),
					)
					resultStatus = renewalStatusDowngraded
					return nil
				}

				// Deduct subscription price from the locked wallet.
				resultReason = renewalReasonRenewed
				prevBalance := wallet.Balance
				wallet.Balance = wallet.Balance.Sub(plan.Price)
				wallet.UpdatedAt = now

				if err := uc.walletRepo.Update(txCtx, wallet); err != nil {
					return err
				}

				// Extend the subscription expiration date.
				newExpiry := now.AddDate(0, 0, days)
				lockedSub.ExpiresAt = &newExpiry
				lockedSub.IsActive = true
				lockedSub.UpdatedAt = now

				if err := uc.userSubRepo.Update(txCtx, lockedSub); err != nil {
					return err
				}

				// Create a transaction statement ledger entry for the renewal payment.
				statement := &entities.WalletStatement{
					UserID:          lockedSub.UserID,
					Amount:          plan.Price.Neg(),
					TransactionType: walletstatementtype.SubscriptionRenewal,
					PreviousBalance: prevBalance,
					NewBalance:      wallet.Balance,
					Description:     fmt.Sprintf("Auto-renewed subscription plan %s", plan.Name),
				}

				if err := uc.statementRepo.Create(txCtx, statement); err != nil {
					return err
				}

				uc.log.Info("Renewed subscription automatically successfully",
					zap.String("user_id", lockedSub.UserID.String()),
					zap.String("plan_id", lockedSub.SubscriptionPlanID.String()),
					zap.Float64("amount", sharedmoney.ToFloat(plan.Price)),
					zap.String("currency", string(wallet.Currency)),
					zap.String("transaction_type", string(walletstatementtype.SubscriptionRenewal)),
					zap.String("result", renewalStatusRenewed),
					zap.String("reason", resultReason),
				)
				resultStatus = renewalStatusRenewed
				return nil
			}

			// Execute the transaction function.
			if err = uc.uow.Execute(ctx, processSubFn); err != nil {
				failedCount++
				outcomeCounts[resultReason]++
				uc.log.Error("Failed to process subscription renewal",
					zap.String("user_id", sub.UserID.String()),
					zap.Time("expires_at", *sub.ExpiresAt),
					zap.String("reason", resultReason),
					zap.String("result", renewalStatusFailed),
					zap.Error(err),
				)
			} else {
				outcomeCounts[resultReason]++
				switch resultStatus {
				case renewalStatusRenewed:
					renewedCount++
				case renewalStatusDowngraded:
					downgradedCount++
				case renewalStatusSkipped:
					skippedCount++
				}
			}
		}

		// Break early if the returned batch size is less than limit, indicating we reached the end of the query.
		if len(expiredSubs) < limit {
			break
		}
	}

	// Log batch execution summaries.
	uc.log.Info("Processed scheduled renewals summary",
		zap.Int("renewed", renewedCount),
		zap.Int("downgraded", downgradedCount),
		zap.Int("skipped", skippedCount),
		zap.Int("failed", failedCount),
		zap.Int("renewed_count", outcomeCounts[renewalReasonRenewed]),
		zap.Int("downgraded_auto_renew_disabled", outcomeCounts[renewalReasonDowngradedAutoRenewOff]),
		zap.Int("downgraded_insufficient_balance", outcomeCounts[renewalReasonDowngradedInsufficient]),
		zap.Int("failed_missing_plan", outcomeCounts[renewalReasonFailedMissingPlan]),
		zap.Int("failed_invalid_price", outcomeCounts[renewalReasonFailedInvalidPrice]),
		zap.Int("failed_invalid_duration", outcomeCounts[renewalReasonFailedInvalidDuration]),
		zap.Int("failed_inactive_plan", outcomeCounts[renewalReasonFailedInactivePlan]),
		zap.Int("failed_wallet_missing", outcomeCounts[renewalReasonFailedWalletMissing]),
		zap.Int("failed_lock_subscription", outcomeCounts[renewalReasonFailedLockSubscription]),
		zap.Int("failed_lock_wallet", outcomeCounts[renewalReasonFailedLockWallet]),
		zap.Int("failed_processing", outcomeCounts[renewalReasonFailedProcessing]),
		zap.Int("skipped_nil_expires_at", outcomeCounts[renewalReasonSkippedNilExpiresAt]),
		zap.Int("skipped_not_found_after_lock", outcomeCounts[renewalReasonSkippedNotFoundAfterLock]),
	)
	if failedCount > 0 {
		return fmt.Errorf("scheduled renewal job completed with %d failed records", failedCount)
	}

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

// downgradeToFree updates the subscription record to default/free plan.
func (uc *SubscriptionUseCase) downgradeToFree(txCtx context.Context, lockedSub *entities.UserSubscription, freePlanID uuid.UUID, now time.Time) error {
	lockedSub.SubscriptionPlanID = freePlanID
	lockedSub.ExpiresAt = nil
	lockedSub.IsActive = true
	lockedSub.UpdatedAt = now

	return uc.userSubRepo.Update(txCtx, lockedSub)
}
