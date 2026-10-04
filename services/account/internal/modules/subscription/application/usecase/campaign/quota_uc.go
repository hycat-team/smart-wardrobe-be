package campaign

import (
	"context"
	"time"

	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"

	"github.com/google/uuid"
)

// quotaDecider implements condition 8 of `data-model.md` §6.2 — the hard budget
// ceiling — and condition 9, the write of the claim.
//
// It is a separate type from the evaluator and from the use case because it is
// the only part of the mechanism that is allowed to be inside a locked
// transaction. Nothing else in the package takes the advisory lock, so the scope
// of the lock is one file wide and a reader can see all of it at once.
//
// THE TWO COUNTS ARE INDEPENDENT NUMBERS, AND THAT IS THE WHOLE POINT
// (research.md R2b). The obvious implementation counts the main part as
// `COUNT(*) WHERE NOT fills_main_quota`, on the reasoning that the main budget is
// everything except the one row that filled it. That reasoning is wrong, and it
// fails in a way no test of the happy path would catch:
//
//	fills_main_quota marks exactly ONE row, so its negation equals
//	"total claims minus one" — which also counts the RESERVE.
//
// After fifty main claims, the fifty-first account computes `mainCount = 49`,
// concludes it is the fiftieth, sets `fills_main_quota = TRUE`, and collides with
// `ux_campaign_claims_watermark` — an error instead of falling through to the
// reserve branch. The reserve is then unreachable for the entire campaign, and it
// exists precisely for the accounts the main budget turned away. The second
// consequence is quieter: the admin numbers report fifty main and one reserve
// where the truth is forty-nine main and two reserve (FR-065).
//
// So both counts read `grant_bucket`, and the flag is read for exactly one thing:
// finding the watermark row.
//
// IT ALSO HOLDS THE REGISTRY, and that is what FR-045 costs this file: the decider
// is the only part of the mechanism allowed inside a locked transaction, so the
// re-read of the campaign row under a SHARE lock belongs here and nowhere else.
type quotaDecider struct {
	claimRepo repositories.ICampaignClaimRepository
	registry  repositories.ICampaignRepository
}

func newQuotaDecider(
	claimRepo repositories.ICampaignClaimRepository,
	registry repositories.ICampaignRepository,
) *quotaDecider {
	return &quotaDecider{claimRepo: claimRepo, registry: registry}
}

// quotaOutcome is the decision conditions 8 and 9 produced.
type quotaOutcome struct {
	// Granted is true when the account may receive the plan.
	Granted bool
	// Bucket is which part of the budget the grant draws from. Written to
	// `grant_bucket` at insert time, never inferred afterwards.
	Bucket entities.GrantBucket
	// FillsMainQuota is true on exactly one claim per campaign: the one that
	// filled the main budget. It is the FR-030 watermark, and it is a different
	// question from `Bucket` — forty-nine other rows are also "main".
	FillsMainQuota bool
	// ExpiresAt is the term the plan is granted for, taken from the plan's own
	// duration (FR-019) and anchored on the account's registration so that a
	// message arriving late still grants the term the account was promised.
	ExpiresAt time.Time
	// Reason is the skip code when Granted is false.
	Reason SkipReason
}

// claimInput is everything the decider needs, gathered by the caller so that this
// function performs no I/O of its own choosing.
type claimInput struct {
	// Campaign is the registry row the evaluator chose (026, FR-001). A pointer
	// because it is read, never copied into a decision: the quota arithmetic only
	// ever asks `Code`, `Quota` and `Reserve`, and a value copy of the whole row is
	// one more copy that could drift from the row it was read from.
	Campaign     *entities.Campaign
	Plan         *entities.SubscriptionPlan
	UserID       uuid.UUID
	RegisteredAt time.Time
	Now          time.Time
}

