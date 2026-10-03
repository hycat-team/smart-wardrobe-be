package campaign

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// campaignEvaluator answers "which campaign, if any, is this account eligible
// for?" — conditions 1 to 7 of `data-model.md` §6.
//
// It is a separate type from `CampaignUseCase` because those seven conditions
// share no dependency with the two that need a lock. Keeping them apart means
// the eligibility rules can be read, reviewed and tested without wading through
// the quota algorithm, and the boundary is the one the specification draws:
// conditions 1-7 are pure reads, conditions 8-9 are a read-decide-write under an
// advisory lock.
//
// The ORDER of the conditions is load-bearing and is not interchangeable
// (`data-model.md` §6 header): the cheap, no-lock rules run first so that a
// campaign which is switched off, has not started, or does not apply to
// administrators never opens a transaction. Reordering them would not change any
// answer — it would only make the common skips pay for a lock.
//
// IT STILL READS NOTHING ITSELF (026, FR-040). The rows are handed to it by the
// caller, which reads them from `subscription.campaigns` through
// `ICampaignRepository.ListOpen` — one indexed query per signup message, no cache
// (FR-080). That boundary is what keeps conditions 1-7 testable with no database
// at all, and it is why the type below is `[]*entities.Campaign` and not a
// repository.
type campaignEvaluator struct {
	campaigns []*entities.Campaign
	now       time.Time
}

// eligibility is the outcome of conditions 1-7.
//
// A nil Campaign with an empty Reason is impossible: the caller only calls the
// evaluator when it wants a campaign, and the evaluator either returns one or
// names the condition that stopped it. Carrying the reason rather than a bare
// bool is what FR-061 asks for — every skip has to be able to say why.
type eligibility struct {
	Campaign *entities.Campaign
	Reason   SkipReason
}

// newCampaignEvaluator freezes the clock for one evaluation.
//
// `now` is passed in rather than read from `timeutils.GetNow` here so that every
// condition in a single evaluation sees the SAME instant. Reading the clock
// per-condition would let a message that arrives exactly at `ends_at` be
// admitted by condition 2 and rejected by condition 7, and the resulting skip
// reason would name the wrong rule.
func newCampaignEvaluator(campaigns []*entities.Campaign, now time.Time) *campaignEvaluator {
	return &campaignEvaluator{campaigns: campaigns, now: now}
}

// evaluate applies conditions 1 to 7 in the order of `data-model.md` §6.
//
// `sub` is the account's current subscription, read by the caller. It is passed
// in rather than fetched here so that this stays a pure function of its inputs —
// which is what makes conditions 5 and 6 testable without a database.
func (e *campaignEvaluator) evaluate(
	ctx context.Context,
	accountID uuid.UUID,
	registeredAt time.Time,
	createdByAdmin bool,
	sub *entities.UserSubscription,
) eligibility {
	// Condition 1: is the mechanism switched on at all?
	//
	// An EMPTY REGISTRY IS A NORMAL STATE now (026, SC-008), not a
	// misconfiguration: campaigns are created through the admin API, so the very
	// first boot of a fresh environment has none, and it must start, serve and
	// schedule the catch-up sweep without a single warning. The reason code is the
	// one 025 used, so the published set of twelve `reason` codes is unchanged
	// (FR-050).
	if len(e.campaigns) == 0 {
		return eligibility{Reason: SkipCampaignNotConfigured}
	}

	// Condition 2: is any campaign awake? "Awake" is a CLOCK question answered
	// from the rows the caller already read: a campaign that has not opened, has run
	// out its window, or has been force-closed cannot be granted from, and knowing
	// that without a lock is what keeps the queue moving when a campaign is over.
	//
	// It is answered AFTER the one indexed query rather than instead of it, because
	// the registry is no longer in memory (FR-040). What condition 2 still buys is
	// that the query is the only thing this evaluation costs: no transaction, no
	// advisory lock and no counts for a campaign that is asleep.
	if !e.anyCampaignAwake() {
		return eligibility{Reason: e.sleepReason()}
	}

	// Condition 3 (FR-091): the account was created through the administrator
	// endpoint. Its own reason code, because an administrator's test accounts are
	// valid accounts and the campaign simply does not apply to them.
	if reason := adminSkipReason(createdByAdmin); reason != "" {
		return eligibility{Reason: reason}
	}

	// Condition 7 (FR-011/FR-040): choose the candidate that starts LATEST, with
	// `code DESC` as the deterministic tie-break. Evaluated before conditions 5 and
	// 6 because those two are about the CHOSEN campaign's plan — an account already
	// on the winning campaign's plan is not eligible for it, and that is a different
	// answer than being ineligible for both.
	chosen := e.selectCampaign(registeredAt)
	if chosen == nil {
		return eligibility{Reason: SkipOutsideWindow}
	}

	// Condition 5 (FR-018): already on exactly this plan, term not run out.
	// Granting would STACK another term onto it, which is what FR-018 forbids.
	if isOnPlanWithLiveTerm(sub, chosen.PlanSlug, e.now) {
		return eligibility{Reason: SkipAlreadyOnPlan}
	}

	// Condition 6: the account was just demoted to free while the previous plan's
	// cleanup is still recorded. Granting over that produces a row describing two
	// plans at once, and whichever component reads it next picks one.
	if hasUnsettledFallback(sub) {
		return eligibility{Reason: SkipPlanNotSettled}
	}

	_ = accountID
	_ = ctx
	return eligibility{Campaign: chosen}
}

