package subscription

import (
	"testing"
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/subscription/plankind"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

func TestEvaluateLifetimeTransitions(t *testing.T) {
	current := EffectiveSubscription{PlanCode: "basic-lifetime", TierRank: 1, PlanKind: plankind.Lifetime}
	if got := EvaluateSubscriptionTransition(CreateValidation, current, PaymentSnapshot{TierRank: 1, PlanKind: plankind.Lifetime}); got != RejectSameLifetime {
		t.Fatalf("expected same lifetime rejection, got %s", got)
	}
	if got := EvaluateSubscriptionTransition(PaymentCompletion, current, PaymentSnapshot{TierRank: 1, PlanKind: plankind.Lifetime}); got != CreditWalletSameLifetime {
		t.Fatalf("expected wallet credit, got %s", got)
	}
	if got := EvaluateSubscriptionTransition(PaymentCompletion, current, PaymentSnapshot{TierRank: 2, PlanKind: plankind.Finite}); got != OverlayLifetimeWithFinite {
		t.Fatalf("expected finite overlay, got %s", got)
	}
}

func TestExpiredOverlayResolvesAndRestoresFallback(t *testing.T) {
	now := time.Date(2026, 6, 20, 23, 30, 0, 0, time.FixedZone("Asia/Ho_Chi_Minh", 7*3600))
	expired := now.Add(-time.Hour)
	fallbackID := uuid.New()
	fallbackCode := "basic-lifetime"
	fallbackRank := 1
	fallbackKind := plankind.Lifetime
	sub := &entities.UserSubscription{UserID: uuid.New(), SubscriptionPlanID: uuid.New(), CurrentPlanCode: "premium", CurrentTierRank: 3, CurrentPlanKind: plankind.Finite, StartedAt: now.AddDate(0, -1, 0), ExpiresAt: &expired, FallbackPlanID: &fallbackID, FallbackPlanCode: &fallbackCode, FallbackTierRank: &fallbackRank, FallbackPlanKind: &fallbackKind}
	effective, err := ResolveEffectiveSubscription(sub, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if effective.PlanCode != fallbackCode || effective.PlanKind != plankind.Lifetime {
		t.Fatalf("unexpected effective subscription: %#v", effective)
	}
	if !RestoreLifetimeFallbackIfExpired(sub, now) {
		t.Fatal("expected fallback restore")
	}
	if sub.CurrentPlanCode != fallbackCode || sub.ExpiresAt != nil || sub.FallbackPlanID != nil || sub.StartedAt != expired {
		t.Fatalf("unexpected restored projection: %#v", sub)
	}
}

func TestFreeActivatesPaidPlan(t *testing.T) {
	current := EffectiveSubscription{PlanCode: "free", TierRank: 0, PlanKind: plankind.DefaultFree}
	if got := EvaluateSubscriptionTransition(CreateValidation, current, PaymentSnapshot{TierRank: 1, PlanKind: plankind.Finite}); got != ActivateFinite {
		t.Fatalf("expected activation, got %s", got)
	}
}

func TestAddProductDaysKeepsLocalWallClock(t *testing.T) {
	location := time.FixedZone("Asia/Ho_Chi_Minh", 7*3600)
	base := time.Date(2026, 6, 20, 23, 30, 0, 0, location)
	got := AddProductDays(base, 30, location)
	want := time.Date(2026, 7, 20, 23, 30, 0, 0, location)
	if !got.Equal(want) || got.Hour() != 23 || got.Minute() != 30 {
		t.Fatalf("unexpected calendar-day result: %s", got)
	}
}

func TestUserSubscriptionDomainMethods(t *testing.T) {
	now := time.Now().UTC()
	userID := uuid.New()
	freePlanID := uuid.New()
	freePlanCode := "default-free"
	freePlanRank := 0
	freePlanKind := plankind.DefaultFree

	// Initial default-free subscription
	sub := &entities.UserSubscription{
		UserID:             userID,
		SubscriptionPlanID: freePlanID,
		CurrentPlanCode:    freePlanCode,
		CurrentTierRank:    freePlanRank,
		CurrentPlanKind:    freePlanKind,
		StartedAt:          now,
	}

	premium1PlanID := uuid.New()
	premium1Plan := &entities.SubscriptionPlan{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: premium1PlanID}},
		Slug:            "premium-1",
		Name:            "Premium 1",
		TierRank:        1,
		PlanKind:        plankind.Finite,
		DurationDays:    intPtr(30),
	}

	// 1. Activate finite plan from default-free
	evt, err := sub.Activate(premium1Plan, entities.JSONDocument(`{}`), "EVT_ACTIVATE", now)
	if err != nil {
		t.Fatalf("unexpected error on Activate: %v", err)
	}
	if evt.EventType != "ACTIVATED" || sub.CurrentPlanCode != "premium-1" || sub.Version != 1 {
		t.Fatalf("unexpected state after Activate: %#v", sub)
	}
	if sub.ExpiresAt == nil || !sub.ExpiresAt.Equal(now.AddDate(0, 0, 30)) {
		t.Fatalf("expected ExpiresAt to be 30 days from now, got %v", sub.ExpiresAt)
	}

	// 2. Extend finite plan (same plan/tier)
	evt, err = sub.Extend(premium1Plan, entities.JSONDocument(`{}`), "EVT_EXTEND", now)
	if err != nil {
		t.Fatalf("unexpected error on Extend: %v", err)
	}
	if evt.EventType != "EXTENDED" || sub.Version != 2 {
		t.Fatalf("unexpected state after Extend: %#v", sub)
	}
	// ExpiresAt should stack to 60 days from now since current ExpiresAt is in the future
	if sub.ExpiresAt == nil || !sub.ExpiresAt.Equal(now.AddDate(0, 0, 60)) {
		t.Fatalf("expected ExpiresAt to be 60 days from now, got %v", sub.ExpiresAt)
	}

	// 3. Upgrade to a higher tier plan
	premium2PlanID := uuid.New()
	premium2Plan := &entities.SubscriptionPlan{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: premium2PlanID}},
		Slug:            "premium-2",
		Name:            "Premium 2",
		TierRank:        2,
		PlanKind:        plankind.Finite,
		DurationDays:    intPtr(15),
	}
	evt, err = sub.Upgrade(premium2Plan, entities.JSONDocument(`{}`), "EVT_UPGRADE", now)
	if err != nil {
		t.Fatalf("unexpected error on Upgrade: %v", err)
	}
	if evt.EventType != "UPGRADED" || sub.CurrentPlanCode != "premium-2" || sub.Version != 3 {
		t.Fatalf("unexpected state after Upgrade: %#v", sub)
	}
	// ExpiresAt should be exactly now + 15 days because it's a tier switch (doesn't stack)
	if sub.ExpiresAt == nil || !sub.ExpiresAt.Equal(now.AddDate(0, 0, 15)) {
		t.Fatalf("expected ExpiresAt to be 15 days from now, got %v", sub.ExpiresAt)
	}

	// 4. Toggle auto-renew
	evt, err = sub.SetAutoRenew(true, "EVT_AUTO_RENEW", now)
	if err != nil {
		t.Fatalf("unexpected error on SetAutoRenew: %v", err)
	}
	if evt.EventType != "AUTO_RENEW_UPDATED" || !sub.IsAutoRenewEnabled || sub.Version != 4 {
		t.Fatalf("unexpected state after SetAutoRenew: %#v", sub)
	}

	// 5. Downgrade to default-free
	freePlan := &entities.SubscriptionPlan{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: freePlanID}},
		Slug:            freePlanCode,
		Name:            "Default Free",
		TierRank:        freePlanRank,
		PlanKind:        plankind.DefaultFree,
	}
	evt, err = sub.DowngradeToFree(freePlan, "EVT_DOWNGRADE", now)
	if err != nil {
		t.Fatalf("unexpected error on DowngradeToFree: %v", err)
	}
	if evt.EventType != "SUBSCRIPTION_DOWNGRADED_TO_FREE" || sub.CurrentPlanCode != freePlanCode || sub.Version != 5 {
		t.Fatalf("unexpected state after DowngradeToFree: %#v", sub)
	}
	if sub.ExpiresAt != nil || sub.IsAutoRenewEnabled {
		t.Fatalf("expected ExpiresAt to be nil and IsAutoRenewEnabled to be false, got %v and %v", sub.ExpiresAt, sub.IsAutoRenewEnabled)
	}
}

func intPtr(val int) *int {
	return &val
}