// decideAndClaim runs conditions 8 and 9.
//
// It MUST be called inside the caller's `uow.Execute`. The advisory lock it takes
// is transaction-scoped: it is released at COMMIT or ROLLBACK, so a worker that
// dies mid-decision leaves nothing behind (R3). Calling it outside a transaction
// would take a SESSION-level lock on a pooled connection and hold it for that
// connection's lifetime — a leak that only shows up under connection pressure,
// long after the change that caused it.
func (q *quotaDecider) decideAndClaim(ctx context.Context, in claimInput) (quotaOutcome, error) {
	if err := q.claimRepo.LockClaimWindow(ctx, in.Campaign.Code); err != nil {
		return quotaOutcome{}, subscriptionerrors.ErrCampaignLockClaimWindowFailed(in.Campaign.Code)
	}

	// FR-045: RE-READ THE CAMPAIGN UNDER A SHARE LOCK, INSIDE THIS TRANSACTION, and
	// BEFORE ANY COUNT. Conditions 1-7 ran outside it, so a campaign an operator
	// closed can have been chosen from rows that were open a moment ago; this is the
	// single place that race is decided, and the answer comes from the row the close
	// actually committed rather than from the copy this process picked.
	//
	// THE ORDER IS THE ADVISORY LOCK FIRST, THEN THE ROW. Every grant already
	// serialises on the campaign's advisory lock, so the lock order is always
	// "advisory → row share" and the close path — which takes only the row lock —
	// cannot deadlock against it (research.md R12).
	//
	// `FOR SHARE` AND NOT `FOR UPDATE`, and the difference is the point: a close or a
	// budget edit WAITS for this transaction, while other grants reading the same row
	// do not. A burst of signups against a live campaign is therefore not serialised
	// behind one another, which `FOR UPDATE` would do.
	//
	// THE RELOADED ROW REPLACES THE ONE THE EVALUATOR CHOSE. Not only `closed_at`:
	// FR-045 also requires a grant to be skipped when the budget changed in a way that
	// makes it invalid, and deciding that against the reloaded `quota`/`reserve` is
	// what makes the rule one statement rather than two.
	locked, err := q.registry.GetByCodeForShare(ctx, in.Campaign.Code)
	if err != nil {
		// A RETRYABLE-LOOKING infrastructure failure, reported as a failure and never
		// as "no campaign configured": substituting an empty set here would turn a
		// database problem into a silent skip on every signup (spec.md §10's rule for
		// the read path, and it applies identically to this one).
		return quotaOutcome{}, subscriptionerrors.ErrCampaignRegistryReadFailed()
	}
	if locked == nil {
		// Unreachable: there is NO delete path (FR-005), so a code that existed a
		// moment ago still exists. Reported as "no campaign configured" rather than
		// crashing, because a skip is a decision and a nil dereference is a crash on
		// the message path.
		return quotaOutcome{Reason: SkipCampaignNotConfigured}, nil
	}
	if locked.ClosedAt != nil {
		// FR-031/FR-034, at the only point where it can still be enforced. The reason
		// is `campaign_sleeping` and NOT a new code: 025 already uses it for a
		// campaign that is over, a closed campaign is over, and FR-050 keeps the
		// `reason` set at exactly twelve codes (spec.md Câu 4, quickstart V2 step 5).
		return quotaOutcome{Reason: SkipCampaignSleeping}, nil
	}
	in.Campaign = locked

	// Two independent counts, one per part. Read on `grant_bucket` so each one
	// scans only its own part of `ix_campaign_claims_campaign_bucket`.
	//
	// A count that FAILS must surface as a failure, never as zero. Substituting
	// zero would grant straight past the ceiling on the very next message, and the
	// ceiling is the one number a campaign may not exceed.
	mainCount, err := q.claimRepo.CountByBucket(ctx, in.Campaign.Code, entities.GrantBucketMain)
	if err != nil {
		return quotaOutcome{}, subscriptionerrors.ErrCampaignCountClaimFailed(in.Campaign.Code, string(entities.GrantBucketMain))
	}
	reserveCount, err := q.claimRepo.CountByBucket(ctx, in.Campaign.Code, entities.GrantBucketReserve)
	if err != nil {
		return quotaOutcome{}, subscriptionerrors.ErrCampaignCountClaimFailed(in.Campaign.Code, string(entities.GrantBucketReserve))
	}

	// The watermark is read once and used for two questions: is the main budget
	// already frozen, and — in the reserve branch — is this account old enough.
	// Read after the counts because a failure here must not be mistaken for
	// "no watermark yet", which would let the reserve grant against a boundary
	// that does not exist (FR-031).
	watermarkAt, watermarkSet, err := q.claimRepo.GetWatermark(ctx, in.Campaign.Code)
	if err != nil {
		return quotaOutcome{}, subscriptionerrors.ErrCampaignReadWatermarkFailed(in.Campaign.Code)
	}

	outcome := q.chooseBucket(in, mainCount, reserveCount, watermarkAt, watermarkSet)
	if !outcome.Granted {
		return outcome, nil
	}

	// Condition 9: write the claim. `ON CONFLICT (user_id) DO NOTHING` is what
	// enforces FR-025 and FR-012, and it is the INSERT that does it rather than a
	// check that runs first — two concurrent callers both pass any pre-check, and
	// only one can win the constraint. The loser is told "already claimed" and
	// nothing is granted, which is the whole of SC-002.
	inserted, err := q.claimRepo.Create(ctx, &entities.CampaignClaim{
		UserID:         in.UserID,
		CampaignCode:   in.Campaign.Code,
		PlanSlug:       in.Plan.Slug,
		PlanID:         in.Plan.ID,
		RegisteredAt:   in.RegisteredAt,
		ExpiresAt:      outcome.ExpiresAt,
		GrantBucket:    outcome.Bucket,
		FillsMainQuota: outcome.FillsMainQuota,
	})
	if err != nil {
		return quotaOutcome{}, subscriptionerrors.ErrCampaignWriteClaimFailed(in.Campaign.Code)
	}
	if !inserted {
		return quotaOutcome{Reason: SkipAlreadyClaimed}, nil
	}
	return outcome, nil
}

