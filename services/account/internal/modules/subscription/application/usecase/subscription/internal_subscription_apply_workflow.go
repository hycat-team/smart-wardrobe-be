package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"

	"github.com/google/uuid"
)

// ApplySubscriptionPlanWithEvent applies or extends a plan AND returns the
// history row describing the change, without writing it.
//
// Splitting "compute" from "persist" is deliberate. The caller has to write the
// history row and the outbox record inside the same transaction as the
// subscription update, and the only component that knows the transaction is the
// caller. Returning the row lets the caller route it through the single publish
// point (PlanChangePublisher) instead of each call site inventing its own
// history write — which is how three of the nine plan-change paths ended up
// with no history at all.
//
// The event key embeds the resulting version, so a retried apply of the same
// account cannot produce two history rows for one change.
func ApplySubscriptionPlanWithEvent(
	txCtx context.Context,
	userSubRepo repositories.IUserSubscriptionRepository,
	userID uuid.UUID,
	plan *entities.SubscriptionPlan,
	now time.Time,
) (*entities.UserSubscriptionEvent, error) {
	sub, err := userSubRepo.GetByUserIDWithLock(txCtx, userID)
	if err != nil {
		return nil, apperror.NewInternalError("Không thể tải thông tin gói hiện tại.")
	}

	fromPlanCode := ""
	fromTierRank := 0
	isNewSub := false
	if sub == nil {
		isNewSub = true
		sub = &entities.UserSubscription{UserID: userID, CreatedAt: now}
	} else {
		// Captured before the fields are overwritten. The receiver uses it to
		// understand what the account is moving away from.
		fromPlanCode = sub.CurrentPlanCode
		fromTierRank = sub.CurrentTierRank
	}

	previousPlanID := sub.SubscriptionPlanID
	// 024 T166: captured before the fields are overwritten, for the same reason
	// previousPlanID is.
	previousExpiresAt := sub.ExpiresAt
	// 027/D-08: both callers of this workflow are ADMINISTRATIVE (administrator batch
	// grant and campaign grant). An administrative grant starts a fresh period at
	// `now`; stacking it onto a running term would give away days nobody paid for and
	// leave a `future` grant window for the AI service. Paid periods extend through
	// ResolveExpiryFor, which purchase settlement reaches via ApplyTransition.
	expiresAt := sub.ResolveAdminExpiryFor(plan, now)

	sub.SubscriptionPlanID = plan.ID
	sub.ExpiresAt = expiresAt
	sub.CurrentPlanCode = plan.Slug
	sub.CurrentTierRank = plan.TierRank
	sub.CurrentPlanKind = plan.PlanKind
	sub.StartedAt = now
	sub.Version++
	sub.UpdatedAt = now

	// 024 FR-070/FR-071: the benefit snapshot is refreshed even when a
	// subscription row already exists. Before this, re-granting the same plan
	// left the snapshot at whatever the first grant had written, so an
	// administrator changing a plan's benefits silently failed to reach
	// accounts that already had that plan.
	sub.CurrentBenefitSnapshot = BuildPlanBenefitSnapshot(plan)

	// A grant to a different plan must not leave a stale lifetime fallback or a
	// leftover auto-renew flag behind: both describe the previous plan, and a
	// reader that trusts them would restore the wrong plan on expiry.
	//
	// 024 T134: the auto-renew flag is CLEARED, not re-derived from the new plan.
	//
	// It used to be set to `plan.DurationDays != nil`, which turned auto-renew ON
	// for an account that was just handed a paid plan by an administrator. Nobody
	// had given consent to be charged again, and the consequence was concrete: when
	// the granted period ran out the cron worker would attempt a renewal payment
	// for a customer who had never bought anything. FR-053 asks for the flag to be
	// brought "in line with the new plan"; for an administrative grant, in line
	// means OFF.
	//
	// 024 T154: this comment used to close with "Only a purchase - where the
	// customer paid and opted in - may turn it on." That was wrong, and it was
	// wrong in the direction that hides a missing feature.
	//
	// No purchase turns it on. `SetAutoRenew` has exactly one production caller -
	// `SetAutoRenewStatus`, behind `PUT /subscriptions/me/auto-renew` - so today
	// the ONLY way auto-renew becomes true is the account owner asking for it. The
	// purchase path leaves the flag exactly as this block sets it: false.
	//
	// That is the correct state to be in. Consent has to come from somewhere the
	// customer can give it, and silently enrolling people to be charged is the
	// failure T134 removed. But it is worth being explicit, because the old wording
	// implied a purchase-time opt-in that does not exist: someone reading it would
	// assume a paying customer is already enrolled, and would not go looking for the
	// endpoint that is supposed to do it.
	// 024 T166: FR-053 fires on TWO conditions and only one was here. The
	// requirement reads "khi cấp gói cho tài khoản đang dùng gói khác, **hoặc thời
	// hạn gói hiện tại đã hết**" — the second one was missing, so re-granting the
	// SAME plan to an account whose term had already run out came out with a correct
	// expiry and everything else untouched.
	//
	// What it left behind is the part that costs money. An expired Premium account
	// carrying a fallback to the lifetime plan, re-granted Premium by an
	// administrator, kept auto-renew ON. When the new term ran out, the renewal job
	// would then charge a customer for a plan they never agreed to buy again — the
	// exact harm FR-053 gives as its reason for clearing the flag. The surviving
	// fallback group is contradictory in the same way: it describes a plan the
	// account is not on.
	//
	// `nil` is not treated as expired. A nil expiry means a lifetime plan, which is
	// the opposite condition, and those accounts are already covered by the
	// different-plan branch.
	termExpired := previousExpiresAt != nil && !previousExpiresAt.After(now)

	if isNewSub || previousPlanID != plan.ID || termExpired {
		sub.FallbackPlanID = nil
		sub.FallbackPlanCode = nil
		sub.FallbackTierRank = nil
		sub.FallbackPlanKind = nil
		sub.FallbackBenefitSnapshot = nil
		sub.IsAutoRenewEnabled = false
	}

	if isNewSub {
		if err := userSubRepo.Create(txCtx, sub); err != nil {
			return nil, apperror.NewInternalError("Không thể khởi tạo gói mới.")
		}
	} else {
		if err := userSubRepo.Update(txCtx, sub); err != nil {
			return nil, apperror.NewInternalError("Không thể cập nhật thời hạn gói.")
		}
	}

	toPlanCode := plan.Slug
	toTierRank := plan.TierRank
	eventType := subscriptioneventtype.ACTIVATED
	switch {
	case !isNewSub && fromPlanCode == plan.Slug:
		eventType = subscriptioneventtype.EXTENDED
	case !isNewSub && fromTierRank > 0 && plan.TierRank > fromTierRank:
		eventType = subscriptioneventtype.UPGRADED
	}

	return &entities.UserSubscriptionEvent{
		EventKey:     fmt.Sprintf("PLAN_APPLY:%s:%d", userID, sub.Version),
		UserID:       userID,
		EventType:    string(eventType),
		FromPlanCode: optionalString(fromPlanCode),
		FromTierRank: optionalInt(fromTierRank),
		ToPlanCode:   &toPlanCode,
		ToTierRank:   &toTierRank,
		OccurredAt:   now,
		EffectiveAt:  now,
	}, nil
}

