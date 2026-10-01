package campaign

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	sharedconfig "smart-wardrobe-be/shared/config"

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
type campaignEvaluator struct {
	campaigns []sharedconfig.CampaignConfig
	now       time.Time
}

// eligibility is the outcome of conditions 1-7.
//
// A nil Campaign with an empty Reason is impossible: the caller only calls the
// evaluator when it wants a campaign, and the evaluator either returns one or
// names the condition that stopped it. Carrying the reason rather than a bare
// bool is what FR-061 asks for — every skip has to be able to say why.
type eligibility struct {
	Campaign *sharedconfig.CampaignConfig
	Reason   SkipReason
}

// newCampaignEvaluator freezes the clock for one evaluation.
//
// `now` is passed in rather than read from `timeutils.GetNow` here so that every
// condition in a single evaluation sees the SAME instant. Reading the clock
// per-condition would let a message that arrives exactly at `ends_at` be
// admitted by condition 2 and rejected by condition 7, and the resulting skip
// reason would name the wrong rule.
func newCampaignEvaluator(campaigns []sharedconfig.CampaignConfig, now time.Time) *campaignEvaluator {
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
	if len(e.campaigns) == 0 {
		return eligibility{Reason: SkipCampaignNotConfigured}
	}

	// Condition 2: is any campaign awake? "Awake" is a CLOCK question, answered
	// before anything is read from the database: a campaign that has not opened or
	// has closed cannot be granted from, and knowing that without a query is what
	// keeps the queue moving when a campaign is over.
	if !e.anyCampaignAwake() {
		return eligibility{Reason: e.sleepReason()}
	}

	// Condition 3 (FR-091): the account was created through the administrator
	// endpoint. Its own reason code, because an administrator's test accounts are
	// valid accounts and the campaign simply does not apply to them.
	if reason := adminSkipReason(createdByAdmin); reason != "" {
		return eligibility{Reason: reason}
	}

	// Condition 7 (FR-011): choose the candidate that starts LATEST. Evaluated
	// before conditions 5 and 6 because those two are about the CHOSEN campaign's
	// plan — an account already on the winning campaign's plan is not eligible for
	// it, and that is a different answer than being ineligible for both.
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
	for i := range e.campaigns {
		if e.isAwake(e.campaigns[i]) {
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
func (e *campaignEvaluator) sleepReason() SkipReason {
	for i := range e.campaigns {
		if e.now.Before(e.campaigns[i].StartsAtTime) {
			return SkipCampaignNotStarted
		}
	}
	return SkipCampaignSleeping
}

// isAwake reports whether `now` lies inside the campaign's window.
//
// `ends_at` is OPTIONAL (FR-009): a campaign with none is open-ended, and a
// nil end is not "the zero time", which is why it is compared as a pointer rather
// than as a value.
func (e *campaignEvaluator) isAwake(campaign sharedconfig.CampaignConfig) bool {
	if e.now.Before(campaign.StartsAtTime) {
		return false
	}
	if campaign.EndsAtTime != nil && !e.now.Before(*campaign.EndsAtTime) {
		return false
	}
	return true
}

// selectCampaign implements `data-model.md` §6.1.
//
// Both sides of the window test use the ACCOUNT's creation instant, never the
// processing clock. That is what makes the rule pure: the same account is always
// assigned to the same campaign, whether its message arrives in fifty
// milliseconds or five minutes. Using "now" would drop a late message out of the
// window and discard an account for no reason at all (FR-078).
func (e *campaignEvaluator) selectCampaign(registeredAt time.Time) *sharedconfig.CampaignConfig {
	var chosen *sharedconfig.CampaignConfig
	for i := range e.campaigns {
		candidate := &e.campaigns[i]
		// Half-open [starts_at, ends_at): the start instant is INSIDE the window.
		if registeredAt.Before(candidate.StartsAtTime) {
			continue
		}
		if candidate.EndsAtTime != nil && !registeredAt.Before(*candidate.EndsAtTime) {
			continue
		}
		if chosen == nil || candidate.StartsAtTime.After(chosen.StartsAtTime) {
			chosen = candidate
		}
	}
	return chosen
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