// chooseBucket is the decision itself, transcribed from `data-model.md` §6.2.
//
// Note what is NOT here: no comparison of the TOTAL against `quota + reserve`.
// Each branch is bounded by its own counter — the main branch runs only while
// `mainCount < quota`, the reserve branch only while `reserveCount < reserve` —
// so the total ceiling holds by construction. The earlier version's total
// comparison was not merely redundant but wrong, because its "reserve granted"
// figure was itself computed by the subtraction this file refuses to use.
func (q *quotaDecider) chooseBucket(
	in claimInput,
	mainCount, reserveCount int64,
	watermarkAt time.Time,
	watermarkSet bool,
) quotaOutcome {
	quota := int64(in.Campaign.Quota)
	reserve := int64(in.Campaign.Reserve)

	switch {
	case mainCount < quota:
		// Main branch. The claim that brings the count to `quota` is the one that
		// FILLED the budget, so it carries the watermark.
		//
		// The `!watermarkSet` guard is a safe second lock: if the flag were ever
		// cleared by hand, or a campaign were reloaded with a different `quota`, the
		// rule must not write a SECOND watermark and break the partial unique index.
		fills := mainCount+1 == quota && !watermarkSet
		return quotaOutcome{
			Granted:        true,
			Bucket:         entities.GrantBucketMain,
			FillsMainQuota: fills,
			ExpiresAt:      grantedExpiryFor(in),
		}

	case reserveCount < reserve:
		// Reserve branch (FR-031). The reserve is for accounts created BEFORE the
		// watermark: the fifty-first person through the door is not made whole by
		// handing them a fifty-first slot.
		//
		// The watermark MUST EXIST for this branch to run. Main budget full with no
		// watermark is a state the data layer cannot produce — the flag is written
		// by the very grant that fills the budget — so seeing it means the ledger
		// was edited or the campaign was reloaded with a different quota. Rather
		// than guess which accounts the reserve belongs to, report the anomaly:
		// a wrong grant is permanent and silent, an anomaly is visible.
		if !watermarkSet {
			return quotaOutcome{Reason: SkipQuotaAnomaly}
		}
		if !in.RegisteredAt.Before(watermarkAt) {
			return quotaOutcome{Reason: SkipQuotaMain}
		}
		return quotaOutcome{
			Granted:        true,
			Bucket:         entities.GrantBucketReserve,
			FillsMainQuota: false,
			ExpiresAt:      grantedExpiryFor(in),
		}

	default:
		// Both parts full. The reserve is a CEILING and not a target (FR-032), so a
		// campaign that ends with reserve left over has behaved correctly.
		return quotaOutcome{Reason: SkipQuotaCompensation}
	}
}

// grantedExpiryFor computes the granted term from the PLAN, never from a literal
// (FR-019): `premium-monthly` happens to be 30 days, and hard-coding that is how
// a quarterly campaign ends up granting a month.
//
// The term is anchored on the GRANT INSTANT, matching
// `subscription.ResolveExpiryFor` in the shared apply workflow.
//
// This used to anchor on the account's registration, and that was wrong twice
// over. It gave the claim row a different expiry from the subscription row the
// same grant wrote, so the admin query and the audit trail would report a date
// the customer did not actually have — and once the grant event moved to the
// processing clock (see `grant_uc.go`, and why the AI service rejects an
// earlier stamp) the two rows stopped agreeing at all. Two records describing
// one grant must not disagree about when it ends.
//
// Anchoring on the grant also keeps the promise: a message that took four days
// to arrive hands out a full term rather than four fewer days, and every
// account gets the same length regardless of queue latency.
func grantedExpiryFor(in claimInput) time.Time {
	if in.Plan.DurationDays == nil {
		// A plan with no duration has no expiry. The claim column is NOT NULL and
		// the granted plan is finite (FR-010), so this is unreachable for a plan
		// that passed admission; anchoring on the grant instant keeps the value
		// defined if it ever is not.
		return in.Now
	}
	return in.Now.AddDate(0, 0, *in.Plan.DurationDays)
}
