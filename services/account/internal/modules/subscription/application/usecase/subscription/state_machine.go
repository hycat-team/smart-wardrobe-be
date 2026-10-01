package subscription

import (
	"encoding/json"
	"fmt"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/benefitresolution"
)

type TransitionContext string

const (
	CreateValidation  TransitionContext = "CREATE_VALIDATION"
	PaymentCompletion TransitionContext = "PAYMENT_COMPLETION"
)

type SubscriptionTransition string

const (
	ActivateFinite              SubscriptionTransition = "ACTIVATE_FINITE"
	ActivateLifetime            SubscriptionTransition = "ACTIVATE_LIFETIME"
	ExtendFinite                SubscriptionTransition = "EXTEND_FINITE"
	UpgradeFinite               SubscriptionTransition = "UPGRADE_FINITE"
	OverlayLifetimeWithFinite   SubscriptionTransition = "OVERLAY_LIFETIME_WITH_FINITE"
	UpgradeLifetime             SubscriptionTransition = "UPGRADE_LIFETIME"
	CreditWalletSameLifetime    SubscriptionTransition = "CREDIT_WALLET_SAME_LIFETIME"
	CreditWalletLowerTier       SubscriptionTransition = "CREDIT_WALLET_LOWER_TIER"
	RejectSameLifetime          SubscriptionTransition = "REJECT_SAME_LIFETIME"
	RejectDowngradeFromLifetime SubscriptionTransition = "REJECT_DOWNGRADE_FROM_LIFETIME"
	RejectDowngradeFromFinite   SubscriptionTransition = "REJECT_DOWNGRADE_FROM_FINITE"
)

type EffectiveSubscription struct {
	PlanCode        string
	TierRank        int
	PlanKind        plankind.PlanKind
	BenefitSnapshot entities.JSONDocument
	StartedAt       time.Time
	ExpiresAt       *time.Time
}

type PaymentSnapshot struct {
	PlanID          string
	PlanCode        string
	TierRank        int
	PlanKind        plankind.PlanKind
	DurationDays    *int
	BenefitSnapshot entities.JSONDocument
}

type BenefitResultSnapshot struct {
	Resolution      benefitresolution.BenefitResolution `json:"resolution"`
	BeforePlanCode  *string                             `json:"beforePlanCode,omitempty"`
	BeforeTierRank  *int                                `json:"beforeTierRank,omitempty"`
	BeforeExpiresAt *time.Time                          `json:"beforeExpiresAt,omitempty"`
	AfterPlanCode   *string                             `json:"afterPlanCode,omitempty"`
	AfterTierRank   *int                                `json:"afterTierRank,omitempty"`
	AfterExpiresAt  *time.Time                          `json:"afterExpiresAt,omitempty"`
}

func ResolveEffectiveSubscription(sub *entities.UserSubscription, free *entities.SubscriptionPlan, at time.Time) (EffectiveSubscription, error) {
	if sub == nil || sub.CurrentPlanKind == plankind.DefaultFree {
		if free == nil {
			return EffectiveSubscription{}, fmt.Errorf("canonical default-free plan is missing")
		}
		return EffectiveSubscription{PlanCode: free.Slug, TierRank: free.TierRank, PlanKind: plankind.DefaultFree}, nil
	}
	if sub.CurrentPlanKind == plankind.Lifetime {
		if sub.ExpiresAt != nil {
			return EffectiveSubscription{}, fmt.Errorf("lifetime subscription has expiration")
		}
		return effectiveCurrent(sub), nil
	}
	if sub.CurrentPlanKind != plankind.Finite || sub.ExpiresAt == nil {
		return EffectiveSubscription{}, fmt.Errorf("invalid finite subscription projection")
	}
	if sub.ExpiresAt.After(at) {
		return effectiveCurrent(sub), nil
	}
	if sub.FallbackPlanID != nil {
		if sub.FallbackPlanKind == nil || *sub.FallbackPlanKind != plankind.Lifetime || sub.FallbackPlanCode == nil || sub.FallbackTierRank == nil {
			return EffectiveSubscription{}, fmt.Errorf("invalid lifetime fallback projection")
		}
		return EffectiveSubscription{PlanCode: *sub.FallbackPlanCode, TierRank: *sub.FallbackTierRank, PlanKind: plankind.Lifetime, BenefitSnapshot: sub.FallbackBenefitSnapshot, StartedAt: *sub.ExpiresAt}, nil
	}
	if free == nil {
		return EffectiveSubscription{}, fmt.Errorf("canonical default-free plan is missing")
	}
	return EffectiveSubscription{PlanCode: free.Slug, TierRank: free.TierRank, PlanKind: plankind.DefaultFree}, nil
}

func effectiveCurrent(sub *entities.UserSubscription) EffectiveSubscription {
	return EffectiveSubscription{PlanCode: sub.CurrentPlanCode, TierRank: sub.CurrentTierRank, PlanKind: sub.CurrentPlanKind, BenefitSnapshot: sub.CurrentBenefitSnapshot, StartedAt: sub.StartedAt, ExpiresAt: sub.ExpiresAt}
}

