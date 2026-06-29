package entities

import (
	"fmt"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/plankind"
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
		EventType:    "ACTIVATED",
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
		EventType:    "EXTENDED",
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
		EventType:    "UPGRADED",
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

	u.FallbackPlanID = &u.SubscriptionPlanID
	u.FallbackPlanCode = &u.CurrentPlanCode
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
		EventType:    "OVERLAID",
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
		EventType:    "LIFETIME_FALLBACK_RESTORED",
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
		EventType:    "SUBSCRIPTION_DOWNGRADED_TO_FREE",
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
		EventType:    "AUTO_RENEW_UPDATED",
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
