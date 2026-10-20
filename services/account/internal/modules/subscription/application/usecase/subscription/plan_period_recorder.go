package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"
)

// adminRestoreSource is the event metadata value the administrative repair
// path sets (restore_policy_grants_uc.go). Such an event re-publishes the plan
// for the AI grant but does NOT change the subscription window.
const adminRestoreSource = "admin_restore"

// PlanPeriodRecorder appends the user's validity interval when a plan change is
// published (031 D1/R2).
//
// It runs inside the caller's transaction, after the history event insert
// succeeded, so the event and the interval commit or roll back together. The
// publisher's existing `event_key` early return means a retry never reaches
// this recorder, which is the idempotency guarantee (FR-006, research.md R9).
type PlanPeriodRecorder interface {
	Record(
		ctx context.Context,
		event *entities.UserSubscriptionEvent,
		sub *entities.UserSubscription,
		plan *entities.SubscriptionPlan,
	) error
}

type planPeriodRecorder struct {
	repo repositories.IUserPlanPeriodRepository
}

func NewPlanPeriodRecorder(repo repositories.IUserPlanPeriodRepository) PlanPeriodRecorder {
	return &planPeriodRecorder{repo: repo}
}

// Record applies the transition table in 031 data-model §3.
//
// Order matters. `source=admin_restore` is checked BEFORE any branch that reads
// the current interval: the repair path does not lock the subscription row, and
// a concurrent purchase may already have written a newer interval. Treating the
// repair as an absolute no-op is what prevents it from overwriting that row
// (R5). `AUTO_RENEW_UPDATED` is a no-op for the same reason: the plan and its
// window did not change.
func (r *planPeriodRecorder) Record(
	ctx context.Context,
	event *entities.UserSubscriptionEvent,
	sub *entities.UserSubscription,
	plan *entities.SubscriptionPlan,
) error {
	if event == nil {
		return nil
	}
	if isAdminRestoreEvent(event) {
		return nil
	}
	if sub == nil || plan == nil {
		return fmt.Errorf("plan period recorder: subscription and plan are required")
	}

	switch subscriptioneventtype.Type(event.EventType) {
	case subscriptioneventtype.AUTO_RENEW_UPDATED:
		return nil
	case subscriptioneventtype.EXTENDED:
		return r.recordExtended(ctx, event, sub, plan)
	case subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE,
		subscriptioneventtype.LIFETIME_FALLBACK_RESTORED:
		return r.recordFallback(ctx, event, sub, plan)
	case subscriptioneventtype.ACTIVATED,
		subscriptioneventtype.UPGRADED,
		subscriptioneventtype.OVERLAID:
		return r.recordSwitch(ctx, event, sub, plan)
	default:
		// The shared event-type set is closed (subscriptioneventtype.All); an
		// unknown value cannot be produced by a production path. Recording
		// nothing is safer than inventing a window for a type this recorder
		// does not understand.
		return nil
	}
}

// recordSwitch handles ACTIVATED / UPGRADED / OVERLAID.
func (r *planPeriodRecorder) recordSwitch(
	ctx context.Context,
	event *entities.UserSubscriptionEvent,
	sub *entities.UserSubscription,
	plan *entities.SubscriptionPlan,
) error {
	current, err := r.repo.GetCurrentByUserIDWithLock(ctx, sub.UserID)
	if err != nil {
		return fmt.Errorf("read current plan period: %w", err)
	}

	if current == nil {
		return r.repo.Create(ctx, buildPeriod(event, sub, plan, event.EffectiveAt, sub.ExpiresAt))
	}

	// FR-003 / data-model §3: an ACTIVATED for the plan already current (the
	// admin restore path) changes neither the plan nor the window. Without this
	// guard the close below would land at the interval's own start and violate
	// chk_user_plan_periods_valid_range.
	if subscriptioneventtype.Type(event.EventType) == subscriptioneventtype.ACTIVATED && current.PlanCode == plan.Slug {
		return nil
	}

	closeAt := closeInstant(current, event.EffectiveAt)
	if !closeAt.After(current.ValidFrom) {
		// Same-instant (or clock skew): replace the current row instead of
		// closing it and creating a zero-length interval. This also covers
		// provisioning and the first grant in one transaction (R5).
		current.SubscriptionPlanID = plan.ID
		current.PlanCode = plan.Slug
		current.PlanName = plan.Name
		current.TierRank = plan.TierRank
		current.PlanKind = plan.PlanKind
		current.ValidFrom = laterOf(event.EffectiveAt, current.ValidFrom)
		current.ValidTo = sub.ExpiresAt
		current.SourceEventKey = sourceEventKey(event.EventKey)
		current.UpdatedAt = time.Now().UTC()
		if err := r.repo.Update(ctx, current); err != nil {
			return fmt.Errorf("replace current plan period: %w", err)
		}
		return nil
	}

	if err := r.repo.CloseCurrent(ctx, current.ID, closeAt); err != nil {
		return fmt.Errorf("close current plan period: %w", err)
	}

	newFrom := laterOf(event.EffectiveAt, current.ValidFrom)
	return r.repo.Create(ctx, buildPeriod(event, sub, plan, newFrom, sub.ExpiresAt))
}

