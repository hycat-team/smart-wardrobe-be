package entities

import (
	"fmt"
	plankind "smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"
	"time"
)

func (u *UserSubscription) Activate(plan *SubscriptionPlan, snapshot JSONDocument, eventKey string, at time.Time) (*UserSubscriptionEvent, error) {
	if u.CurrentPlanKind != plankind.DefaultFree {
		return nil, fmt.Errorf("cannot activate subscription: current plan is not default-free")
	}
	if plan.PlanKind != plankind.Finite && plan.PlanKind != plankind.Lifetime {
		return nil, fmt.Errorf("cannot activate subscription: target plan must be finite or lifetime")
	}

	fromPlan := u.CurrentPlanCode
	fromTier := u.CurrentTierRank

	u.SubscriptionPlanID = plan.ID
	u.SubscriptionPlan = plan
	u.CurrentPlanCode = plan.Slug
	u.CurrentTierRank = plan.TierRank
	u.CurrentPlanKind = plan.PlanKind
	u.CurrentBenefitSnapshot = snapshot
	u.StartedAt = at

	if plan.PlanKind == plankind.Finite {
		if plan.DurationDays == nil {
			return nil, fmt.Errorf("finite plan missing duration days")
		}
		exp := u.expiryFor(plan, at)
		u.ExpiresAt = &exp
	} else {
		u.ExpiresAt = nil
		u.IsAutoRenewEnabled = false
	}

	u.Version++
	u.UpdatedAt = at

	key := eventKey
	if key == "" {
		key = fmt.Sprintf("STATE_CHANGE:ACTIVATE:%s:%d", u.UserID, u.Version)
	}

	event := &UserSubscriptionEvent{
		EventKey:     key,
		UserID:       u.UserID,
		EventType:    string(subscriptioneventtype.ACTIVATED),
		FromPlanCode: &fromPlan,
		FromTierRank: &fromTier,
		ToPlanCode:   &u.CurrentPlanCode,
		ToTierRank:   &u.CurrentTierRank,
		OccurredAt:   at,
		EffectiveAt:  at,
	}

	return event, nil
}

func (u *UserSubscription) Extend(plan *SubscriptionPlan, snapshot JSONDocument, eventKey string, at time.Time) (*UserSubscriptionEvent, error) {
	if u.CurrentPlanKind != plankind.Finite {
		return nil, fmt.Errorf("cannot extend subscription: current plan is not finite")
	}
	if plan.PlanKind != plankind.Finite {
		return nil, fmt.Errorf("cannot extend subscription: target plan is not finite")
	}
	if plan.TierRank != u.CurrentTierRank {
		return nil, fmt.Errorf("cannot extend subscription: tier rank mismatch")
	}

	fromPlan := u.CurrentPlanCode
	fromTier := u.CurrentTierRank

	base := at
	if u.ExpiresAt != nil && u.ExpiresAt.After(at) {
		base = *u.ExpiresAt
	}

	exp := u.expiryFor(plan, base)
	u.ExpiresAt = &exp
	u.CurrentBenefitSnapshot = snapshot
	u.Version++
	u.UpdatedAt = at

	key := eventKey
	if key == "" {
		key = fmt.Sprintf("STATE_CHANGE:EXTEND:%s:%d", u.UserID, u.Version)
	}

	event := &UserSubscriptionEvent{
		EventKey:     key,
		UserID:       u.UserID,
		EventType:    string(subscriptioneventtype.EXTENDED),
		FromPlanCode: &fromPlan,
		FromTierRank: &fromTier,
		ToPlanCode:   &u.CurrentPlanCode,
		ToTierRank:   &u.CurrentTierRank,
		OccurredAt:   at,
		EffectiveAt:  at,
	}

	return event, nil
}