// adminSkipReason is condition 3 of `data-model.md` §6, in ONE function (FR-080,
// FR-091, FR-050).
//
// The catch-up sweep reaches the same rule from the other direction: it reads
// `identity.users` through the account contract and sees accounts an
// administrator created, where the message path learns the fact from the
// payload. Two facts, one rule. If each path tested the flag itself, one of them
// would eventually be edited without the other and the SAME account would be
// granted by the sweep and refused by the message — which is exactly the drift
// FR-080 forbids, and it is a drift nobody would notice because both answers look
// reasonable in isolation.
//
// A function returning the REASON rather than a bool is deliberate: the reason
// code is what FR-061 requires each path to record, so neither path has to
// restate it, and the two cannot end up reporting different codes for the same
// condition. An empty reason means "not an administrator account".
func adminSkipReason(createdByAdmin bool) SkipReason {
	if createdByAdmin {
		return SkipCreatedByAdmin
	}
	return ""
}

// anyCampaignAwake reports whether at least one campaign is inside its window.
func (e *campaignEvaluator) anyCampaignAwake() bool {
	for _, campaign := range e.campaigns {
		if campaign != nil && e.isAwake(campaign) {
			return true
		}
	}
	return false
}

// sleepReason distinguishes "has not started yet" from "already finished".
//
// Two codes, not one, because the two need different people to act: the first is
// a countdown, the second is a retrospective. Collapsing them would make a
// campaign that ended on schedule look misconfigured.
//
// A FORCE-CLOSED campaign (026, FR-031) falls into "already finished", which is
// exactly right: `campaign_sleeping` is the code 025 already used for "this
// campaign is over", so a campaign closed by an operator needs no new reason and
// the published set of twelve `reason` codes is unchanged (FR-050). The codes are
// not split per campaign — `selectCampaign` below skips closed campaigns, so by
// the time this runs they cannot reach the "not started" branch.
func (e *campaignEvaluator) sleepReason() SkipReason {
	for _, campaign := range e.campaigns {
		if campaign.ClosedAt != nil {
			continue
		}
		if e.now.Before(campaign.StartsAt) {
			return SkipCampaignNotStarted
		}
	}
	return SkipCampaignSleeping
}

// isAwake is THE ONE DEFINITION of "this campaign can still grant", used by
// condition 2 and by the candidate set of `selectCampaign` (026, data-model.md
// §4.3, FR-031/FR-040).
//
// Three terms, and the third is new:
//
//	starts_at <= now  &&  now < ends_at   — the window 025 checked
//	closed_at IS NULL                     — 026's force-close (FR-031)
//
// The closed term is why "closed" is not a fourth window term. A force-closed
// campaign is asleep wherever the clock is: its window may still say it is open,
// and an open-ended campaign has no window term at all. Folding it into the
// window would mean rewriting the window for closed campaigns, and the registry
// would then need to mutate `ends_at` on close — which is exactly the write a
// one-way close must not do.
//
// `ends_at` is OPTIONAL: a campaign with none is open-ended, and a nil end is not
// "the zero time", which is why it is compared as a pointer.
func (e *campaignEvaluator) isAwake(campaign *entities.Campaign) bool {
	if campaign.ClosedAt != nil {
		return false
	}
	if e.now.Before(campaign.StartsAt) {
		return false
	}
	if campaign.EndsAt != nil && !e.now.Before(*campaign.EndsAt) {
		return false
	}
	return true
}

