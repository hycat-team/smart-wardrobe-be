package grant

import (
	"testing"
	"time"

	"smart-wardrobe-be/shared/domain/constants/aipolicygrantstatus"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"
)

var windowBase = time.Date(2026, 9, 29, 8, 15, 0, 0, time.UTC)

func renewalEvent(startsAt time.Time, durationDays int) *PlanChangedEvent {
	return &PlanChangedEvent{
		EventVersion: 2,
		EventType:    subscriptioneventtype.EXTENDED,
		ToPlanCode:   "premium-monthly",
		OccurredAt:   windowBase,
		EffectiveAt:  startsAt,
		EffectiveTo:  &startsAt,
		DurationDays: durationDays,
	}
}

// TestRenewalStartsAtExpiryMinusDuration is the rule that is easiest to get
// wrong. A renewal applied on 2026-09-29 that takes the account to 2026-11-21
// bought the stretch 2026-10-22..2026-11-21. Anchoring on "now" would leave the
// account without an entitlement for the 23 days it already paid for.
func TestRenewalStartsAtExpiryMinusDuration(t *testing.T) {
	newExpiry := windowBase.AddDate(0, 0, 30)
	event := renewalEvent(newExpiry, 30)

	window := EventGrantWindow(event)
	want := newExpiry.AddDate(0, 0, -30)
	if !window.EffectiveFrom.Equal(want) {
		t.Fatalf("expected the window to start at %v, got %v", want, window.EffectiveFrom)
	}
	if window.Status != aipolicygrantstatus.Active {
		t.Fatalf("a renewal whose period already started must be active, got %s", window.Status)
	}
}

// TestRenewalStartingInTheFutureIsPending: applying a renewal early must not hand
// over an entitlement for time that has not been reached.
func TestRenewalStartingInTheFutureIsPending(t *testing.T) {
	newExpiry := windowBase.AddDate(0, 0, 60)
	event := renewalEvent(newExpiry, 30)

	window := EventGrantWindow(event)
	want := newExpiry.AddDate(0, 0, -30)
	if !window.EffectiveFrom.Equal(want) {
		t.Fatalf("expected start %v, got %v", want, window.EffectiveFrom)
	}
	if !window.EffectiveFrom.After(windowBase) {
		t.Fatalf("the window start %v is not in the future relative to %v", window.EffectiveFrom, windowBase)
	}
	if window.Status != aipolicygrantstatus.Future {
		t.Fatalf("a not-yet-started renewal must be future, got %s", window.Status)
	}
}

// TestActivationUsesTheEventTime: only a RENEWAL derives its start from the
// expiry. Every other change takes effect immediately.
func TestActivationUsesTheEventTime(t *testing.T) {
	for _, eventType := range []subscriptioneventtype.Type{
		subscriptioneventtype.ACTIVATED,
		subscriptioneventtype.UPGRADED,
		subscriptioneventtype.OVERLAID,
		subscriptioneventtype.LIFETIME_FALLBACK_RESTORED,
	} {
		expiry := windowBase.AddDate(0, 0, 30)
		event := &PlanChangedEvent{
			EventType:    eventType,
			OccurredAt:   windowBase,
			EffectiveAt:  windowBase,
			EffectiveTo:  &expiry,
			DurationDays: 30,
		}
		window := EventGrantWindow(event)
		if !window.EffectiveFrom.Equal(windowBase) {
			t.Fatalf("%s must start at the effective time, got %v", eventType, window.EffectiveFrom)
		}
		if window.Status != aipolicygrantstatus.Active {
			t.Fatalf("%s must be active, got %s", eventType, window.Status)
		}
	}
}

// TestOpenEndedPlanHasNoEnd: a lifetime or free plan produces a NULL expiry, and
// the column is nullable precisely for that.
func TestOpenEndedPlanHasNoEnd(t *testing.T) {
	event := &PlanChangedEvent{
		EventType:   subscriptioneventtype.ACTIVATED,
		OccurredAt:  windowBase,
		EffectiveAt: windowBase,
	}
	window := EventGrantWindow(event)
	if window.EffectiveTo != nil {
		t.Fatalf("an open-ended plan must have a NULL end, got %v", window.EffectiveTo)
	}
}

// TestRenewalWithoutDurationFallsBackToImmediate: a renewal carrying no duration
// has no way to place its window, so it takes effect at the event's effective
// time. Guessing a period would be worse than using the stated effective time.
func TestRenewalWithoutDurationFallsBackToImmediate(t *testing.T) {
	expiry := windowBase.AddDate(0, 0, 30)
	event := renewalEvent(expiry, 0)

	window := EventGrantWindow(event)
	if !window.EffectiveFrom.Equal(event.EffectiveAt) {
		t.Fatalf("expected the effective time %v, got %v", event.EffectiveAt, window.EffectiveFrom)
	}
	if window.Status != aipolicygrantstatus.Active {
		t.Fatalf("expected active, got %s", window.Status)
	}
}

// TestChangeTypeRouting pins which types close prior grants, create new ones and
// drop pending rows. The old code routed everything except a hard-coded pair
// through a default branch, which is what made an auto-renew toggle revoke and
// re-grant entitlements.
func TestChangeTypeRouting(t *testing.T) {
	closes := map[subscriptioneventtype.Type]bool{
		subscriptioneventtype.ACTIVATED:                       true,
		subscriptioneventtype.UPGRADED:                        true,
		subscriptioneventtype.OVERLAID:                        true,
		subscriptioneventtype.LIFETIME_FALLBACK_RESTORED:      true,
		subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE: true,
		subscriptioneventtype.EXTENDED:                        false,
		subscriptioneventtype.AUTO_RENEW_UPDATED:              false,
	}
	for eventType, want := range closes {
		if got := ClosesPriorGrants(eventType); got != want {
			t.Fatalf("ClosesPriorGrants(%s) = %v, want %v", eventType, got, want)
		}
	}

	creates := map[subscriptioneventtype.Type]bool{
		subscriptioneventtype.ACTIVATED:                  true,
		subscriptioneventtype.EXTENDED:                   true,
		subscriptioneventtype.UPGRADED:                   true,
		subscriptioneventtype.OVERLAID:                   true,
		subscriptioneventtype.LIFETIME_FALLBACK_RESTORED: true,
		// 024 T188: a downgrade creates one, exactly like a paid plan change.
		// ClosesPriorGrants above still closes the Premium row, so the pair
		// is "close then create" and never "close only" — closing without
		// creating left the account with no row to resolve against, which is
		// the data loss the dev-stack verification measured.
		subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE: true,
		// Auto-renew changes no entitlement whatsoever. It also cannot produce a
		// missing-grant report, because the account already holds the row written
		// by the activation that gave it the plan.
		subscriptioneventtype.AUTO_RENEW_UPDATED: false,
	}
	for eventType, want := range creates {
		if got := CreatesNewGrant(eventType); got != want {
			t.Fatalf("CreatesNewGrant(%s) = %v, want %v", eventType, got, want)
		}
	}

	// A pending row survives a renewal, because the renewal may be what it was
	// waiting for. Every other change invalidates it.
	for eventType, want := range map[subscriptioneventtype.Type]bool{
		subscriptioneventtype.EXTENDED:           false,
		subscriptioneventtype.ACTIVATED:          true,
		subscriptioneventtype.UPGRADED:           true,
		subscriptioneventtype.OVERLAID:           true,
		subscriptioneventtype.AUTO_RENEW_UPDATED: true,
	} {
		if got := DeletesPendingGrants(eventType); got != want {
			t.Fatalf("DeletesPendingGrants(%s) = %v, want %v", eventType, got, want)
		}
	}
}
