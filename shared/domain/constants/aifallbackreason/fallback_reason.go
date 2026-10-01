// Package aifallbackreason holds the reason codes recorded when the AI service
// has to degrade instead of failing (024 FR-008, FR-010).
//
// The reason code is what makes a degradation diagnosable. Without one, "the
// system fell back" cannot be separated from "the model provider was down", and
// a counter that mixes both tells an operator nothing about which knob to turn.
package aifallbackreason

// Reason identifies why a request could not run on its intended route.
type Reason string

const (
	// Tier1DefaultPolicy — the account has no policy grant row, so the
	// configured default cost policy was applied in its place. Still bounded:
	// the request goes through the reservation-first ledger of constitution §VI,
	// with grant_id NULL, so the anti-overspend control stays in force.
	Tier1DefaultPolicy Reason = "tier_1_default_policy"

	// Tier2ObserveOnly — tier 1 could not be used at all (the configured policy
	// code does not exist, the policy is inactive, or the operation is missing
	// from it, or an observe_only policy has no per-day attempt cap). The
	// request runs on the free model with no hard cap and zero paid attempts
	// allowed. Always accompanied by a warning: this is degradation of a paid
	// feature, not a normal outcome.
	Tier2ObserveOnly Reason = "tier_2_observe_only"

	// --- Extendable codes, recorded when a degradation is detected ---

	// DefaultPolicyNotFound — ai.default_cost_policy_code names a code that is
	// not in aiexecutor.ai_cost_policies.
	DefaultPolicyNotFound Reason = "default_policy_not_found"
	// DefaultPolicyInactive — the default policy row exists but is_active is
	// false.
	DefaultPolicyInactive Reason = "default_policy_inactive"
	// OperationNotConfigured — the resolved policy has no enabled row for the
	// requested operation.
	OperationNotConfigured Reason = "operation_not_configured"
	// ObserveOnlyAttemptCapMissing — an observe_only policy has a NULL
	// max_paid_attempts_per_day. In observe_only there is no hard cost cap, so
	// that column is the only bound; treating NULL as "no limit" would let the
	// free policy spend without any ceiling and no error would ever surface.
	ObserveOnlyAttemptCapMissing Reason = "observe_only_attempt_cap_missing"
	// PolicyLookupFailed — the database could not be read. Transient, so the
	// request degrades rather than failing.
	PolicyLookupFailed Reason = "policy_lookup_failed"
)