func (u *UserSubscription) Upgrade(plan *SubscriptionPlan, snapshot JSONDocument, eventKey string, at time.Time) (*UserSubscriptionEvent, error) {
	if u.CurrentPlanKind == plankind.DefaultFree {
		return nil, fmt.Errorf("cannot upgrade subscription: use Activate for default-free plans")
	}
	if plan.TierRank <= u.CurrentTierRank {
		return nil, fmt.Errorf("cannot upgrade subscription: target plan tier rank must be higher than current")
	}

	fromPlan := u.CurrentPlanCode
	fromTier := u.CurrentTierRank

	if u.CurrentPlanKind == plankind.Lifetime {
		u.clearFallbackFields()
	}

	u.SubscriptionPlanID = plan.ID
	u.SubscriptionPlan = plan
	u.CurrentPlanCode = plan.Slug
	u.CurrentTierRank = plan.TierRank
	u.CurrentPlanKind = plan.PlanKind
	u.CurrentBenefitSnapshot = snapshot
	u.StartedAt = at

	if plan.PlanKind == plankind.Finite {
		if plan.DurationDays == nil {
			return nil, fmt.Errorf("finite plan missing duration days")
		}
		exp := u.expiryFor(plan, at)
		u.ExpiresAt = &exp
	} else {
		u.ExpiresAt = nil
		u.IsAutoRenewEnabled = false
	}

	u.Version++
	u.UpdatedAt = at

	key := eventKey
	if key == "" {
		key = fmt.Sprintf("STATE_CHANGE:UPGRADE:%s:%d", u.UserID, u.Version)
	}

	event := &UserSubscriptionEvent{
		EventKey:     key,
		UserID:       u.UserID,
		EventType:    string(subscriptioneventtype.UPGRADED),
		FromPlanCode: &fromPlan,
		FromTierRank: &fromTier,
		ToPlanCode:   &u.CurrentPlanCode,
		ToTierRank:   &u.CurrentTierRank,
		OccurredAt:   at,
		EffectiveAt:  at,
	}

	return event, nil
}

func (u *UserSubscription) OverlayLifetime(plan *SubscriptionPlan, snapshot JSONDocument, eventKey string, at time.Time) (*UserSubscriptionEvent, error) {
	if u.CurrentPlanKind != plankind.Lifetime {
		return nil, fmt.Errorf("cannot overlay subscription: current plan is not lifetime")
	}
	if plan.PlanKind != plankind.Finite {
		return nil, fmt.Errorf("cannot overlay subscription: target plan is not finite")
	}
	if plan.TierRank <= u.CurrentTierRank {
		return nil, fmt.Errorf("cannot overlay subscription: target plan tier rank must be higher than current")
	}

	fromPlan := u.CurrentPlanCode
	fromTier := u.CurrentTierRank

	// Copy the lifetime values into local variables before taking their address.
	// Pointing the fallback fields at the struct fields themselves makes the
	// overlay below overwrite the lifetime plan with the overlaid finite plan.
	planID := u.SubscriptionPlanID
	u.FallbackPlanID = &planID
	planCode := u.CurrentPlanCode
	u.FallbackPlanCode = &planCode
	rank := u.CurrentTierRank
	u.FallbackTierRank = &rank
	kind := u.CurrentPlanKind
	u.FallbackPlanKind = &kind
	u.FallbackBenefitSnapshot = u.CurrentBenefitSnapshot

	u.SubscriptionPlanID = plan.ID
	u.SubscriptionPlan = plan
	u.CurrentPlanCode = plan.Slug
	u.CurrentTierRank = plan.TierRank
	u.CurrentPlanKind = plan.PlanKind
	u.CurrentBenefitSnapshot = snapshot
	u.StartedAt = at

	if plan.DurationDays == nil {
		return nil, fmt.Errorf("finite plan missing duration days")
	}
	exp := u.expiryFor(plan, at)
	u.ExpiresAt = &exp

	u.Version++
	u.UpdatedAt = at

	key := eventKey
	if key == "" {
		key = fmt.Sprintf("STATE_CHANGE:OVERLAY:%s:%d", u.UserID, u.Version)
	}

	event := &UserSubscriptionEvent{
		EventKey:     key,
		UserID:       u.UserID,
		EventType:    string(subscriptioneventtype.OVERLAID),
		FromPlanCode: &fromPlan,
		FromTierRank: &fromTier,
		ToPlanCode:   &u.CurrentPlanCode,
		ToTierRank:   &u.CurrentTierRank,
		OccurredAt:   at,
		EffectiveAt:  at,
	}

	return event, nil
}

