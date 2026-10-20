package subscription

import (
	"context"
	"errors"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure/persistence"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"

	"github.com/google/uuid"
)

// 031 T029 — idempotency and rollback for the interval write path.
//
// The publisher's `event_key` early return is what makes a retry cheap: the
// recorder is reached only after a NEW history row is inserted (research R9).
// The transaction is what keeps a recorder failure from leaving an orphaned
// interval: the event insert and the interval insert commit or roll back
// together.

// countingPlanPeriodRecorder counts how many times the recorder was invoked.
type countingPlanPeriodRecorder struct {
	calls int
}

func (c *countingPlanPeriodRecorder) Record(
	context.Context,
	*entities.UserSubscriptionEvent,
	*entities.UserSubscription,
	*entities.SubscriptionPlan,
) error {
	c.calls++
	return nil
}

// TestPlanPeriodRecorderPublisherRetryAddsOneInterval proves the unit-level
// contract without a database: the same event_key is inserted once, published
// once, and recorded once, even when the retry uses a fresh event value.
func TestPlanPeriodRecorderPublisherRetryAddsOneInterval(t *testing.T) {
	eventRepo := &grantTxEventRepo{}
	broker := &fakeRenewalPublisher{}
	recorder := &countingPlanPeriodRecorder{}
	publisher := NewPlanChangePublisher(eventRepo, broker, recorder)

	plan := renewalTestPlan()
	userID := uuid.New()
	startedAt := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	expiresAt := startedAt.Add(30 * 24 * time.Hour)
	sub := &entities.UserSubscription{
		UserID:             userID,
		SubscriptionPlanID: plan.ID,
		CurrentPlanCode:    plan.Slug,
		CurrentTierRank:    plan.TierRank,
		CurrentPlanKind:    plan.PlanKind,
		StartedAt:          startedAt,
		ExpiresAt:          &expiresAt,
	}
	planCode := plan.Slug
	tierRank := plan.TierRank
	event := &entities.UserSubscriptionEvent{
		EventKey:    "031-t029:retry",
		UserID:      userID,
		EventType:   string(subscriptioneventtype.ACTIVATED),
		ToPlanCode:  &planCode,
		ToTierRank:  &tierRank,
		OccurredAt:  startedAt,
		EffectiveAt: startedAt,
	}

	if _, err := publisher.PublishPlanChange(context.Background(), event, sub, plan); err != nil {
		t.Fatalf("first publish failed: %v", err)
	}

	// A retry is a new value with the same key, not the same pointer: that is
	// how a second request reaches the publisher in production.
	retry := *event
	if _, err := publisher.PublishPlanChange(context.Background(), &retry, sub, plan); err != nil {
		t.Fatalf("retry failed: %v", err)
	}

	if eventRepo.creates != 1 {
		t.Errorf("event inserts = %d, want 1: event_key is UNIQUE and the second call must return the stored row", eventRepo.creates)
	}
	if len(broker.events) != 1 {
		t.Errorf("published messages = %d, want 1: a retry must not re-publish", len(broker.events))
	}
	if recorder.calls != 1 {
		t.Errorf("recorder calls = %d, want 1: the interval write must happen only for a newly inserted event", recorder.calls)
	}
}

// failAfterPlanPeriodRecord is a recorder decorator that performs the real
// write and THEN fails. It is the only way to prove both rows roll back: a
// recorder that fails before writing would make the interval assertion
// vacuous.
type failAfterPlanPeriodRecord struct {
	inner    PlanPeriodRecorder
	err      error
	recorded bool
}

func (f *failAfterPlanPeriodRecord) Record(
	ctx context.Context,
	event *entities.UserSubscriptionEvent,
	sub *entities.UserSubscription,
	plan *entities.SubscriptionPlan,
) error {
	if err := f.inner.Record(ctx, event, sub, plan); err != nil {
		return err
	}
	f.recorded = true
	return f.err
}

