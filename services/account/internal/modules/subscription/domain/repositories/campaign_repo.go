package repositories

import (
	"context"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
)

// ICampaignRepository is the campaign registry itself (026, FR-001/FR-005).
//
// It is the ONLY source of which campaigns exist. The `campaigns` YAML block that
// used to answer that question is deleted from both config files, so every path
// that used to read `cfg.Campaigns` now reads this — at RUN TIME, never at load
// time, which is what keeps configuration loading free of database access
// (FR-003, constitution §I/§VIII).
//
// WHY THERE IS NO `Delete` METHOD, and not merely no caller for one (FR-005,
// FR-034). A delete path would have to be reachable from somewhere, and the day
// somebody calls it the ledger is orphaned: `campaign_claims.campaign_code` and
// `campaign_sweep_runs.campaign_code` are historical text keys with no foreign
// key (FR-091), so the rows would survive while the campaign they describe
// vanished, and the operator look-up would report a budget nobody can spend or
// revoke. The absence is the guarantee, so it is structural rather than a
// convention: there is nothing to delete through.
type ICampaignRepository interface {
	// ListAll returns EVERY row of the registry, newest-starting first with
	// `code DESC` as the deterministic tie-break (FR-040, research.md R6).
	//
	// "Every" is what the operator list endpoint needs and is the reason this is a
	// separate method from `ListOpen`: a closed campaign must still be VISIBLE on
	// the admin list — FR-032 adds `status = closed` precisely so an operator can
	// see that a campaign was closed and when. A list that hid closed campaigns
	// would make the sixth status unreachable.
	ListAll(ctx context.Context) ([]*entities.Campaign, error)

	// ListOpen returns the campaigns that are NOT FORCE-CLOSED, newest-starting
	// first with `code DESC` as the deterministic tie-break (FR-040, research.md
	// R6).
	//
	// THE WINDOW IS NOT PART OF THIS PREDICATE, and the placement is deliberate
	// rather than an oversight. `data-model.md` §4.1 writes the "still awake"
	// query as `closed_at IS NULL AND starts_at <= :now AND (ends_at IS NULL OR
	// :now < ends_at)`, and `campaignEvaluator.isAwake` is specified to be the SAME
	// predicate in Go (data-model.md §4.3). The two cannot both be the one that
	// runs: if the SQL already dropped every campaign outside its window, the
	// evaluator would receive an empty list for a campaign that is merely asleep,
	// and condition 2 would report `campaign_not_configured` where the
	// specification says a closed campaign must report `campaign_sleeping`
	// (spec.md Câu 4, quickstart V2 step 5) — and FR-042/FR-043 require 025's
	// conditions to keep their existing behaviour.
	//
	// So the SQL keeps the ONE term no amount of application logic can recover —
	// a force-closed campaign is never a candidate — and `isAwake` applies the
	// window. The SELECTED campaign set is identical either way; only the reason
	// code an account observes when nothing is awake stays meaningful.
	//
	// The cost is one comparison in Go on a handful of rows: the registry is a few
	// rows for the whole product lifetime, so there is no index to save (plan.md,
	// "Số hàng trong sổ: vài hàng").
	//
	// ONE query, no `LIMIT`, and no `registered_at` term (FR-040). The
	// registration-instant half of the window belongs to the pure evaluator, so
	// conditions 1-7 of 025 stay testable without a database (spec.md Câu 6).
	ListOpen(ctx context.Context) ([]*entities.Campaign, error)

	// GetByCode returns one campaign by code, or nil when the code does not exist.
	//
	// `nil, nil` rather than an error for "absent" because absent is a business
	// answer the caller turns into 404 (FR-054), not a fault — the same convention
	// `ISubscriptionPlanRepository.GetBySlug` uses.
	GetByCode(ctx context.Context, campaignCode string) (*entities.Campaign, error)

	// GetByCodeForShare re-reads the campaign row under a SHARE lock: the row
	// cannot be updated or deleted by anyone else until this transaction ends,
	// while other readers are still unblocked (FR-045).
	//
	// It is what makes "a closed campaign grants nobody" ABSOLUTE rather than
	// eventually true. Conditions 1-7 run OUTSIDE the grant transaction, so a close
	// can commit after the campaign was chosen and before the claim is written;
	// re-reading here is the single point where that race is decided, and the
	// answer is taken from the row the close actually committed.
	//
	// MUST be called inside the caller's `uow.Execute`. Outside a transaction
	// `FOR SHARE` degrades to a plain read and the guarantee silently disappears.
	GetByCodeForShare(ctx context.Context, campaignCode string) (*entities.Campaign, error)

	// Create inserts a new campaign.
	//
	// A duplicate code is a `PRIMARY KEY` violation, which is the data-layer half
	// of FR-011's uniqueness rule; the caller turns it into 409 (FR-054) rather
	// than reading first and racing.
	Create(ctx context.Context, campaign *entities.Campaign) error

	// Update writes the mutable fields of an existing campaign and reports
	// whether it actually changed a row.
	//
	// MUTABLE: `plan_slug`, `quota`, `reserve`, `starts_at`, `ends_at`.
	// NOT MUTABLE: `code` (FR-024) and `closed_at` (FR-034, one-way).
	//
	// `updated` is FALSE when the row no longer carries `expectedVersion`, which is
	// the optimistic-concurrency answer: two administrators edited the same campaign
	// and the second one is refused with 412 instead of silently overwriting
	// (FR-055). It is also FALSE for a row that does not exist at all, which the
	// caller must distinguish by reading the code first (FR-054: 404 before 412).
	//
	// The version is incremented by the statement itself, so it cannot be forgotten
	// at a call site — a `SET version = version + 1` written by hand in three
	// places is three places to get wrong.
	Update(ctx context.Context, campaign *entities.Campaign, expectedVersion int) (updated bool, err error)

	// Close force-closes a campaign by writing `closed_at`, and reports whether it
	// changed a row (FR-030/FR-031, FR-055).
	//
	// The instant is written by the DATABASE (`now()`), not passed in: `closed_at`
	// answers "when was this campaign closed" to an operator reading the audit trail
	// weeks later, and a caller's clock is not a trustworthy witness to its own
	// request.
	//
	// The same `WHERE code = ? AND version = ?` shape as `Update`, for the same
	// reason. It is NOT idempotent at this level: the idempotent check ("already
	// closed ⇒ succeed, do not compare versions", FR-034/FR-055) belongs to the use
	// case, which is the only layer that knows whether the caller already saw the
	// closed state.
	Close(ctx context.Context, campaignCode string, expectedVersion int) (updated bool, err error)

	// ExistsClaims reports whether the campaign has at least one grant
	// (FR-020/FR-021).
	//
	// This is the application layer's half of the budget immutability rule: the
	// trigger below it is the backstop that a hand-written UPDATE cannot get past,
	// and this read is what lets the API refuse with a 422 and an explanation
	// instead of a raw `check_violation` from the database.
	//
	// The condition is "has granted", NOT "the clock passed starts_at": a campaign
	// that has not granted anybody has no historical row to reinterpret, so editing
	// its budget is safe (spec.md Câu 3, research.md R3).
	ExistsClaims(ctx context.Context, campaignCode string) (bool, error)
}