func (u *UserSubscription) RestoreFallback(eventKey string, at time.Time) (*UserSubscriptionEvent, error) {
	if u.CurrentPlanKind != plankind.Finite {
		return nil, fmt.Errorf("cannot restore fallback: current plan is not finite")
	}
	if u.FallbackPlanID == nil {
		return nil, fmt.Errorf("cannot restore fallback: no fallback plan configured")
	}
	if u.FallbackPlanKind == nil || *u.FallbackPlanKind != plankind.Lifetime {
		return nil, fmt.Errorf("cannot restore fallback: fallback plan is not lifetime")
	}

	fromPlan := u.CurrentPlanCode
	fromTier := u.CurrentTierRank

	u.SubscriptionPlanID = *u.FallbackPlanID
	u.CurrentPlanCode = *u.FallbackPlanCode
	u.CurrentTierRank = *u.FallbackTierRank
	u.CurrentPlanKind = *u.FallbackPlanKind
	u.CurrentBenefitSnapshot = u.FallbackBenefitSnapshot
	u.StartedAt = *u.ExpiresAt
	u.ExpiresAt = nil
	u.IsAutoRenewEnabled = false

	u.clearFallbackFields()
	u.Version++
	u.UpdatedAt = at

	key := eventKey
	if key == "" {
		key = fmt.Sprintf("LIFETIME_FALLBACK_RESTORED:%s:%s", u.UserID, u.StartedAt.UTC().Format(time.RFC3339))
	}

	event := &UserSubscriptionEvent{
		EventKey:     key,
		UserID:       u.UserID,
		EventType:    string(subscriptioneventtype.LIFETIME_FALLBACK_RESTORED),
		FromPlanCode: &fromPlan,
		FromTierRank: &fromTier,
		ToPlanCode:   &u.CurrentPlanCode,
		ToTierRank:   &u.CurrentTierRank,
		OccurredAt:   at,
		EffectiveAt:  u.StartedAt,
	}

	return event, nil
}

func (u *UserSubscription) DowngradeToFree(freePlan *SubscriptionPlan, eventKey string, at time.Time) (*UserSubscriptionEvent, error) {
	if freePlan.PlanKind != plankind.DefaultFree {
		return nil, fmt.Errorf("cannot downgrade to free: target plan is not default-free")
	}

	fromPlan := u.CurrentPlanCode
	fromTier := u.CurrentTierRank

	u.SubscriptionPlanID = freePlan.ID
	u.SubscriptionPlan = freePlan
	u.CurrentPlanCode = freePlan.Slug
	u.CurrentTierRank = freePlan.TierRank
	u.CurrentPlanKind = freePlan.PlanKind
	u.CurrentBenefitSnapshot = JSONDocument(`{}`)
	u.StartedAt = at
	u.ExpiresAt = nil
	u.IsAutoRenewEnabled = false

	u.clearFallbackFields()
	u.Version++
	u.UpdatedAt = at

	key := eventKey
	if key == "" {
		key = fmt.Sprintf("SUBSCRIPTION_DOWNGRADED_TO_FREE:%s:%d", u.UserID, u.Version)
	}

	event := &UserSubscriptionEvent{
		EventKey:     key,
		UserID:       u.UserID,
		EventType:    string(subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE),
		FromPlanCode: &fromPlan,
		FromTierRank: &fromTier,
		ToPlanCode:   &u.CurrentPlanCode,
		ToTierRank:   &u.CurrentTierRank,
		OccurredAt:   at,
		EffectiveAt:  at,
	}

	return event, nil
}

func (u *UserSubscription) SetAutoRenew(enable bool, eventKey string, at time.Time) (*UserSubscriptionEvent, error) {
	if u.CurrentPlanKind != plankind.Finite {
		return nil, fmt.Errorf("cannot toggle auto-renew: plan is not finite")
	}

	fromPlan := u.CurrentPlanCode
	fromTier := u.CurrentTierRank

	u.IsAutoRenewEnabled = enable
	u.Version++
	u.UpdatedAt = at

	key := eventKey
	if key == "" {
		key = fmt.Sprintf("STATE_CHANGE:AUTO_RENEW:%s:%d", u.UserID, u.Version)
	}

	event := &UserSubscriptionEvent{
		EventKey:     key,
		UserID:       u.UserID,
		EventType:    string(subscriptioneventtype.AUTO_RENEW_UPDATED),
		FromPlanCode: &fromPlan,
		FromTierRank: &fromTier,
		ToPlanCode:   &u.CurrentPlanCode,
		ToTierRank:   &u.CurrentTierRank,
		OccurredAt:   at,
		EffectiveAt:  at,
	}

	return event, nil
}

