package campaign

// SkipReason is the code recorded for every campaign message that is not
// granted (025 T044, FR-061).
//
// It exists because "the account was skipped" is not an actionable answer on its
// own: a campaign that never granted anything and a campaign that exhausted its
// budget look identical from the outside, and the second one is a bug report
// while the first one is the design working. Every branch of the decision table in
// `data-model.md` §6 therefore names itself, and the codes below are those names.
//
// The codes are stable strings rather than an iota because they are read back by
// the operator look-up (FR-065) and written into logs that outlive this binary
// (FR-062). Renumbering them later would silently change what an old log line
// means.
type SkipReason string

const (
	// SkipCampaignNotConfigured: no campaign is declared at all (condition 1).
	// The mechanism exists but was never switched on, which is different from a
	// campaign that ran and had nothing left.
	SkipCampaignNotConfigured SkipReason = "campaign_not_configured"
	// SkipCampaignNotStarted: every declared campaign opens later than now
	// (condition 2).
	SkipCampaignNotStarted SkipReason = "campaign_not_started"
	// SkipCampaignSleeping: every declared campaign has passed its declared end
	// (condition 2). "Sleeping" is the word the specification uses for a campaign
	// that is over rather than one that is out of stock, and keeping them apart is
	// what lets an operator tell "we closed it" from "we ran out of giveaways".
	SkipCampaignSleeping SkipReason = "campaign_sleeping"
	// SkipCreatedByAdmin: the account was created through the administrator
	// endpoint (condition 3, FR-050/FR-091).
	//
	// Deliberately its own code rather than a flavour of "not eligible": the
	// account is a perfectly good customer, the campaign simply does not apply to
	// it. An operator reading a wall of `not_eligible` would look for a
	// configuration error, and there is none.
	SkipCreatedByAdmin SkipReason = "created_by_admin"
	// SkipAlreadyClaimed: the account already holds a claim, from a redelivered
	// message or from an earlier campaign (condition 4, FR-012/FR-025).
	SkipAlreadyClaimed SkipReason = "already_claimed"
	// SkipAlreadyOnPlan: the account is already on exactly this campaign's plan
	// and the term has not run out (condition 5, FR-018). Granting would STACK
	// another month onto it, which is the specific harm FR-018 forbids.
	SkipAlreadyOnPlan SkipReason = "already_on_plan"
	// SkipPlanNotSettled: the account was just demoted to the free plan while the
	// previous plan's cleanup has not finished (condition 6).
	//
	// The signal is a free-plan subscription that still carries a fallback plan.
	// Granting over that would produce a row describing two plans at once, and the
	// next renewal or downgrade would read whichever it happened to look at.
	SkipPlanNotSettled SkipReason = "plan_not_settled"
	// SkipOutsideWindow: the account's creation instant lies outside the chosen
	// campaign's [starts_at, ends_at) window (FR-017). The window is evaluated
	// against `registered_at`, never against "now" — see `data-model.md` §6.1.
	SkipOutsideWindow SkipReason = "outside_window"
	// SkipQuotaMain: the main budget is full and the account was created after the
	// priority watermark, so even the reserve cannot help (condition 8, FR-031).
	SkipQuotaMain SkipReason = "quota_main"
	// SkipQuotaCompensation: the reserve budget is full as well (condition 8,
	// FR-032). The reserve is a CEILING, so a campaign that ends with reserve left
	// over is behaving correctly, not incorrectly.
	SkipQuotaCompensation SkipReason = "quota_compensation"
	// SkipQuotaAnomaly: the main budget is full but no priority watermark exists.
	//
	// That combination should be impossible: the watermark is written by the very
	// grant that fills the main budget, under a partial unique index that allows
	// only one. Seeing it means the ledger was edited by hand or a campaign was
	// reloaded with a different `quota`. Rather than guess which accounts the
	// reserve belongs to, the message is skipped and the anomaly is reported
	// (`data-model.md` §6.2).
	SkipQuotaAnomaly SkipReason = "quota_anomaly"
	// SkipPlanUnavailable: the campaign's `plan_slug` names a plan that does not
	// exist or is not active (FR-010). Loading configuration cannot check this -
	// it must not reach into the database - so it is caught here, where the plan
	// row is read anyway.
	SkipPlanUnavailable SkipReason = "plan_unavailable"
)
