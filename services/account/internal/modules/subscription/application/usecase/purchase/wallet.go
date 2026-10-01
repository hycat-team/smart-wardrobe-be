package purchase

import (
	"context"
	"encoding/json"
	"fmt"

	notificationContract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/wallet"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/walletstatementtype"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	sharedmoney "smart-wardrobe-be/shared/domain/money"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/google/uuid"
)

// subscriptionPlanChangedPayload was the hand-written six-field schema for
// TopicSubscriptionPlanChanged. It is gone as of feature 024: it could not
// carry toPlanId, effectiveTo or durationDays, so the AI receiver could not
// fill ai_policy_grants.plan_id (NOT NULL, foreign key dropped) nor compute the
// start of a renewal period. Every publish now goes through
// subscription.PlanChangePublisher, which builds the full 14-field payload in
// the subscription module's mapper.
//
// A DTO type is still never shared across services — the JSON is the contract.
// The receiver keeps its own declaration in services/ai/dto/aiexecutor.

func (uc *SubscriptionPurchaseUseCase) PurchasePlanWithWallet(ctx context.Context, userID uuid.UUID, planSlug string) error {
	plan, err := uc.planRepo.GetBySlug(ctx, planSlug)
	if err != nil {
		return subscriptionerrors.ErrSearchPlanFailed()
	}
	if plan == nil {
		return subscriptionerrors.ErrRequestedPlanNotFound()
	}

	now := timeutils.GetNow(uc.cfg.Database.TimeZone)

	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		sub, isNewSub, err := uc.getOrInitLockedSubscriptionForPurchase(txCtx, userID, now)
		if err != nil {
			return err
		}

		var lockedSub *entities.UserSubscription
		if !isNewSub {
			lockedSub = sub
		}
		if err := uc.validatePurchase(txCtx, userID, plan, lockedSub, now); err != nil {
			return err
		}

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

		freePlan, err := uc.planRepo.GetDefaultPlan(txCtx)
		if err != nil {
			return err
		}
		effective, err := subscription.ResolveEffectiveSubscription(sub, freePlan, now)
		if err != nil {
			return err
		}

		paymentSnapshot := subscription.PaymentSnapshot{
			PlanID:          plan.ID.String(),
			PlanCode:        plan.Slug,
			TierRank:        plan.TierRank,
			PlanKind:        plan.PlanKind,
			DurationDays:    plan.DurationDays,
			BenefitSnapshot: snapshot,
		}

		transition := subscription.EvaluateSubscriptionTransition(subscription.CreateValidation, effective, paymentSnapshot)

		var innerSubEvent *entities.UserSubscriptionEvent
		switch transition {
		case subscription.ActivateFinite, subscription.ActivateLifetime:
			innerSubEvent, err = sub.Activate(plan, snapshot, "", now)
		case subscription.ExtendFinite:
			innerSubEvent, err = sub.Extend(plan, snapshot, "", now)
		case subscription.UpgradeFinite, subscription.UpgradeLifetime:
			innerSubEvent, err = sub.Upgrade(plan, snapshot, "", now)
		case subscription.OverlayLifetimeWithFinite:
			innerSubEvent, err = sub.OverlayLifetime(plan, snapshot, "", now)
		default:
			return subscriptionerrors.ErrSubscriptionStillActive()
		}
		if err != nil {
			return err
		}

		if !plan.Price.IsZero() {
			walletEntity, _, err := uc.getOrInitLockedWalletForPurchase(txCtx, userID, now)
			if err != nil {
				return err
			}
			if walletEntity.Balance.LessThan(plan.Price) {
				return subscriptionerrors.ErrWalletInsufficientBalance()
			}
			metadata := wallet.WalletStatementMetadata{
				SourcePlanCode:             &plan.Slug,
				SourceTierRank:             &plan.TierRank,
				ActiveTierRankAtCompletion: &plan.TierRank,
				RenewalAttemptKey:          nil,
			}
			description := fmt.Sprintf("Đăng ký gói hội viên %s thành công qua ví nội bộ", plan.Name)
			if err := wallet.ProcessWalletTransaction(
				txCtx,
				uc.walletRepo,
				uc.statementRepo,
				userID,
				plan.Price.Neg(),
				walletstatementtype.SubscriptionPurchase,
				description,
				nil,
				metadata,
				now,
			); err != nil {
				return err
			}
		}

		if err := uc.persistSubscriptionForPurchase(txCtx, sub, isNewSub); err != nil {
			return err
		}

		if innerSubEvent != nil {
			// 024 T045: one helper writes the history row and the outbox record
			// in this transaction. The hand-written six-field payload that used
			// to live here is gone — it could not carry toPlanId or the renewal
			// window, so the receiver could not write a valid grant.
			if _, err := uc.planChangePublisher.PublishPlanChange(txCtx, innerSubEvent, sub, plan); err != nil {
				return err
			}
		}

		if user, userErr := uc.userBriefContract.GetByID(txCtx, userID); userErr == nil && user != nil && user.Email != "" {
			expiresAt := ""
			if sub.ExpiresAt != nil {
				expiresAt = sub.ExpiresAt.Format("02/01/2006")
			}
			walletEntity, _ := uc.walletRepo.GetByUserID(txCtx, userID)
			newBalance := sharedmoney.Zero
			if walletEntity != nil {
				newBalance = walletEntity.Balance
			}
			_ = uc.notifContract.Send(txCtx, notificationContract.ChannelEmail, notificationContract.TemplateWalletPurchaseSuccess, map[string]any{
				"PlanName":   plan.Name,
				"Amount":     sharedmoney.ToFloat(plan.Price),
				"NewBalance": sharedmoney.ToFloat(newBalance),
				"ExpiresAt":  expiresAt,
			}, user.Email)
		}

		return nil
	}); err != nil {
		return err
	}

	// 024 T154: was `_ = uc.overviewCache.Invalidate(ctx, userID)` - the same
	// post-commit defect T151 fixed on the grant path, copied. The top-up has
	// already committed here, so a cancelled client context or an expired deadline
	// made the delete fail silently and the overview kept serving the old wallet and
	// plan state until its TTL. Now shared with the other three call sites.
	subscription.InvalidateOverviewCache(ctx, uc.log, uc.overviewCache, userID)

	return nil
}
