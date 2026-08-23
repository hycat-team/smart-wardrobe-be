package webhook

import (
	"context"
	"fmt"
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefitresolution"
	notificationContract "smart-wardrobe-be/internal/modules/notification/presentation/contract"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/wallet"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/walletstatementtype"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"
)

func (uc *PaymentWebhookUseCase) executeDirectPurchaseWorkflow(txCtx context.Context, tx *entities.DepositTransaction, now time.Time) error {
	if tx.SubscriptionPlanID == nil {
		return subscriptionerrors.ErrLinkedPlanNotFound()
	}

	plan, err := uc.planRepo.GetByID(txCtx, *tx.SubscriptionPlanID)
	if err != nil {
		return subscriptionerrors.ErrLoadPlanFailed()
	}
	if plan == nil {
		return subscriptionerrors.ErrRequestedPlanNotFound()
	}

	if tx.PlanCode != nil {
		plan.Slug = *tx.PlanCode
	}
	if tx.PlanName != nil {
		plan.Name = *tx.PlanName
	}
	if tx.TierRank != nil {
		plan.TierRank = *tx.TierRank
	}
	if tx.PlanKind != nil {
		plan.PlanKind = *tx.PlanKind
	}
	plan.DurationDays = tx.PurchasedDurationDays
	free, err := uc.planRepo.GetDefaultPlan(txCtx)
	if err != nil || free == nil {
		return subscriptionerrors.ErrDefaultPlanLoadFailed()
	}
	sub, err := uc.userSubRepo.GetByUserIDWithLock(txCtx, tx.UserID)
	if err != nil {
		return err
	}
	if sub == nil {
		sub = &entities.UserSubscription{UserID: tx.UserID, SubscriptionPlanID: free.ID, SubscriptionPlan: free, CurrentPlanCode: free.Slug, CurrentTierRank: free.TierRank, CurrentPlanKind: plankind.DefaultFree, CurrentBenefitSnapshot: entities.JSONDocument(`{}`), StartedAt: now}
		if err := uc.userSubRepo.ProvisionDefault(txCtx, sub); err != nil {
			return err
		}
		sub, err = uc.userSubRepo.GetByUserIDWithLock(txCtx, tx.UserID)
		if err != nil {
			return err
		}
	}
	if sub.ExpiresAt != nil && !sub.ExpiresAt.After(now) && sub.FallbackPlanID != nil {
		subEvent, err := sub.RestoreFallback("", now)
		if err != nil {
			return err
		}
		if err := uc.userSubRepo.Update(txCtx, sub); err != nil {
			return err
		}
		if subEvent != nil {
			key := fmt.Sprintf("LIFETIME_FALLBACK_RESTORED:%s:%s", sub.UserID, sub.StartedAt.UTC().Format(time.RFC3339))
			subEvent.EventKey = key
			if existing, err := uc.subscriptionEventRepo.GetByKey(txCtx, key); err != nil {
				return err
			} else if existing == nil {
				if err := uc.subscriptionEventRepo.Create(txCtx, subEvent); err != nil {
					return err
				}
			}
		}
	}
	effective, err := subscription.ResolveEffectiveSubscription(sub, free, now)
	if err != nil {
		return err
	}
	transition := subscription.EvaluateSubscriptionTransition(subscription.PaymentCompletion, effective, subscription.PaymentSnapshot{PlanCode: plan.Slug, TierRank: plan.TierRank, PlanKind: plan.PlanKind, DurationDays: plan.DurationDays, BenefitSnapshot: tx.BenefitSnapshot})
	if transition == subscription.CreditWalletLowerTier || transition == subscription.CreditWalletSameLifetime {
		statementType, resolution := walletstatementtype.LowerTierPaymentCredit, benefitresolution.LowerTierPaymentCreditedToWallet
		if transition == subscription.CreditWalletSameLifetime {
			statementType, resolution = walletstatementtype.SameLifetimePaymentCredit, benefitresolution.SameLifetimePaymentCreditedToWallet
		}
		metadata := wallet.WalletStatementMetadata{}
		if err := wallet.ProcessWalletTransaction(txCtx, uc.walletRepo, uc.statementRepo, tx.UserID, tx.Amount, statementType, "Khoản thanh toán được hoàn vào ví do gói hiện tại có quyền lợi cao hơn hoặc tương đương", &tx.ID, metadata, now); err != nil {
			return err
		}
		tx.BenefitResolution = &resolution
		credit := tx.Amount
		tx.WalletCreditAmount = &credit
		return nil
	}
	before := sub.Version
	resolution, subEvent, err := subscription.ApplyTransition(sub, plan, tx.BenefitSnapshot, transition, now)
	if err != nil {
		return err
	}
	sub.LastDepositTransactionID = &tx.ID
	if err := uc.userSubRepo.Update(txCtx, sub); err != nil {
		return err
	}
	eventKey := fmt.Sprintf("PAYMENT_BENEFIT:%s", tx.ID)
	if subEvent != nil {
		subEvent.EventKey = eventKey
		subEvent.SourceDepositTransactionID = &tx.ID
		if existing, err := uc.subscriptionEventRepo.GetByKey(txCtx, eventKey); err != nil {
			return err
		} else if existing == nil {
			if err := uc.subscriptionEventRepo.Create(txCtx, subEvent); err != nil {
				return err
			}
		}
	}
	after := sub.Version
	tx.BenefitResolution = &resolution
	if subEvent != nil {
		tx.BenefitResultSnapshot = subEvent.Metadata
	}
	tx.SubscriptionVersionBefore = &before
	tx.SubscriptionVersionAfter = &after

	if user, userErr := uc.userBriefContract.GetByID(txCtx, tx.UserID); userErr == nil && user != nil && user.Email != "" {
		expiresAt := ""
		if sub.ExpiresAt != nil {
			expiresAt = sub.ExpiresAt.Format("02/01/2006")
		}
		_ = uc.notifContract.Send(txCtx, notificationContract.ChannelEmail, notificationContract.TemplatePaymentSuccess, map[string]any{
			"PlanName":  plan.Name,
			"Amount":    sharedmoney.ToFloat(tx.Amount),
			"ExpiresAt": expiresAt,
		}, user.Email)
	}

	return nil
}
