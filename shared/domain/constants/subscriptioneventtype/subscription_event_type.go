// Package subscriptioneventtype holds the closed set of subscription change
// types, shared by the publisher (Account) and the consumer (AI).
//
// Before this package existed, each side carried its own list. They did not
// match: the consumer branched on "RENEWED" and "EXPIRED", which nobody ever
// published, while ignoring "EXTENDED" (which every renewal does publish) and
// creating a grant for "AUTO_RENEW_UPDATED" (which changes nothing). That
// mismatch is the root cause of feature 024: plan changes produced no AI policy
// grant, silently, with no error anywhere.
//
// Constitution §II allows importing domain/constants across modules, and
// docs/system-design/design-pattern.md has an explicit exception for domain
// constants, so the shared enum belongs in shared/ rather than being copied —
// copying is what produced the bug.
package subscriptioneventtype

// Type is one subscription change type. The string values are the wire format
// on subscription.plan.changed, so they must never be changed once published.
type Type string

const (
	// ACTIVATED — a plan became active (PayOS, wallet, administrator).
	ACTIVATED Type = "ACTIVATED"
	// EXTENDED — a renewal. The period may start in the future, in which case
	// the consumer records the new grant as `future` instead of `active`.
	EXTENDED Type = "EXTENDED"
	// UPGRADED — moved to a higher plan.
	UPGRADED Type = "UPGRADED"
	// OVERLAID — a limited plan overwrote a lifetime plan.
	OVERLAID Type = "OVERLAID"
	// LIFETIME_FALLBACK_RESTORED — the lifetime plan became effective again.
	LIFETIME_FALLBACK_RESTORED Type = "LIFETIME_FALLBACK_RESTORED"
	// SUBSCRIPTION_DOWNGRADED_TO_FREE — expired, or revoked by an
	// administrator. The consumer closes every active grant.
	SUBSCRIPTION_DOWNGRADED_TO_FREE Type = "SUBSCRIPTION_DOWNGRADED_TO_FREE"
	// AUTO_RENEW_UPDATED — auto-renew toggled. The plan did NOT change, so the
	// consumer does nothing. It is still published, because the contract
	// requires an event for every plan-relevant change and the receiver — not
	// the sender — decides what to do with it.
	AUTO_RENEW_UPDATED Type = "AUTO_RENEW_UPDATED"
)

// All lists every valid type. The consumer rejects anything outside this set
// rather than falling through to a default branch: an unknown type must never
// be guessed at, because guessing wrong writes a grant that should not exist.
//
// "RENEWED" and "EXPIRED" are deliberately absent. The consumer used to handle
// them, but no code path ever published them, so that handling was dead.
var All = map[Type]bool{
	ACTIVATED:                       true,
	EXTENDED:                        true,
	UPGRADED:                        true,
	OVERLAID:                        true,
	LIFETIME_FALLBACK_RESTORED:      true,
	SUBSCRIPTION_DOWNGRADED_TO_FREE: true,
	AUTO_RENEW_UPDATED:              true,
}

// IsValid reports whether t is one of the seven known change types.
func IsValid(t Type) bool {
	return All[t]
}
