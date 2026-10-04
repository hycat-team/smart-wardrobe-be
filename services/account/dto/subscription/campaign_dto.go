package dto

import (
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// 025 T077: the read-only operator look-up (FR-065 to FR-072).
//
// SHAPE OF THIS FILE. Three response types, one per question the contract asks:
// what does a campaign look like, who was granted a plan, and what happened to
// this one account. There is no write request type anywhere, and that absence is
// the requirement rather than an oversight — FR-069 makes the whole feature
// read-only, so a request struct here would be a design that contradicts it.
//
// JSON tags are camelCase (constitution, "JSON Naming for API DTOs"), and every
// timestamp is a pointer where "not set" is a meaningful answer. The three that
// matter are `watermarkAt` (nil means the main budget has never filled, which is
// a different campaign state from "filled at the zero instant"),
// `endsAt` (nil means open-ended, FR-009) and `lastSweepAt` (nil means no sweep
// has ever completed, FR-059).

// CampaignSummaryRes is one campaign in GET /admin/campaigns
// (contracts/campaign-query-api.md §3.1).
//
// EVERY COUNT HERE IS DERIVED AT QUERY TIME and none of it is stored
// (data-model.md §4.3). A stored count would have to be synchronised by hand and
// would be wrong the moment a grant committed without touching it — and an
// operator looking at a wrong budget number is worse than one looking at no
// number, because they would act on it.
type CampaignSummaryRes struct {
	// Code identifies the campaign and is the same value the grant path writes
	// into `campaign_claims.campaign_code`, so an operator can join this response
	// against the ledger themselves.
	Code string `json:"code"`
	// PlanSlug is the plan the campaign grants. It is the campaign's reference to
	// the plan, never a copy of the plan's attributes (data-model.md §8.2): a
	// copied name would go stale the moment the catalogue is edited.
	PlanSlug string `json:"planSlug"`
	// PlanName is read from the plan row at query time. It is absent for a
	// campaign whose `plan_slug` names nothing — which is FR-010's `plan_slug`
	// check refusing at runtime rather than at startup, and the operator sees the
	// empty name plus a `plan_unavailable` reason on the account look-up.
	PlanName string `json:"planName"`
	// Quota is the main budget: the number of grants that fill it and freeze the
	// priority watermark.
	Quota int `json:"quota"`
	// Reserve is the compensation budget — a CEILING, not a target (FR-032), so a
	// finished campaign may legitimately hold fewer grants than this.
	Reserve int `json:"reserve"`
	// GrantedMainCount is `COUNT(*)` filtered on `grant_bucket = 'main'`.
	//
	// Read from `grant_bucket` and NEVER from `NOT fills_main_quota`: that flag
	// marks exactly one row, so its negation equals "total minus one", which
	// folds the reserve in as well (research.md R2b).
	GrantedMainCount int64 `json:"grantedMainCount"`
	// GrantedCompensationCount is `COUNT(*)` filtered on
	// `grant_bucket = 'reserve'`. It is an INDEPENDENT number, not the remainder
	// of the total.
	GrantedCompensationCount int64 `json:"grantedCompensationCount"`
	// GrantedTotalCount is the sum of the two above.
	//
	// Present because without it a client cannot check the invariant that matters:
	// `grantedTotalCount <= hardCap` (FR-022). Deriving the total on the client
	// would mean adding two fields, and a client that added them wrongly would
	// report a breach that never happened.
	GrantedTotalCount int64 `json:"grantedTotalCount"`
	// RemainingMain is `max(0, quota - grantedMainCount)`.
	RemainingMain int64 `json:"remainingMain"`
	// RemainingCompensation is `max(0, reserve - grantedCompensationCount)`.
	RemainingCompensation int64 `json:"remainingCompensation"`
	// HardCap is `quota + reserve`: the ceiling no campaign may cross (FR-022).
	HardCap int64 `json:"hardCap"`
	// StartsAt is the instant the campaign opens, as configured.
	StartsAt time.Time `json:"startsAt"`
	// EndsAt is nil for an open-ended campaign (FR-009) — and nil is a DIFFERENT
	// answer from a zero time, so it is a pointer.
	EndsAt *time.Time `json:"endsAt"`
	// WatermarkAt is the `registered_at` of the claim that filled the main budget,
	// or nil when that has not happened. It is the instant every later priority
	// comparison uses (FR-030).
	WatermarkAt *time.Time `json:"watermarkAt"`
	// Status is the derived lifecycle state: `not_started`, `running`,
	// `compensation`, `exhausted` or `expired`. Derived at query time and never
	// stored (data-model.md §5) — a stored status would need synchronising with
	// the ledger along a second path, and two paths are where duplicate grants
	// come from.
	Status string `json:"status"`
	// Degraded reports that the most recent sweep FAILED for this campaign
	// (FR-059). Read from `campaign_sweep_runs`, never from the sweep worker's
	// memory: the service runs several replicas, so an in-memory flag read by one
	// replica says nothing about a failure another replica produced.
	Degraded bool `json:"degraded"`
	// DegradedReason is the most recent failure text, so an operator does not have
	// to correlate a timestamp with a log search. Empty when healthy.
	DegradedReason string `json:"degradedReason,omitempty"`
	// LastSweepAt is when a sweep last COMPLETED. A sweep failing every fifteen
	// minutes leaves it where the last success put it, which is the alarm.
	LastSweepAt *time.Time `json:"lastSweepAt"`
}

// 026 T029 (FR-052): the first WRITE request — open a campaign.
//
// WHY THIS FILE NOW HOLDS A WRITE REQUEST. The header above says there is none, and
// that was true for 025: FR-069 made the whole phase read-only. 026 is the feature
// that ends that, because "open a campaign without a deploy" (spec.md §2.2) is only
// possible through an API. The comment is corrected rather than deleted so the
// change is visible: the read-only guarantee moved from the whole feature to
// `CampaignAdminHandler`, which still holds an interface with no write method on it.
//
// WHY `quota` AND `planSlug` ARE POINTERS AND `reserve` IS NOT. FR-054 separates
// "thiếu trường bắt buộc" (400) from "`quota = 0`" and "`planSlug` rỗng" (422), and
// those two cannot be told apart on a plain `int`/`string`: both arrive as the zero
// value. A pointer is what distinguishes "the field was absent" from "the field was
// sent with an empty value", which is the only way to answer 400 and 422 correctly
// without a second rule that guesses.
//
// `reserve` needs no pointer because it is OPTIONAL and its absent value has a
// meaning of its own — 0 (data-model.md §2.1 `DEFAULT 0`) — so "absent" and "sent 0"
// produce the same row and may share one representation. It is still a pointer so
// that a negative value reaches the validator instead of being silently dropped.
//
// WHY `startsAt`/`endsAt` ARE STRINGS. Two reasons, both from FR-013. A
// `time.Time` field would hand parsing to Gin's decoder, which accepts a format the
// operator's documentation does not show, so the accepted inputs would differ from
// the documented ones with nothing to say so (the same argument
// `campaignClaimsQuery.from`/`.to` makes). And the zone-less value FR-013 rejects
// cannot be represented by a Go `time.Time` at all — it would be read in the SERVER's
// zone, seven hours off in production, and stored as an instant nobody typed. Parsing
// the string here keeps "must carry a zone" a rule the code can state.
type CampaignCreateReq struct {
	// Code identifies the campaign and is immutable afterwards (FR-024).
	//
	// Restricted to `^[a-z0-9][a-z0-9-]{2,63}$`, matching `chk_campaigns_code_format`
	// (FR-011). A malformed code is a 400 — a broken shape, not a wrong meaning
	// (FR-054) — and the alphabet is restricted because the code reaches an audit
	// payload and possibly a queue or file name.
	Code string `json:"code" binding:"required"`
	// PlanSlug is the plan granted to an eligible account. Required (FR-014).
	//
	// Whether the plan EXISTS and is ACTIVE is checked at RUN time, not here and not
	// at startup (FR-003, FR-014): a wrong slug must not be able to crash a service,
	// and it makes that one campaign grant nothing rather than failing a write.
	PlanSlug *string `json:"planSlug" binding:"required"`
	// Quota is the main budget. `>= 1` (FR-012); zero is 422, not 400, because the
	// field was sent.
	Quota *int `json:"quota" binding:"required"`
	// Reserve is the compensation CEILING, not a target. Optional, `>= 0`, default 0.
	Reserve *int `json:"reserve"`
	// StartsAt is the instant the campaign opens, RFC3339 WITH an explicit zone.
	StartsAt string `json:"startsAt" binding:"required"`
	// EndsAt is the instant it closes, RFC3339 with a zone, or absent/null for an
	// open-ended campaign. When present it must be strictly AFTER StartsAt (FR-013).
	//
	// A pointer, and that is what makes `"endsAt": null` differ from `"endsAt": ""`:
	// the first is an open-ended campaign, the second is a malformed timestamp.
	EndsAt *string `json:"endsAt"`
}

// CampaignClaimRes is one granted account in
// GET /admin/campaigns/:code/claims (contracts/campaign-query-api.md §3.3).
type CampaignClaimRes struct {
	// UserID identifies the account. It is the same identifier the customer
	// quotes when asking.
	UserID uuid.UUID `json:"userId"`
	// Username is joined through `IUserBriefContract.GetByIDs`, never read from
	// the account table directly — the campaign module does not own
	// `identity.users` (constitution §II). Empty when the account has since been
	// deleted, because the claim row survives its subject: `user_id` cascades,
	// but a soft-deleted account is still readable.
	Username string `json:"username"`
	// PlanSlug is the plan that was actually granted, copied onto the claim at
	// grant time. It is a copy rather than a lookup precisely so that a later
	// catalogue edit cannot rewrite the history of what this account received.
	PlanSlug string `json:"planSlug"`
	// ExpiresAt is the granted term: `granted_at + duration_days` of the plan
	// (FR-034, FR-019).
	//
	// NOT `registered_at + duration_days`. Anchoring on registration cuts the
	// term short by however long the message took, and in the late-delivery case
	// FR-078 exists for that is most of the gift.
	ExpiresAt time.Time `json:"expiresAt"`
	// GrantedAt is the instant the grant was written — the filter axis of FR-067.
	GrantedAt time.Time `json:"grantedAt"`
	// RegisteredAt is the account's creation instant, snapshotted at grant time
	// so an operator can see WHY this account ranked where it did, without the
	// campaign module reading the account table.
	RegisteredAt time.Time `json:"registeredAt"`
	// Source says which part of the budget the grant came from.
	//
	// Its values are EXACTLY `"main"` and `"reserve"` — the values of the
	// `grant_bucket` column. Not `"main_quota"` and not `"compensation"`: those
	// are state names, and putting them here would create a second vocabulary for
	// one idea, which is how a client ends up switching on a string that only one
	// endpoint produces.
	Source string `json:"source"`
}

// CampaignAccountEligibilityRes answers "what happened to this one account" in
// GET /admin/campaigns/:code/eligibility/:userId
// (contracts/campaign-query-api.md §3.4).
//
// This is the endpoint the whole phase exists for. FR-068 requires the three
// states to be DISTINGUISHABLE, and the reason they must be is that the answer
// changes what an operator tells a customer: "you were given the plan" and "you
// qualified but the giveaways ran out" and "you were never in the group" are
// three different conversations. A response that returned an empty list, or the
// same answer for all three, would fail FR-068 while looking perfectly healthy.
type CampaignAccountEligibilityRes struct {
	// CampaignCode echoes the campaign asked about, so a client holding several
	// responses cannot mix them up.
	CampaignCode string `json:"campaignCode"`
	// UserID echoes the account asked about.
	UserID uuid.UUID `json:"userId"`
	// Eligibility is the coarse verdict an operator switches on. One of exactly
	// SIX values (contracts/campaign-query-api.md §3.4): `granted`,
	// `eligible_pending`, `eligible_but_exhausted`, `not_eligible`,
	// `already_claimed_other_campaign`, `created_by_admin`.
	//
	// It is deliberately COARSE. The diagnostic detail is `reason`, and splitting
	// one idea across two fields is what keeps each of them small enough to be
	// exhaustive: `eligibility` for a client to branch on, `reason` for a human to
	// act on.
	Eligibility string `json:"eligibility"`
	// Granted is true only for `granted`. It is repeated alongside `eligibility`
	// so a client does not have to know that one specific string is the granted
	// one — and so "granted" and "eligibility" can never disagree, because they
	// are written by the same branch.
	Granted bool `json:"granted"`
	// Reason is the stable diagnostic code, one of the twelve in
	// contracts/campaign-query-api.md §3.5, which is the same set as
	// `campaign.SkipReason`. Empty when there is nothing to diagnose, which is
	// exactly two cases: the account was granted, and the account is
	// `eligible_pending` — a grant that has not happened yet has no fault to
	// name, and inventing one would point an operator at a budget that is not
	// the problem.
	//
	// Stable STRINGS rather than an iota, because this value is read back by the
	// operator look-up and written into logs that outlive the binary (FR-062) —
	// renumbering later would silently change what an old log line means.
	Reason string `json:"reason,omitempty"`
	// WatermarkAt is the campaign's priority watermark, so an operator can see
	// the boundary an account was measured against. Nil when not yet set.
	WatermarkAt *time.Time `json:"watermarkAt"`
	// UserRegisteredAt is the account's creation instant. It is what the priority
	// rule compares (FR-030/FR-031), and an operator asking "why did they miss
	// out?" needs the number to check that comparison themselves.
	UserRegisteredAt *time.Time `json:"userRegisteredAt"`
}

// CampaignAccountEligibility constants.
//
// Declared here rather than only in the use case because they are part of the
// API CONTRACT: a client switches on these strings, so they belong in the same
// package as the JSON tags that carry them.
const (
	// CampaignEligibilityGranted: the account holds a claim from THIS campaign.
	CampaignEligibilityGranted = "granted"
	// CampaignEligibilityEligiblePending: the account meets every condition, the
	// campaign still has quota, and the grant has NOT happened yet — the message
	// is still in flight, or the catch-up sweep has not reached it (FR-079).
	//
	// It is its own value rather than a flavour of
	// `eligible_but_exhausted` because reading `exhausted` tells a customer "we
	// have run out of giveaways", and that is false in BOTH senses here: the
	// campaign has quota left, and this customer's grant is on its way. It is the
	// single most common question on the phone — "I just signed up, where is my
	// plan?" — and the answer has to say "qualified, the system is still
	// working on it".
	//
	// `reason` is the EMPTY STRING, deliberately: every other verdict names a
	// condition to act on, and there is no code that says "in flight".
	CampaignEligibilityEligiblePending = "eligible_pending"
	// CampaignEligibilityEligibleButExhausted: the account meets every condition
	// but was not granted — the budget is full, or it registered after the
	// watermark (FR-033). This is the middle state FR-068 exists to keep
	// separate from `not_eligible`, and it always carries a reason:
	// `quota_main` or `quota_compensation` (or `quota_anomaly`).
	CampaignEligibilityEligibleButExhausted = "eligible_but_exhausted"
	// CampaignEligibilityNotEligible: the account meets no condition of this
	// campaign's rules.
	CampaignEligibilityNotEligible = "not_eligible"
	// CampaignEligibilityAlreadyClaimedOtherCampaign: the account already holds
	// a claim, written by an EARLIER campaign (FR-012). It is its own value
	// because the account is a perfectly good customer who simply arrived too
	// late — the same argument `created_by_admin` makes.
	CampaignEligibilityAlreadyClaimedOtherCampaign = "already_claimed_other_campaign"
	// CampaignEligibilityCreatedByAdmin: the account was created through the
	// administrator endpoint, so the campaign does not apply to it (FR-091).
	CampaignEligibilityCreatedByAdmin = "created_by_admin"
)

// Campaign status constants (contracts/campaign-query-api.md §3.2).
//
// Five values, derived at query time from the configuration plus the claim counts
// (data-model.md §5).
const (
	// CampaignStatusNotStarted: `now` is before `starts_at`.
	CampaignStatusNotStarted = "not_started"
	// CampaignStatusRunning: inside the window, the main budget is not full, and
	// no priority watermark has been frozen.
	CampaignStatusRunning = "running"
	// CampaignStatusCompensation: the watermark is frozen and the reserve still
	// has room. It is a DISTINCT state from `running` because it is exactly the
	// one in which the reserve — the budget that exists for the accounts the main
	// budget turned away — is still usable.
	CampaignStatusCompensation = "compensation"
	// CampaignStatusExhausted: the hard ceiling `quota + reserve` is reached, so
	// the campaign goes to sleep on its own (FR-013).
	CampaignStatusExhausted = "exhausted"
	// CampaignStatusExpired: `now` is past `ends_at`, so the campaign goes to
	// sleep on its own (FR-013).
	CampaignStatusExpired = "expired"
)

// CampaignClaimSourceMain and CampaignClaimSourceReserve are the two values of
// `CampaignClaimRes.Source`.
//
// They are the `grant_bucket` column values verbatim, which is why they are not
// typed as `entities.GrantBucket`: the response type must not depend on the
// persistence vocabulary, so that a later rename of the column values cannot
// silently change a published API.
const (
	CampaignClaimSourceMain    = string(entities.GrantBucketMain)
	CampaignClaimSourceReserve = string(entities.GrantBucketReserve)
)