// recordExtended handles a renewal of the same plan: it only moves valid_to
// (FR-004 / R4). A missing current row is the legacy case - insert the current
// window rather than dropping the user from the history.
func (r *planPeriodRecorder) recordExtended(
	ctx context.Context,
	event *entities.UserSubscriptionEvent,
	sub *entities.UserSubscription,
	plan *entities.SubscriptionPlan,
) error {
	current, err := r.repo.GetCurrentByUserIDWithLock(ctx, sub.UserID)
	if err != nil {
		return fmt.Errorf("read current plan period: %w", err)
	}

	if current == nil {
		return r.repo.Create(ctx, buildPeriod(event, sub, plan, sub.StartedAt, sub.ExpiresAt))
	}
	if err := r.repo.UpdateValidTo(ctx, current.ID, sub.ExpiresAt); err != nil {
		return fmt.Errorf("extend current plan period: %w", err)
	}
	return nil
}

// recordFallback handles SUBSCRIPTION_DOWNGRADED_TO_FREE and
// LIFETIME_FALLBACK_RESTORED: close the previous interval at the contractual
// boundary and open the free/lifetime interval at that same instant (FR-005 /
// R5). `closeAt` uses min(current.valid_to, effective) so a late worker does not
// extend the paid plan past the moment it actually ended.
func (r *planPeriodRecorder) recordFallback(
	ctx context.Context,
	event *entities.UserSubscriptionEvent,
	sub *entities.UserSubscription,
	plan *entities.SubscriptionPlan,
) error {
	current, err := r.repo.GetCurrentByUserIDWithLock(ctx, sub.UserID)
	if err != nil {
		return fmt.Errorf("read current plan period: %w", err)
	}

	closeAt := event.EffectiveAt
	if current != nil {
		closeAt = closeInstant(current, event.EffectiveAt)
		if err := r.repo.CloseCurrent(ctx, current.ID, closeAt); err != nil {
			return fmt.Errorf("close plan period on fallback: %w", err)
		}
	}

	// Free and lifetime intervals are open-ended by definition (plan kinds 0
	// and 2), so the new row carries no upper bound.
	return r.repo.Create(ctx, buildPeriod(event, sub, plan, closeAt, nil))
}

// buildPeriod snapshots the plan attributes into a new interval row.
func buildPeriod(
	event *entities.UserSubscriptionEvent,
	sub *entities.UserSubscription,
	plan *entities.SubscriptionPlan,
	validFrom time.Time,
	validTo *time.Time,
) *entities.UserPlanPeriod {
	return &entities.UserPlanPeriod{
		UserID:             sub.UserID,
		SubscriptionPlanID: plan.ID,
		PlanCode:           plan.Slug,
		PlanName:           plan.Name,
		TierRank:           plan.TierRank,
		PlanKind:           plan.PlanKind,
		ValidFrom:          validFrom,
		ValidTo:            validTo,
		SourceEventKey:     sourceEventKey(event.EventKey),
	}
}

// closeInstant is min(current.valid_to, effective) when the interval already
// has an end that is earlier than the event, otherwise the event instant.
func closeInstant(current *entities.UserPlanPeriod, effective time.Time) time.Time {
	if current.ValidTo != nil && current.ValidTo.Before(effective) {
		return *current.ValidTo
	}
	return effective
}

// laterOf returns the later of two instants; it clamps a new interval's start
// so clock skew can never produce a zero/negative interval.
func laterOf(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// sourceEventKey keeps the event key nullable, as the column allows.
func sourceEventKey(key string) *string {
	if key == "" {
		return nil
	}
	copied := key
	return &copied
}

// isAdminRestoreEvent reads the one metadata flag that overrides every
// transition. A malformed document simply does not carry the flag; it must not
// turn a plan change into an error at the storage layer.
func isAdminRestoreEvent(event *entities.UserSubscriptionEvent) bool {
	if len(event.Metadata) == 0 {
		return false
	}
	var metadata struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal(event.Metadata, &metadata); err != nil {
		return false
	}
	return metadata.Source == adminRestoreSource
}
