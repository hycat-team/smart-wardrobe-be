package grant

import (
	"time"

	"smart-wardrobe-be/shared/domain/constants/aipolicygrantstatus"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"
)

// GrantWindow is the validity window and status of the grant a change produces.
type GrantWindow struct {
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	Status        aipolicygrantstatus.AIPolicyGrantStatus
}

// ComputeGrantWindow derives the window for an incoming plan change.
//
// This rule lived in projectAIPolicyGrant, inside the Account service, writing
// to a table it did not own (constitution §I). It is moved here verbatim in
// behaviour so the AI service can become the single legitimate owner, and it is
// kept as a pure function so it can be tested without a database.
//
// The one rule that is easy to get wrong is the renewal start. A renewal may be
// applied early or late, and the customer paid for a specific stretch of time, so
// the window has to start at "new expiry minus duration" rather than at "now".
// Anchoring on "now" would leave a gap between the old period and the new one —
// an account with no entitlement for the days it already paid for.
func ComputeGrantWindow(event *PlanChangedEvent) GrantWindow {
	// A downgrade or an auto-renew toggle carries no window of its own; those
	// change types never reach this function, and returning the effective time
	// keeps the caller honest if one does.
	window := GrantWindow{
		EffectiveFrom: event.EffectiveAt,
		EffectiveTo:   event.EffectiveTo,
		Status:        aipolicygrantstatus.Active,
	}
	if window.EffectiveTo == nil {
		return window
	}
	// 024 T168: the parser now refuses a renewal whose durationDays is absent or
	// non-positive, so this branch is a defensive default rather than the renewal
	// rule. It still must NOT be reached by a real renewal event: falling through to
	// the ordinary branch would set the start to EffectiveAt and the status to Active,
	// opening the entitlement at the moment the event arrived instead of at "new expiry
	// minus duration" — early access to a period that has not been paid for, which is
	// what the note above says must not happen.
	if event.EventType != subscriptioneventtype.EXTENDED || event.DurationDays <= 0 {
		return window
	}

	// Renewal: the paid stretch is the last DurationDays of the new period.
	start := event.EffectiveTo.AddDate(0, 0, -event.DurationDays)
	window.EffectiveFrom = start
	// If that stretch has not started yet, the grant is future, not active:
	// handing it over now would grant a period that has not been paid for.
	if start.After(event.OccurredAt) {
		window.Status = aipolicygrantstatus.Future
	}
	return window
}

// ClosesPriorGrants reports whether a change type invalidates the grants that
// currently exist, as opposed to only moving a window boundary.
func ClosesPriorGrants(eventType subscriptioneventtype.Type) bool {
	switch eventType {
	case subscriptioneventtype.ACTIVATED,
		subscriptioneventtype.UPGRADED,
		subscriptioneventtype.OVERLAID,
		subscriptioneventtype.LIFETIME_FALLBACK_RESTORED,
		subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE:
		return true
	default:
		// EXTENDED moves the boundary instead of closing the prior grant, and
		// AUTO_RENEW_UPDATED changes nothing at all.
		return false
	}
}

// CreatesNewGrant reports whether a change type produces a new grant row.
//
// 024 T188: SUBSCRIPTION_DOWNGRADED_TO_FREE now creates one. The comment this
// replaces said the opposite — that a downgrade only revokes because "the free
// policy is applied by the fallback resolver instead" — and that reasoning was
// wrong about which file decides what a plan is entitled to.
//
// A grant row is the single source of truth for an entitlement, and the rule is
// applied the same way for a paid plan and a free one. The free plan's limits live
// in the free-default cost policy, and a frozen snapshot of that policy is
// precisely what a grant carries — including the per-day attempt cap the customer
// actually experiences. Leaving the account with no row meant its only source of
// limits was a fallback, which is a safety net rather than a definition, so the
// downgrade produced an account that satisfied "has a plan" and had nothing to
// resolve against. Measured on the dev stack, a downgraded account was reported
// by check 1 of 07-verify_policy_grants.sql until an operator ran the repair
// endpoint by hand: data loss produced by a state the design considered correct.
//
// The fallback resolver still exists and is still correct, but only for the case
// it was actually built for: an account with NO grant row at all — typically data
// that predates this feature. It is not where the free plan's quotas are defined.
//
// AUTO_RENEW_UPDATED stays false, and for a different reason: toggling auto-renew
// changes no entitlement whatsoever, so there is nothing new to write. It also
// cannot produce a missing-grant report, because an account that can toggle
// auto-renew already holds a row from the activation that gave it the plan.
// Closing the old row without creating a new one would be exactly the data loss
// described above, which is why a downgrade takes the create path (and
// ClosesPriorGrants still closes the Premium row) while this does not.
func CreatesNewGrant(eventType subscriptioneventtype.Type) bool {
	if eventType == subscriptioneventtype.AUTO_RENEW_UPDATED {
		return false
	}
	return true
}

// DeletesPendingGrants reports whether pending (future) rows must be removed
// rather than closed. A closed row would still satisfy a query that filters on
// status, so pending rows are deleted outright.
func DeletesPendingGrants(eventType subscriptioneventtype.Type) bool {
	return eventType != subscriptioneventtype.EXTENDED
}
