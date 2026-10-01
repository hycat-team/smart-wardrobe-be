package entities

import (
	"testing"
	"time"

	plankind "smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"

	"github.com/google/uuid"
)

// TestOverlayLifetimeKeepsFallbackPlanPointerIndependent guards the lifetime
// fallback fields of a lifetime subscription. OverlayLifetime used to point
// FallbackPlanID/FallbackPlanCode straight at SubscriptionPlanID/CurrentPlanCode,
// so the overlay immediately overwrote the values behind those pointers and the
// lifetime plan was lost. RestoreFallback then emitted
// LIFETIME_FALLBACK_RESTORED with to_plan_code == from_plan_code and the account
// stayed on the overlaid finite plan.
func TestOverlayLifetimeKeepsFallbackPlanPointerIndependent(t *testing.T) {
	lifetimePlanID := uuid.New()
	durationDays := 30
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	lifetimePlan := &SubscriptionPlan{
		Slug:         "premium-lifetime",
		PlanKind:     plankind.Lifetime,
		TierRank:     10,
		DurationDays: nil,
	}
	lifetimeSub := &UserSubscription{
		UserID:                 uuid.New(),
		SubscriptionPlanID:     lifetimePlanID,
		SubscriptionPlan:       lifetimePlan,
		CurrentPlanCode:        lifetimePlan.Slug,
		CurrentTierRank:        lifetimePlan.TierRank,
		CurrentPlanKind:        lifetimePlan.PlanKind,
		CurrentBenefitSnapshot: JSONDocument(`{"lifetime":true}`),
		StartedAt:              at,
		ExpiresAt:              nil,
	}

	premiumPlan := &SubscriptionPlan{
		Slug:         "premium-monthly",
		PlanKind:     plankind.Finite,
		TierRank:     20,
		DurationDays: &durationDays,
	}

	overlayEvent, err := lifetimeSub.OverlayLifetime(premiumPlan, JSONDocument(`{"overlay":true}`), "", at)
	if err != nil {
		t.Fatalf("OverlayLifetime returned error: %v", err)
	}

	// The current plan must now be the overlaid finite plan.
	if lifetimeSub.CurrentPlanCode != premiumPlan.Slug {
		t.Fatalf("current plan code = %q, want %q", lifetimeSub.CurrentPlanCode, premiumPlan.Slug)
	}

	if lifetimeSub.FallbackPlanID == nil || lifetimeSub.FallbackPlanCode == nil {
		t.Fatal("fallback plan id/code must be set after overlaying a lifetime subscription")
	}
	if lifetimeSub.FallbackPlanKind == nil || lifetimeSub.FallbackTierRank == nil {
		t.Fatal("fallback plan kind/tier rank must be set after overlaying a lifetime subscription")
	}

	// Regression: the fallback must keep the lifetime plan, not the overlaid one.
	if got := *lifetimeSub.FallbackPlanCode; got != lifetimePlan.Slug {
		t.Errorf("fallback plan code = %q, want lifetime plan code %q", got, lifetimePlan.Slug)
	}
	if got := *lifetimeSub.FallbackPlanID; got != lifetimePlanID {
		t.Errorf("fallback plan id = %s, want lifetime plan id %s", got, lifetimePlanID)
	}
	if got := *lifetimeSub.FallbackPlanCode; got == lifetimeSub.CurrentPlanCode {
		t.Errorf("fallback plan code = %q must differ from current plan code %q", got, lifetimeSub.CurrentPlanCode)
	}
	if got := *lifetimeSub.FallbackTierRank; got != lifetimePlan.TierRank {
		t.Errorf("fallback tier rank = %d, want %d", got, lifetimePlan.TierRank)
	}
	if got := *lifetimeSub.FallbackPlanKind; got != plankind.Lifetime {
		t.Errorf("fallback plan kind = %v, want %v", got, plankind.Lifetime)
	}

	// The overlay event must report a real plan change.
	if overlayEvent.ToPlanCode == nil || *overlayEvent.ToPlanCode != premiumPlan.Slug {
		t.Errorf("overlay to_plan_code = %v, want %q", overlayEvent.ToPlanCode, premiumPlan.Slug)
	}
	if overlayEvent.FromPlanCode == nil || *overlayEvent.FromPlanCode != lifetimePlan.Slug {
		t.Errorf("overlay from_plan_code = %v, want %q", overlayEvent.FromPlanCode, lifetimePlan.Slug)
	}

	// Restoring the fallback must actually put the account back on lifetime.
	expiry := *lifetimeSub.ExpiresAt
	restoreAt := expiry.AddDate(0, 0, 1)
	restoreEvent, err := lifetimeSub.RestoreFallback("", restoreAt)
	if err != nil {
		t.Fatalf("RestoreFallback returned error: %v", err)
	}
	if lifetimeSub.CurrentPlanCode != lifetimePlan.Slug {
		t.Errorf("restored plan code = %q, want lifetime plan code %q", lifetimeSub.CurrentPlanCode, lifetimePlan.Slug)
	}
	if lifetimeSub.CurrentPlanKind != plankind.Lifetime {
		t.Errorf("restored plan kind = %v, want %v", lifetimeSub.CurrentPlanKind, plankind.Lifetime)
	}
	if lifetimeSub.ExpiresAt != nil {
		t.Errorf("restored expiry = %v, want nil for lifetime", lifetimeSub.ExpiresAt)
	}

	if restoreEvent.EventType != string(subscriptioneventtype.LIFETIME_FALLBACK_RESTORED) {
		t.Errorf("restore event type = %q, want %q", restoreEvent.EventType, subscriptioneventtype.LIFETIME_FALLBACK_RESTORED)
	}
	if restoreEvent.ToPlanCode == nil {
		t.Fatal("restore event to_plan_code must not be nil")
	}
	if *restoreEvent.ToPlanCode != lifetimePlan.Slug {
		t.Errorf("restore event to_plan_code = %q, want %q", *restoreEvent.ToPlanCode, lifetimePlan.Slug)
	}
	if restoreEvent.FromPlanCode == nil {
		t.Fatal("restore event from_plan_code must not be nil")
	}
	if *restoreEvent.ToPlanCode == *restoreEvent.FromPlanCode {
		t.Errorf("restore event to_plan_code %q must differ from from_plan_code %q", *restoreEvent.ToPlanCode, *restoreEvent.FromPlanCode)
	}
}