func (u *UserSubscription) expiryFor(plan *SubscriptionPlan, base time.Time) time.Time {
	if plan.DurationDays == nil {
		return base
	}
	return u.addProductDays(base, *plan.DurationDays, base.Location())
}

// ResolveExpiryFor decides the new expiry when a plan change EXTENDS an existing
// period rather than starting one (FR-051..FR-053).
//
// 024 T153: the comment here used to claim this was "THE expiry rule for every path
// that changes a plan", and that FR-072 requires admin grant, auto-renewal and
// purchase to share one rule. Neither is true, and an overstated comment is worse
// than none: it told the next reader that the other paths had been migrated, so
// nobody looked.
//
// What actually exists today:
//
//   - THIS function has one production caller: the shared apply workflow
//     (`internal_subscription_apply_workflow.go:56`), which admin grant, purchase
//     settlement and default provisioning all go through.
//   - `Activate` and `Grant` in this same file do NOT call it. Each computes
//     `expiryFor(plan, at)` itself, because both start a period from `at` and
//     neither stacks.
//   - `state_machine.go` carries its own `AddProductDays` copy of the same
//     arithmetic.
//   - `renewal_payment.go` reads `DurationDays` without going through any of it.
//
// So it is the shared rule for the path that can extend, and one of several rules
// for the rest. Consolidating the others is real work with real behavioural risk -
// `Activate` deliberately errors when a finite plan has no duration, where
// `ResolveExpiryFor` returns a nil expiry - so it is left as a known gap rather
// than done quietly. What matters for now is that the comment states which is
// which.
//
// The rule itself:
//   - A plan with no duration has no expiry at all.
//   - The SAME plan, still unexpired: the period stacks onto the existing expiry.
//     That is FR-051 and FR-052 - a second grant adds to what the customer already
//     has rather than throwing it away.
//   - Anything else starts fresh from `now`: a different plan, or a period that
//     has already run out. FR-053 requires exactly this for both, and the "has
//     already run out" half is why the unexpired test is on the expiry rather than
//     on plan identity alone.
//
// `isNewSub` is deliberately absent: a row that has never been provisioned has a
// nil expiry, which fails the unexpired test on its own.
//
// 027/D-08: this is the rule for a PAID period being extended, and it currently has
// no production caller — purchase settlement stacks through `ApplyTransition`. The
// administrative paths used to call it too, which is what made a re-grant add 30
// days to a customer who had not paid for them; they now use
// `ResolveAdminExpiryFor`.
func (u *UserSubscription) ResolveExpiryFor(plan *SubscriptionPlan, now time.Time) *time.Time {
	if plan.DurationDays == nil {
		return nil
	}
	if u.ExpiresAt != nil && u.ExpiresAt.After(now) && u.SubscriptionPlanID == plan.ID {
		stacked := u.expiryFor(plan, *u.ExpiresAt)
		return &stacked
	}
	fresh := u.expiryFor(plan, now)
	return &fresh
}

// ResolveAdminExpiryFor is the expiry for an ADMINISTRATIVE grant (027/D-08).
//
// An administrative grant always starts a fresh period at `now`, even when the
// account is already on this plan with time left. Nobody paid for the extension, so
// stacking it onto the running term would hand out days that were never bought, and
// it left the previous period as the grant window the AI service reads — which is
// how a re-granted account ended up with a `future` grant record and degraded AI
// access for the rest of its paid term.
//
// This REPLACES the remaining term rather than adding to it: 20 days left plus a
// 30-day administrative grant is 30 days from now, not 50. Callers that extend a
// period the customer paid for must use ResolveExpiryFor instead.
func (u *UserSubscription) ResolveAdminExpiryFor(plan *SubscriptionPlan, now time.Time) *time.Time {
	if plan.DurationDays == nil {
		return nil
	}
	fresh := u.expiryFor(plan, now)
	return &fresh
}

func (u *UserSubscription) addProductDays(base time.Time, days int, location *time.Location) time.Time {
	if location == nil {
		location = time.UTC
	}
	local := base.In(location)
	return local.AddDate(0, 0, days)
}

func (u *UserSubscription) clearFallbackFields() {
	u.FallbackPlanID = nil
	u.FallbackPlanCode = nil
	u.FallbackTierRank = nil
	u.FallbackPlanKind = nil
	u.FallbackBenefitSnapshot = nil
}
