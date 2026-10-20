package subscription

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// 024 T166 / FR-053: re-granting the SAME plan to an account whose term has already
// run out must clear the fallback group and the auto-renew flag, exactly like a grant
// to a different plan does.
//
// FR-053 fires on two conditions — "khi cấp gói cho tài khoản đang dùng gói khác,
// hoặc thời hạn gói hiện tại đã hết" — and only the first was implemented.
//
// The scenario below is the one that costs money rather than merely being untidy. An
// expired Premium account carrying a fallback to the lifetime plan, re-granted Premium
// by an administrator, used to keep auto-renew ON. When the new term ran out the
// renewal job would then attempt to charge a customer for a plan they had never agreed
// to buy again — which is the harm FR-053 gives as its reason for clearing the flag.

func premiumPlanWithTerm(days int) *entities.SubscriptionPlan {
	return &entities.SubscriptionPlan{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		Slug:            "premium",
		Name:            "Premium",
		TierRank:        2,
		PlanKind:        plankind.Finite,
		DurationDays:    &days,
	}
}

func TestRegrantingTheSameExpiredPlanClearsFallbackAndAutoRenew(t *testing.T) {
	now := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	userID := uuid.New()
	premium := premiumPlanWithTerm(30)

	lifetimeID := uuid.New()
	lifetimeCode := "basic-lifetime"
	lifetimeRank := 1
	lifetimeKind := plankind.Lifetime

	// The account is on Premium, the term ran out a day ago, auto-renew is on, and a
	// fallback points at the lifetime plan.
	expired := now.AddDate(0, 0, -1)
	sub := &entities.UserSubscription{
		UserID:                  userID,
		SubscriptionPlanID:      premium.ID,
		CurrentPlanCode:         premium.Slug,
		CurrentTierRank:         premium.TierRank,
		CurrentPlanKind:         premium.PlanKind,
		StartedAt:               now.AddDate(0, 0, -31),
		ExpiresAt:               &expired,
		IsAutoRenewEnabled:      true,
		FallbackPlanID:          &lifetimeID,
		FallbackPlanCode:        &lifetimeCode,
		FallbackTierRank:        &lifetimeRank,
		FallbackPlanKind:        &lifetimeKind,
		FallbackBenefitSnapshot: entities.JSONDocument(`{"legacy":true}`),
	}

	repo := &fakeGrantSubRepo{subs: map[uuid.UUID]*entities.UserSubscription{userID: sub}}

	if _, err := ApplySubscriptionPlanWithEvent(context.Background(), repo, userID, premium, now); err != nil {
		t.Fatalf("re-granting an expired plan must succeed: %v", err)
	}

	got := repo.subs[userID]

	// The expiry must be the fresh one, and this is the half of FR-053 that already
	// worked. Asserting it keeps the test honest about what it is protecting: the
	// flags below, not the date.
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(now.AddDate(0, 0, 30)) {
		t.Fatalf("must start a new period at the moment of the grant, got = %v", got.ExpiresAt)
	}

	if got.IsAutoRenewEnabled {
		t.Error("the auto-renew flag must be OFF: the plan had expired and was re-granted, and nobody consented to being charged again")
	}
	if got.FallbackPlanID != nil || got.FallbackPlanCode != nil ||
		got.FallbackTierRank != nil || got.FallbackPlanKind != nil ||
		got.FallbackBenefitSnapshot != nil {
		t.Errorf("the whole fallback field group must be cleared, got = id %v / code %v / rank %v / kind %v / snapshot %v",
			got.FallbackPlanID, got.FallbackPlanCode, got.FallbackTierRank,
			got.FallbackPlanKind, got.FallbackBenefitSnapshot)
	}
}

// The guard against over-clearing: a live term on the same plan must not lose the
// auto-renew flag. FR-053 does not ask for it to be cleared here, and clearing it
// would silently unsubscribe paying customers every time a plan was re-granted early.
//
// 027/D-08 changed the DATE half of this test: this workflow is administrative, so
// the period restarts at `now` instead of stacking onto the remaining term.
func TestRegrantingALiveSamePlanKeepsAutoRenewAndStartsAFreshPeriod(t *testing.T) {
	now := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	userID := uuid.New()
	premium := premiumPlanWithTerm(30)

	live := now.AddDate(0, 0, 10)
	sub := &entities.UserSubscription{
		UserID:             userID,
		SubscriptionPlanID: premium.ID,
		CurrentPlanCode:    premium.Slug,
		CurrentTierRank:    premium.TierRank,
		CurrentPlanKind:    premium.PlanKind,
		StartedAt:          now.AddDate(0, 0, -20),
		ExpiresAt:          &live,
		IsAutoRenewEnabled: true,
	}

	repo := &fakeGrantSubRepo{subs: map[uuid.UUID]*entities.UserSubscription{userID: sub}}

	if _, err := ApplySubscriptionPlanWithEvent(context.Background(), repo, userID, premium, now); err != nil {
		t.Fatalf("renewing a plan that is still in force must succeed: %v", err)
	}

	if !repo.subs[userID].IsAutoRenewEnabled {
		t.Error("renewing a plan still in force must not clear the auto-renew flag: a paying customer already consented")
	}
	// An ADMINISTRATIVE grant replaces the remaining term: nobody paid for the extra
	// days, and stacking them would also leave the old period as the grant window the
	// AI service reads (027/D-08).
	want := now.AddDate(0, 0, 30)
	if got := repo.subs[userID].ExpiresAt; got == nil || !got.Equal(want) {
		t.Errorf("an administrative grant must start a fresh period at now, got = %v, want = %v", got, want)
	}
}

// A lifetime plan has a nil expiry, which is the opposite of "expired". Guarding on
// `!ExpiresAt.After(now)` without the nil check would classify every lifetime
// subscription as expired and clear its auto-renew flag on any grant.
func TestLifetimePlanExpiryIsNotTreatedAsExpired(t *testing.T) {
	now := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	userID := uuid.New()

	// No DurationDays means a lifetime plan, so ExpiresAt is nil.
	lifetime := &entities.SubscriptionPlan{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		Slug:            "basic-lifetime",
		Name:            "Basic Lifetime",
		TierRank:        1,
		PlanKind:        plankind.Lifetime,
		DurationDays:    nil,
	}
	sub := &entities.UserSubscription{
		UserID:             userID,
		SubscriptionPlanID: lifetime.ID,
		CurrentPlanCode:    lifetime.Slug,
		CurrentTierRank:    lifetime.TierRank,
		CurrentPlanKind:    lifetime.PlanKind,
		StartedAt:          now.AddDate(0, 0, -400),
		ExpiresAt:          nil,
		IsAutoRenewEnabled: true,
	}

	repo := &fakeGrantSubRepo{subs: map[uuid.UUID]*entities.UserSubscription{userID: sub}}

	// Re-granting the SAME lifetime plan: a change of plan would clear the flag anyway,
	// so this only isolates the expiry condition.
	if _, err := ApplySubscriptionPlanWithEvent(context.Background(), repo, userID, lifetime, now); err != nil {
		t.Fatalf("re-granting a lifetime plan must succeed: %v", err)
	}

	if !repo.subs[userID].IsAutoRenewEnabled {
		t.Error("a nil expiry is a LIFETIME plan, not an expired one: the auto-renew flag must not be cleared")
	}
}
