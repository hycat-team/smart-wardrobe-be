package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
)

// ICampaignClaimRepository is the durable counter behind a campaign's
// "first N accounts" rule (025 T033, FR-022 / FR-024 / FR-025 / FR-030).
//
// The counter and the claim are the SAME row on purpose (data-model.md §3.1):
// they have to commit together, and split across two tables they would be two
// sources of truth that have to be reconciled — which is exactly where a
// duplicate grant comes from.
type ICampaignClaimRepository interface {
	// NOTE: this interface does NOT embed `repositories.IGenericRepository`, unlike
	// its neighbours in this module.
	//
	// The generic interface declares `Create(ctx, entity) error`, and this ledger
	// needs `Create` to report WHETHER it inserted — the difference between "the
	// account's first claim was written" and "the account was already claimed",
	// which is the whole of FR-025. Two methods cannot share a name with
	// different results, so the ledger declares its own five methods and does not
	// inherit the generic CRUD set.
	//
	// That is also the right shape: the ledger is append-only. Nothing updates or
	// deletes a claim — FR-028 requires it to survive restarts, and a claim that
	// existed and then vanished would be a second chance at a grant.

	// LockClaimWindow takes the transaction-scoped advisory lock that serialises
	// quota decisions for a campaign.
	//
	// An advisory lock rather than `SELECT ... FOR UPDATE`, because the case that
	// matters is a campaign with no claims at all: a row lock locks nothing, so
	// two concurrent messages both count the same zero and both proceed (R3).
	//
	// MUST be called inside the caller's `uow.Execute` — the lock is released at
	// COMMIT or ROLLBACK, so it cannot outlive the transaction that took it, and a
	// nested `uow.Execute` would open a SECOND transaction on a second connection
	// that cannot see the outer one's rows.
	LockClaimWindow(ctx context.Context, campaignCode string) error

	// CountByBucket returns how many claims of ONE bucket exist for a campaign.
	//
	// It returns ONE number for ONE part, and both campaign counts are read with
	// it: `grantedMainCount` AND `grantedCompensationCount`.
	//
	// There is deliberately NO method that counts the main part by excluding a
	// condition on `fills_main_quota`. That flag marks exactly ONE row, so its
	// negation equals "total minus one" — which folds the reserve in as well, and
	// the reserve branch then trips `ux_campaign_claims_watermark` instead of
	// falling through. The reserve would never be usable, which is when it is most
	// needed (research.md R2b, data-model.md §6.2).
	//
	// Reads `ix_campaign_claims_campaign_bucket`, so it scans one part rather than
	// the whole table.
	CountByBucket(ctx context.Context, campaignCode string, bucket entities.GrantBucket) (int64, error)

	// GetWatermark returns the FR-030 priority watermark: the `registered_at` of
	// the claim that filled the main budget.
	//
	// It returns `found` as well as the value because `registered_at` is NOT NULL
	// while the row itself may be ABSENT — the common case before the main budget
	// is full. A pointer-only signature would make those two indistinguishable
	// (R3).
	//
	// The reserve branch requires the watermark to EXIST before granting: reserve
	// is by definition for accounts created BEFORE the watermark (FR-031), and
	// guessing when it is missing is how a wrong account gets a plan.
	GetWatermark(ctx context.Context, campaignCode string) (registeredAt time.Time, found bool, err error)

	// Create writes the claim unless the account already has one, and reports
	// whether it inserted.
	//
	// `ux_campaign_claims_user` is the data-layer guarantee behind FR-025 and
	// FR-012: a redelivered message, a second campaign, and a concurrent worker all
	// collide on the same constraint instead of each producing a grant. The insert
	// is what enforces the rule — not a check that runs before it, which two
	// concurrent callers would both pass.
	Create(ctx context.Context, claim *entities.CampaignClaim) (inserted bool, err error)

	// ListByCampaign returns a campaign's claims newest-first, for the FR-062
	// operator look-up. Reads `ix_campaign_claims_campaign_granted`.
	ListByCampaign(ctx context.Context, campaignCode string, limit int) ([]*entities.CampaignClaim, error)
}
