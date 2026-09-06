package purchase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	aiexecutor_dto "smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	notificationContract "smart-wardrobe-be/internal/modules/notification/presentation/contract"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/wallet"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/walletstatementtype"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/internal/pkg/utils/timeutils"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"

	"github.com/google/uuid"
)

func (uc *SubscriptionPurchaseUseCase) PurchasePlanWithWallet(ctx context.Context, userID uuid.UUID, planSlug string) error {
	plan, err := uc.planRepo.GetBySlug(ctx, planSlug)
	if err != nil {
		return subscriptionerrors.ErrSearchPlanFailed()
	}
	if plan == nil {
		return subscriptionerrors.ErrRequestedPlanNotFound()
	}

	now := timeutils.GetNow(uc.cfg.Database.TimeZone)
	var subEvent *entities.UserSubscriptionEvent

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
			if err := uc.eventRepo.Create(txCtx, innerSubEvent); err != nil {
				return err
			}
			subEvent = innerSubEvent

			payload := aiexecutor_dto.PlanChangedEventPayload{
				UserID:         userID.String(),
				EventType:      subEvent.EventType,
				ToPlanCode:     plan.Slug,
				ToTierRank:     plan.TierRank,
				AICostPolicyID: plan.AICostPolicyID.String(),
				OccurredAt:     now.Format(time.RFC3339Nano),
			}
			if err := uc.eventPublisher.Publish(txCtx, eventconstants.TopicSubscriptionPlanChanged, payload); err != nil {
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

	return nil
}