// TestIntegrationPlanPeriodRecorderPublishIsIdempotent runs the real publisher,
// real event repository and real recorder on Postgres. A repeated event_key
// must leave exactly one history row and one interval.
func TestIntegrationPlanPeriodRecorderPublishIsIdempotent(t *testing.T) {
	fixture := newPlanRecorderFixture(t)
	premium := fixture.plan("premium", "Premium", plankind.Finite, 1)

	eventRepo := persistence.NewUserSubscriptionEventRepository(fixture.db)
	broker := &fakeRenewalPublisher{}
	publisher := NewPlanChangePublisher(eventRepo, broker, fixture.recorder)

	startedAt := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	expiresAt := startedAt.Add(30 * 24 * time.Hour)
	sub := planRecorderSub(fixture.userID, premium, startedAt, &expiresAt)
	event := &entities.UserSubscriptionEvent{
		EventKey:    "031-recorder:" + fixture.userID.String() + ":idempotent",
		UserID:      fixture.userID,
		EventType:   string(subscriptioneventtype.ACTIVATED),
		OccurredAt:  startedAt,
		EffectiveAt: startedAt,
	}

	publish := func(ev *entities.UserSubscriptionEvent) {
		t.Helper()
		err := fixture.uow.Execute(context.Background(), func(txCtx context.Context) error {
			_, publishErr := publisher.PublishPlanChange(txCtx, ev, sub, premium)
			return publishErr
		})
		if err != nil {
			t.Fatalf("publish %s: %v", ev.EventKey, err)
		}
	}

	publish(event)
	retry := *event
	publish(&retry)

	if count := planRecorderCount(t, fixture, `SELECT COUNT(*) FROM subscription.user_subscription_events WHERE event_key = ?`, event.EventKey); count != 1 {
		t.Errorf("event rows = %d, want 1: the second publish must return the stored event", count)
	}
	if len(broker.events) != 1 {
		t.Errorf("published messages = %d, want 1: a retry must not re-publish", len(broker.events))
	}
	periods := planPeriodHistory(t, fixture.periodRepo, fixture.ctx, fixture.userID)
	if len(periods) != 1 {
		t.Fatalf("interval rows = %d, want 1: a retry must not add a second interval (%s)",
			len(periods), planPeriodSummary(periods))
	}
	planRecorderAssertInterval(t, periods[0], premium.Slug, startedAt, &expiresAt)
}

// TestIntegrationPlanPeriodRecorderFailureRollsBackBothRows proves the failure
// path: the event row is inserted, the interval is inserted, the recorder then
// returns an error, and the unit of work rolls BOTH rows back.
func TestIntegrationPlanPeriodRecorderFailureRollsBackBothRows(t *testing.T) {
	fixture := newPlanRecorderFixture(t)
	premium := fixture.plan("premium", "Premium", plankind.Finite, 1)

	eventRepo := persistence.NewUserSubscriptionEventRepository(fixture.db)
	failure := errors.New("injected recorder failure after write")
	recorder := &failAfterPlanPeriodRecord{inner: fixture.recorder, err: failure}
	publisher := NewPlanChangePublisher(eventRepo, &fakeRenewalPublisher{}, recorder)

	startedAt := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	expiresAt := startedAt.Add(30 * 24 * time.Hour)
	sub := planRecorderSub(fixture.userID, premium, startedAt, &expiresAt)
	event := &entities.UserSubscriptionEvent{
		EventKey:    "031-recorder:" + fixture.userID.String() + ":rollback",
		UserID:      fixture.userID,
		EventType:   string(subscriptioneventtype.ACTIVATED),
		OccurredAt:  startedAt,
		EffectiveAt: startedAt,
	}

	err := fixture.uow.Execute(context.Background(), func(txCtx context.Context) error {
		_, publishErr := publisher.PublishPlanChange(txCtx, event, sub, premium)
		return publishErr
	})
	if !errors.Is(err, failure) {
		t.Fatalf("unit of work error = %v, want the injected recorder failure", err)
	}
	if !recorder.recorded {
		t.Fatal("the recorder did not write before the injected failure; the rollback assertions would be vacuous")
	}

	eventCount := planRecorderCount(t, fixture, `SELECT COUNT(*) FROM subscription.user_subscription_events WHERE event_key = ?`, event.EventKey)
	if eventCount != 0 {
		t.Errorf("event rows after rollback = %d, want 0", eventCount)
	}
	periodCount := planRecorderCount(t, fixture, `SELECT COUNT(*) FROM subscription.user_plan_periods WHERE user_id = ?`, fixture.userID)
	if periodCount != 0 {
		t.Errorf("interval rows after rollback = %d, want 0", periodCount)
	}
}

func planRecorderCount(t *testing.T, fixture *planRecorderFixture, query string, args ...any) int64 {
	t.Helper()
	var count int64
	if err := fixture.db.Raw(query, args...).Scan(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return count
}
