package usecase

import (
	"context"
	"fmt"
	"time"

	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
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
	now := timeutils.GetNow(uc.cfg.Database.TimeZone)
	freePlanID, err := uc.planContract.GetDefaultSubscriptionPlanID(ctx)
	if err != nil {
		return errorcode.NewInternalError("Lỗi khi tải thông tin cấu hình gói hội viên mặc định")
	}

	var lastUserID uuid.UUID
	var lastExpiresAt time.Time
	limit := 100

	var renewedCount, downgradedCount, skippedCount, failedCount int
	outcomeCounts := map[string]int{}

	for {
		expiredSubs, err := uc.userSubRepo.GetActiveExpiredSubscriptionsBatch(ctx, now, lastUserID, lastExpiresAt, limit)
		if err != nil {
			return errorcode.NewInternalError("Lỗi khi truy vấn danh sách gói hội viên hết hạn")
		}

		if len(expiredSubs) == 0 {
			break
		}

		for _, sub := range expiredSubs {
			if sub.ExpiresAt == nil {
				skippedCount++
				outcomeCounts[renewalReasonSkippedNilExpiresAt]++
				continue
			}

			cursorUserID := sub.UserID
			cursorExpiresAt := *sub.ExpiresAt

			lastUserID = cursorUserID
			lastExpiresAt = cursorExpiresAt

			resultStatus := renewalStatusSkipped
			resultReason := renewalReasonFailedProcessing

			processSubFn := func(txCtx context.Context) error {
				resultReason = renewalReasonFailedLockSubscription
				lockedSub, err := uc.userSubRepo.GetActiveExpiredSubscriptionByUserIDWithLock(txCtx, sub.UserID, now)
				if err != nil {
					return err
				}
				if lockedSub == nil {
					resultStatus = renewalStatusSkipped
					resultReason = renewalReasonSkippedNotFoundAfterLock
					return nil
				}

				plan, days, validationReason, err := uc.validateRenewalPlan(lockedSub)
				if err != nil {
					resultStatus = renewalStatusFailed
					resultReason = validationReason
					return err
				}

				if !lockedSub.IsAutoRenewEnabled {
					resultReason = renewalReasonDowngradedAutoRenewOff
					if err := uc.downgradeToFree(txCtx, lockedSub, freePlanID, now); err != nil {
						return err
					}
					resultStatus = renewalStatusDowngraded
					return nil
				}

				resultReason = renewalReasonFailedLockWallet
				wallet, err := uc.walletRepo.GetByUserIDWithLock(txCtx, lockedSub.UserID)
				if err != nil {
					return err
				}
				if wallet == nil {
					resultStatus = renewalStatusFailed
					resultReason = renewalReasonFailedWalletMissing
					return fmt.Errorf("missing wallet for renewal")
				}

				if wallet.Balance.LessThan(plan.Price) {
					resultReason = renewalReasonDowngradedInsufficient
					if err := uc.downgradeToFree(txCtx, lockedSub, freePlanID, now); err != nil {
						return err
					}
					resultStatus = renewalStatusDowngraded
					return nil
				}

				resultReason = renewalReasonRenewed
				prevBalance := wallet.Balance
				wallet.Balance = wallet.Balance.Sub(plan.Price)
				wallet.UpdatedAt = now

				if err := uc.walletRepo.Update(txCtx, wallet); err != nil {
					return err
				}

				newExpiry := now.AddDate(0, 0, days)
				lockedSub.ExpiresAt = &newExpiry
				lockedSub.IsActive = true
				lockedSub.UpdatedAt = now

				if err := uc.userSubRepo.Update(txCtx, lockedSub); err != nil {
					return err
				}

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

				resultStatus = renewalStatusRenewed
				return nil
			}

			if err = uc.uow.Execute(ctx, processSubFn); err != nil {
				failedCount++
				outcomeCounts[resultReason]++
				uc.log.Error("Failed to process subscription renewal",
					zap.String("userID", sub.UserID.String()),
					zap.Time("expiresAt", *sub.ExpiresAt),
					zap.String("reason", resultReason),
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

		if len(expiredSubs) < limit {
			break
		}
	}

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

func (uc *SubscriptionUseCase) downgradeToFree(txCtx context.Context, lockedSub *entities.UserSubscription, freePlanID uuid.UUID, now time.Time) error {
	lockedSub.SubscriptionPlanID = freePlanID
	lockedSub.ExpiresAt = nil
	lockedSub.IsActive = true
	lockedSub.UpdatedAt = now

	return uc.userSubRepo.Update(txCtx, lockedSub)
}