// BuildPlanBenefitSnapshot captures the plan's customer-visible benefits at the
// moment of the grant.
//
// Only gate-1 values are captured. Gate 2 — the AI engine's internal caps — must
// never appear in a customer-facing document: a customer must not be able to
// read the engine's internal limits out of a subscription record.
func BuildPlanBenefitSnapshot(plan *entities.SubscriptionPlan) entities.JSONDocument {
	raw, err := json.Marshal(map[string]any{
		"maxWardrobeItems":   plan.MaxWardrobeItems,
		"maxOutfits":         plan.MaxOutfits,
		"aiOutfitDailyQuota": plan.AiOutfitDailyQuota,
		"aiChatDailyQuota":   plan.AiChatDailyQuota,
	})
	if err != nil {
		// A snapshot that cannot be marshalled would have to be reported; an
		// empty document keeps the NOT NULL column satisfiable and is corrected
		// on the next grant.
		return entities.JSONDocument(`{}`)
	}
	return entities.JSONDocument(raw)
}

func optionalString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func optionalInt(v int) *int {
	return &v
}

// 024 T178: `ApplySubscriptionPlan` is GONE, and this is the note explaining why.
//
// Its doc comment said it "exists only for the call sites feature 024 has not migrated
// yet", which described a migration that had already finished: there were no remaining
// call sites. It was a wrapper that called ApplySubscriptionPlanWithEvent and threw the
// history row away — the exact defect 024 exists to remove — and it was marked
// Deprecated in the same breath, so nobody should ever have added a new one.
//
// It is worth being explicit that it was called by NOTHING. A grep for it finds only
// its own definition, and a wrapper that discards the audit history is a trap for the
// next person who reads it looking for "the simple way to apply a plan".
