package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	notificationContract "smart-wardrobe-be/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/wallet"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/subscriptionrenewalstatus"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/walletstatementtype"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"

	"go.uber.org/zap"
)

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

	snapshot, err := buildRenewalBenefitSnapshot(plan)
	if err != nil {
		return err
	}

	if err := uc.debitWalletForRenewal(txCtx, lockedSub, plan, now, result.RenewalAttemptKey); err != nil {
		return err
	}

	return uc.completeRenewalExtension(txCtx, lockedSub, userWallet, plan, snapshot, now, result)
}

// buildRenewalBenefitSnapshot serializes the plan benefits stored on the subscription.
func buildRenewalBenefitSnapshot(plan *entities.SubscriptionPlan) (entities.JSONDocument, error) {
	snapshotBytes, err := json.Marshal(map[string]any{
		"maxWardrobeItems":   plan.MaxWardrobeItems,
		"maxOutfits":         plan.MaxOutfits,
		"aiOutfitDailyQuota": plan.AiOutfitDailyQuota,
		"aiChatDailyQuota":   plan.AiChatDailyQuota,
	})
	if err != nil {
		return nil, err
	}
	return entities.JSONDocument(snapshotBytes), nil
}

// debitWalletForRenewal charges the renewal price to the user's wallet.
func (uc *SubscriptionUseCase) debitWalletForRenewal(
	txCtx context.Context,
	lockedSub *entities.UserSubscription,
	plan *entities.SubscriptionPlan,
	now time.Time,
	renewalAttemptKey string,
) error {
	metadata := wallet.WalletStatementMetadata{
		SourcePlanCode:             &plan.Slug,
		SourceTierRank:             &plan.TierRank,
		ActiveTierRankAtCompletion: &plan.TierRank,
		RenewalAttemptKey:          &renewalAttemptKey,
	}
	desc := fmt.Sprintf("Auto-renewed subscription plan %s", plan.Name)
	return wallet.ProcessWalletTransaction(
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
	)
}

// completeRenewalExtension extends the subscription, records the ledger event,
// marks the renewal attempt succeeded and notifies the user.
func (uc *SubscriptionUseCase) completeRenewalExtension(
	txCtx context.Context,
	lockedSub *entities.UserSubscription,
	userWallet *entities.UserWallet,
	plan *entities.SubscriptionPlan,
	snapshot entities.JSONDocument,
	now time.Time,
	result *RenewalExecutionResult,
) error {
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
		attempt.Status = subscriptionrenewalstatus.Succeeded
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
		zap.String("result", string(renewalStatusRenewed)),
		zap.String("reason", result.Reason),
	)

	_ = uc.notifContract.SendByUserID(txCtx, notificationContract.ChannelEmail, notificationContract.TemplateRenewalSuccess, map[string]any{
		"PlanName":     plan.Name,
		"Amount":       sharedmoney.ToFloat(plan.Price),
		"NewExpiresAt": lockedSub.ExpiresAt.Format("02/01/2006"),
	}, lockedSub.UserID)

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