func EvaluateSubscriptionTransition(ctx TransitionContext, current EffectiveSubscription, payment PaymentSnapshot) SubscriptionTransition {
	if current.PlanKind == plankind.DefaultFree {
		if payment.PlanKind == plankind.Lifetime {
			return ActivateLifetime
		}
		return ActivateFinite
	}
	if current.PlanKind == plankind.Lifetime {
		if payment.TierRank == current.TierRank {
			if ctx == PaymentCompletion {
				return CreditWalletSameLifetime
			}
			return RejectSameLifetime
		}
		if payment.TierRank < current.TierRank {
			if ctx == PaymentCompletion {
				return CreditWalletLowerTier
			}
			return RejectDowngradeFromLifetime
		}
		if payment.PlanKind == plankind.Lifetime {
			return UpgradeLifetime
		}
		return OverlayLifetimeWithFinite
	}
	if payment.TierRank < current.TierRank {
		if ctx == PaymentCompletion {
			return CreditWalletLowerTier
		}
		return RejectDowngradeFromFinite
	}
	if payment.TierRank == current.TierRank && payment.PlanKind == plankind.Finite {
		return ExtendFinite
	}
	if payment.PlanKind == plankind.Lifetime {
		return UpgradeLifetime
	}
	return UpgradeFinite
}

// 024 T178: `RestoreLifetimeFallbackIfExpired` and `expiryFor` are GONE, and this is
// the note explaining why.
//
// RestoreLifetimeFallbackIfExpired MUTATED the subscription in place to swap a lapsed
// finite plan back to its lifetime fallback. No production code called it, so the
// behaviour it implements never ran; `ResolveEffectiveSubscription` is what actually
// decides which plan an account is on, and it does not write. A mutation nobody performs
// is worse than absent code, because the test that exercised it looked like coverage of
// a working behaviour.
//
// It is also worth naming WHY it is safe to remove rather than wire up: a read path that
// silently rewrote a subscription would be a write hidden inside a read, and it would
// bump `version` on a row an administrator may be holding. If fallback restoration is
// ever needed, it belongs in the same transaction as whatever decided to restore it.
//
// The private `expiryFor` went with it: no caller, and a duplicate of the entity's own
// `u.expiryFor`, which is the one the account methods use. Two expiry calculations in a
// codebase is how they drift apart — and FR-072 requires them to be ONE rule.
//
// The test that covered RestoreLifetimeFallbackIfExpired is removed too, rather than
// left asserting a function that no longer exists.

func ApplyTransition(sub *entities.UserSubscription, plan *entities.SubscriptionPlan, snapshot entities.JSONDocument, transition SubscriptionTransition, at time.Time) (benefitresolution.BenefitResolution, *entities.UserSubscriptionEvent, error) {
	beforeCode, beforeTier, beforeExpiry := sub.CurrentPlanCode, sub.CurrentTierRank, sub.ExpiresAt
	var event *entities.UserSubscriptionEvent
	var err error
	resolution := benefitresolution.SubscriptionActivated

	switch transition {
	case ActivateFinite, ActivateLifetime:
		event, err = sub.Activate(plan, snapshot, "", at)
		resolution = benefitresolution.SubscriptionActivated
	case ExtendFinite:
		event, err = sub.Extend(plan, snapshot, "", at)
		resolution = benefitresolution.SubscriptionExtended
	case UpgradeFinite:
		event, err = sub.Upgrade(plan, snapshot, "", at)
		resolution = benefitresolution.SubscriptionUpgraded
	case OverlayLifetimeWithFinite:
		event, err = sub.OverlayLifetime(plan, snapshot, "", at)
		resolution = benefitresolution.LifetimeOverlaidByFinite
	case UpgradeLifetime:
		event, err = sub.Upgrade(plan, snapshot, "", at)
		resolution = benefitresolution.LifetimeReplaced
	default:
		return "", nil, fmt.Errorf("transition %s does not mutate subscription", transition)
	}

	if err != nil {
		return "", nil, err
	}

	afterCode, afterTier := sub.CurrentPlanCode, sub.CurrentTierRank
	b, _ := json.Marshal(BenefitResultSnapshot{
		Resolution:      resolution,
		BeforePlanCode:  &beforeCode,
		BeforeTierRank:  &beforeTier,
		BeforeExpiresAt: beforeExpiry,
		AfterPlanCode:   &afterCode,
		AfterTierRank:   &afterTier,
		AfterExpiresAt:  sub.ExpiresAt,
	})
	event.Metadata = entities.JSONDocument(b)

	return resolution, event, nil
}

// 024 T178: the private `expiryFor` that used to sit here is removed — see the note
// above. The live calculation is the entity's own `u.expiryFor`.

func AddProductDays(base time.Time, days int, location *time.Location) time.Time {
	if location == nil {
		location = time.UTC
	}
	local := base.In(location)
	return local.AddDate(0, 0, days)
}