// selectCampaign implements `data-model.md` §6.1.
//
// TWO INSTANTS, TWO JOBS. The window test uses the ACCOUNT's creation instant,
// never the processing clock. That is what makes the rule pure: the same account
// is always assigned to the same campaign, whether its message arrives in fifty
// milliseconds or five minutes. Using "now" would drop a late message out of the
// window and discard an account for no reason at all (FR-078).
//
// "Is this campaign still running?" is the OTHER term of the same candidate set
// and it is a clock question, answered against `now`. Without it a campaign that
// closed yesterday keeps granting from its window forever — which is how a
// seasonal campaign ends up giving away next season's budget to last season's
// signups (FR-013). Note the asymmetry is deliberate: the window asks "was this
// account inside A?", the liveness asks "is A still open?", and an account can
// fail either without failing the other.
//
// THE TIE-BREAK IS `code DESC` (026, FR-040, spec.md Câu 6). 025 did not need one:
// a YAML list cannot contain two entries with the same code, but it CAN contain
// two with the same `starts_at`, and the winner among them depended on the order
// the operator happened to write them down. With the registry an operator can add
// a campaign from any screen at any time, so an unpinned tie would make the same
// account land in a different campaign depending on which row the database
// returned first — and FR-041 requires the choice to be a deterministic function
// of (rows, `registered_at`) and of nothing else.
//
// The same ordering appears in the repository's `ORDER BY`, which is not a second
// rule: it makes the SQL's row order stable, and this comparison makes the CHOICE
// stable even if a caller hands the rows over in another order.
func (e *campaignEvaluator) selectCampaign(registeredAt time.Time) *entities.Campaign {
	var chosen *entities.Campaign
	for _, candidate := range e.campaigns {
		if candidate == nil {
			continue
		}
		// `c.still_running(now)`, the first term of §6.1's candidate set. It is not
		// spelled out here: `isAwake` IS that term — the very function condition 2
		// asks with — so "this campaign is awake" has ONE definition in this
		// package. A second copy of the comparison is a second rule, and two rules
		// about the same window eventually disagree (FR-080).
		if !e.isAwake(candidate) {
			continue
		}
		// Half-open [starts_at, ends_at): the start instant is INSIDE the window.
		if registeredAt.Before(candidate.StartsAt) {
			continue
		}
		if candidate.EndsAt != nil && !registeredAt.Before(*candidate.EndsAt) {
			continue
		}
		if chosen == nil || startsLaterThan(candidate, chosen) {
			chosen = candidate
		}
	}
	return chosen
}

// startsLaterThan implements the two-key ordering of FR-040: newest `starts_at`
// wins, and the LATEST `code` breaks the tie.
func startsLaterThan(candidate, chosen *entities.Campaign) bool {
	if candidate.StartsAt.Equal(chosen.StartsAt) {
		return candidate.Code > chosen.Code
	}
	return candidate.StartsAt.After(chosen.StartsAt)
}

// isOnPlanWithLiveTerm implements condition 5 (FR-018).
//
// A nil subscription is a brand-new account and is therefore NOT on any plan —
// the default provisioning may not have run yet, and treating "no row" as "on
// the free plan" would make every fresh account look like it needed nothing.
func isOnPlanWithLiveTerm(sub *entities.UserSubscription, planSlug string, now time.Time) bool {
	if sub == nil || sub.CurrentPlanCode != planSlug {
		return false
	}
	if sub.ExpiresAt == nil {
		// A plan with no expiry is a LIFETIME plan, so its term never runs out and
		// stacking a campaign term onto it is exactly the harm FR-018 names.
		return true
	}
	return sub.ExpiresAt.After(now)
}

// hasUnsettledFallback implements condition 6.
//
// The signal is a subscription that sits on the free plan while still carrying a
// recorded fallback. That combination means a downgrade is in flight: the account
// has left its old plan and the old plan has not been cleared. Granting a new plan
// now would either erase the pending downgrade or leave two plans described by one
// row, and both outcomes are silent.
func hasUnsettledFallback(sub *entities.UserSubscription) bool {
	if sub == nil {
		return false
	}
	if sub.CurrentTierRank != 0 {
		// On a paid plan: there is no downgrade in flight.
		return false
	}
	return sub.FallbackPlanCode != nil && *sub.FallbackPlanCode != ""
}
