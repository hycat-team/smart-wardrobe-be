package subscription

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure/persistence"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"
	sharedrepositories "smart-wardrobe-be/shared/domain/repositories"
	shareddb "smart-wardrobe-be/shared/infrastructure/db"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// 031 T028/T030 — Postgres coverage for the recorder's lifecycle and
// concurrency guarantees (FR-002/FR-004/FR-005/FR-006, data-model §3).
//
// These tests run the real recorder against the real repository inside the real
// unit of work, so what is asserted is the persisted validity history - row
// count, boundaries, contiguity and the absence of overlaps - not the branch
// the recorder took.

// planRecorderFixture owns one fixture user and the plans its subtests create.
// Every plan slug carries the user id so parallel runs and re-runs cannot
// collide.
type planRecorderFixture struct {
	t          *testing.T
	db         *gorm.DB
	ctx        context.Context
	uow        sharedrepositories.IUnitOfWork
	recorder   PlanPeriodRecorder
	periodRepo repositories.IUserPlanPeriodRepository
	planRepo   repositories.ISubscriptionPlanRepository
	userID     uuid.UUID
	planIDs    []uuid.UUID
}

func newPlanRecorderFixture(t *testing.T) *planRecorderFixture {
	t.Helper()

	dbConn := openSubscriptionTestDB(t)
	userID := uuid.New()
	ensureSubscriptionUser(t, dbConn, userID, "plan-period-recorder")

	fixture := &planRecorderFixture{
		t:          t,
		db:         dbConn,
		ctx:        context.Background(),
		uow:        shareddb.NewGormUnitOfWork(dbConn),
		periodRepo: persistence.NewUserPlanPeriodRepository(dbConn),
		planRepo:   persistence.NewSubscriptionPlanRepository(dbConn),
		userID:     userID,
	}
	fixture.recorder = NewPlanPeriodRecorder(fixture.periodRepo)

	t.Cleanup(func() {
		if err := dbConn.Exec(`DELETE FROM subscription.user_plan_periods WHERE user_id = ?`, userID).Error; err != nil {
			t.Errorf("cleanup plan periods: %v", err)
		}
		if err := dbConn.Exec(`DELETE FROM subscription.user_subscription_events WHERE user_id = ?`, userID).Error; err != nil {
			t.Errorf("cleanup subscription events: %v", err)
		}
		for _, planID := range fixture.planIDs {
			if err := dbConn.Exec(`DELETE FROM subscription.subscription_plans WHERE id = ?`, planID).Error; err != nil {
				t.Errorf("cleanup plan %s: %v", planID, err)
			}
		}
		if err := dbConn.Exec(`DELETE FROM identity.users WHERE id = ?`, userID).Error; err != nil {
			t.Errorf("cleanup fixture user: %v", err)
		}
	})

	return fixture
}

func (f *planRecorderFixture) plan(label, name string, kind plankind.PlanKind, tierRank int) *entities.SubscriptionPlan {
	f.t.Helper()

	// The schema allows only ONE active default-free plan
	// (ux_single_active_default_free_plan), seeded by database/init/03-seed.sql.
	// Reuse it instead of inserting a duplicate; the recorder's snapshot fields
	// are still exercised because the row carries a slug, name, rank and kind.
	if kind == plankind.DefaultFree {
		plan, err := f.planRepo.GetDefaultPlan(f.ctx)
		if err != nil || plan == nil {
			f.t.Fatalf("default free plan missing from the test database (apply database/init/03-seed.sql): %v", err)
		}
		return plan
	}

	plan := &entities.SubscriptionPlan{
		Slug:               fmt.Sprintf("031-recorder-%s-%s", label, f.userID.String()),
		Name:               name,
		Price:              decimal.Zero,
		MaxWardrobeItems:   100,
		MaxOutfits:         30,
		AiOutfitDailyQuota: 10,
		AiChatDailyQuota:   20,
		IsActive:           true,
		PlanKind:           kind,
		TierRank:           tierRank,
	}
	plan.ID = uuid.New()

	if err := f.planRepo.Create(f.ctx, plan); err != nil {
		f.t.Fatalf("seed plan %q: %v", plan.Slug, err)
	}
	f.planIDs = append(f.planIDs, plan.ID)
	return plan
}

func (f *planRecorderFixture) record(
	eventType subscriptioneventtype.Type,
	keySuffix string,
	effectiveAt time.Time,
	sub *entities.UserSubscription,
	plan *entities.SubscriptionPlan,
) error {
	event := &entities.UserSubscriptionEvent{
		EventKey:    fmt.Sprintf("031-recorder:%s:%s", f.userID.String(), keySuffix),
		UserID:      f.userID,
		EventType:   string(eventType),
		OccurredAt:  effectiveAt,
		EffectiveAt: effectiveAt,
	}
	return f.uow.Execute(context.Background(), func(txCtx context.Context) error {
		return f.recorder.Record(txCtx, event, sub, plan)
	})
}

func (f *planRecorderFixture) mustRecord(
	eventType subscriptioneventtype.Type,
	keySuffix string,
	effectiveAt time.Time,
	sub *entities.UserSubscription,
	plan *entities.SubscriptionPlan,
) {
	f.t.Helper()
	if err := f.record(eventType, keySuffix, effectiveAt, sub, plan); err != nil {
		f.t.Fatalf("record %s: %v", eventType, err)
	}
}

func planRecorderSub(userID uuid.UUID, plan *entities.SubscriptionPlan, startedAt time.Time, expiresAt *time.Time) *entities.UserSubscription {
	return &entities.UserSubscription{
		UserID:             userID,
		SubscriptionPlanID: plan.ID,
		CurrentPlanCode:    plan.Slug,
		CurrentTierRank:    plan.TierRank,
		CurrentPlanKind:    plan.PlanKind,
		StartedAt:          startedAt,
		ExpiresAt:          expiresAt,
	}
}

func planRecorderAssertInterval(t *testing.T, period *entities.UserPlanPeriod, planCode string, validFrom time.Time, validTo *time.Time) {
	t.Helper()
	if period.PlanCode != planCode {
		t.Errorf("interval plan_code = %q, want %q", period.PlanCode, planCode)
	}
	if !period.ValidFrom.Equal(validFrom) {
		t.Errorf("interval %q valid_from = %s, want %s", planCode, period.ValidFrom, validFrom)
	}
	if validTo == nil {
		if period.ValidTo != nil {
			t.Errorf("interval %q valid_to = %s, want NULL", planCode, period.ValidTo)
		}
		return
	}
	if period.ValidTo == nil || !period.ValidTo.Equal(*validTo) {
		t.Errorf("interval %q valid_to = %v, want %s", planCode, period.ValidTo, *validTo)
	}
}

func planRecorderAssertContiguous(t *testing.T, periods []*entities.UserPlanPeriod) {
	t.Helper()
	for i := 0; i < len(periods); i++ {
		if periods[i].ValidTo != nil && !periods[i].ValidTo.After(periods[i].ValidFrom) {
			t.Errorf("interval %d is not a positive half-open range: [%s, %s)", i, periods[i].ValidFrom, *periods[i].ValidTo)
		}
		if i == 0 {
			continue
		}
		previous := periods[i-1]
		if previous.ValidTo == nil {
			t.Fatalf("interval %d follows open-ended interval %d", i, i-1)
		}
		if !periods[i].ValidFrom.Equal(*previous.ValidTo) {
			t.Errorf("interval %d starts at %s, want the previous end %s: a continuous lifecycle must not leave a gap or an overlap",
				i, periods[i].ValidFrom, *previous.ValidTo)
		}
	}
}

// TestIntegrationPlanPeriodRecorderLifecycle covers the canonical four-step
// holding (activate -> extend -> upgrade -> downgrade), the late downgrade
// boundary, and the lifetime overlay/restore pair. Every interval is asserted
// by value and the whole history by contiguity and non-overlap.
func TestIntegrationPlanPeriodRecorderLifecycle(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

	t.Run("activate extend upgrade downgrade is contiguous", func(t *testing.T) {
		fixture := newPlanRecorderFixture(t)
		premium := fixture.plan("premium", "Premium", plankind.Finite, 1)
		pro := fixture.plan("pro", "Pro", plankind.Finite, 2)
		free := fixture.plan("free", "Free", plankind.DefaultFree, 0)

		firstExpiry := t0.Add(30 * 24 * time.Hour)
		extendedExpiry := t0.Add(60 * 24 * time.Hour)
		upgradedAt := t0.Add(40 * 24 * time.Hour)
		proExpiry := t0.Add(70 * 24 * time.Hour)
		downgradedAt := t0.Add(65 * 24 * time.Hour)

		// 1. Activate: one finite paid interval.
		fixture.mustRecord(subscriptioneventtype.ACTIVATED, "lifecycle:activate", t0,
			planRecorderSub(fixture.userID, premium, t0, &firstExpiry), premium)

		// 2. Extend: valid_to moves; no new row.
		fixture.mustRecord(subscriptioneventtype.EXTENDED, "lifecycle:extend", firstExpiry,
			planRecorderSub(fixture.userID, premium, t0, &extendedExpiry), premium)

		// 3. Upgrade: the premium interval closes at the switch instant and pro
		// opens at the same instant.
		fixture.mustRecord(subscriptioneventtype.UPGRADED, "lifecycle:upgrade", upgradedAt,
			planRecorderSub(fixture.userID, pro, upgradedAt, &proExpiry), pro)

		// 4. Downgrade before the pro expiry: free opens at the event instant.
		fixture.mustRecord(subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE, "lifecycle:downgrade", downgradedAt,
			planRecorderSub(fixture.userID, free, downgradedAt, nil), free)

		periods := planPeriodHistory(t, fixture.periodRepo, fixture.ctx, fixture.userID)
		if len(periods) != 3 {
			t.Fatalf("history has %d interval(s), want 3: %s", len(periods), planPeriodSummary(periods))
		}
		planRecorderAssertInterval(t, periods[0], premium.Slug, t0, &upgradedAt)
		planRecorderAssertInterval(t, periods[1], pro.Slug, upgradedAt, &downgradedAt)
		planRecorderAssertInterval(t, periods[2], free.Slug, downgradedAt, nil)
		planRecorderAssertContiguous(t, periods)
		assertNoPlanPeriodOverlap(t, fixture.db, fixture.userID)

		current, err := fixture.periodRepo.GetCurrentByUserIDWithLock(fixture.ctx, fixture.userID)
		if err != nil {
			t.Fatalf("read current interval: %v", err)
		}
		if current == nil || current.PlanCode != free.Slug || current.ValidTo != nil {
			t.Fatalf("current interval after downgrade = %v, want the open free row", current)
		}
	})

	t.Run("late downgrade closes the paid interval at its expiry", func(t *testing.T) {
		fixture := newPlanRecorderFixture(t)
		premium := fixture.plan("premium", "Premium", plankind.Finite, 1)
		free := fixture.plan("free", "Free", plankind.DefaultFree, 0)
		expiry := t0.Add(30 * 24 * time.Hour)
		late := expiry.Add(10 * 24 * time.Hour)

		fixture.mustRecord(subscriptioneventtype.ACTIVATED, "late:activate", t0,
			planRecorderSub(fixture.userID, premium, t0, &expiry), premium)
		fixture.mustRecord(subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE, "late:downgrade", late,
			planRecorderSub(fixture.userID, free, late, nil), free)

		periods := planPeriodHistory(t, fixture.periodRepo, fixture.ctx, fixture.userID)
		if len(periods) != 2 {
			t.Fatalf("history has %d interval(s), want 2: %s", len(periods), planPeriodSummary(periods))
		}
		planRecorderAssertInterval(t, periods[0], premium.Slug, t0, &expiry)
		planRecorderAssertInterval(t, periods[1], free.Slug, expiry, nil)
		planRecorderAssertContiguous(t, periods)
		assertNoPlanPeriodOverlap(t, fixture.db, fixture.userID)
	})

	t.Run("downgrade exactly at the interval start replaces the paid row", func(t *testing.T) {
		fixture := newPlanRecorderFixture(t)
		premium := fixture.plan("premium", "Premium", plankind.Finite, 1)
		free := fixture.plan("free", "Free", plankind.DefaultFree, 0)
		expiry := t0.Add(30 * 24 * time.Hour)

		fixture.mustRecord(subscriptioneventtype.ACTIVATED, "same-instant:activate", t0,
			planRecorderSub(fixture.userID, premium, t0, &expiry), premium)

		// Clock skew (or an immediate downgrade) puts the effective instant on
		// the current interval's own start. Close+insert would write
		// valid_to == valid_from and the CHECK constraint would reject it.
		fixture.mustRecord(subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE, "same-instant:downgrade", t0,
			planRecorderSub(fixture.userID, free, t0, nil), free)

		periods := planPeriodHistory(t, fixture.periodRepo, fixture.ctx, fixture.userID)
		if len(periods) != 1 {
			t.Fatalf("history has %d interval(s), want 1 after replacement: %s", len(periods), planPeriodSummary(periods))
		}
		planRecorderAssertInterval(t, periods[0], free.Slug, t0, nil)
		assertNoPlanPeriodOverlap(t, fixture.db, fixture.userID)
	})

	t.Run("lifetime overlay then restore is contiguous", func(t *testing.T) {
		fixture := newPlanRecorderFixture(t)
		lifetime := fixture.plan("lifetime", "Lifetime", plankind.Lifetime, 9)
		premium := fixture.plan("premium", "Premium", plankind.Finite, 1)
		overlayAt := t0.Add(10 * 24 * time.Hour)
		overlayExpiry := overlayAt.Add(30 * 24 * time.Hour)
		restoreAt := overlayExpiry

		fixture.mustRecord(subscriptioneventtype.ACTIVATED, "overlay:lifetime", t0,
			planRecorderSub(fixture.userID, lifetime, t0, nil), lifetime)
		fixture.mustRecord(subscriptioneventtype.OVERLAID, "overlay:finite", overlayAt,
			planRecorderSub(fixture.userID, premium, overlayAt, &overlayExpiry), premium)
		fixture.mustRecord(subscriptioneventtype.LIFETIME_FALLBACK_RESTORED, "overlay:restore", restoreAt,
			planRecorderSub(fixture.userID, lifetime, restoreAt, nil), lifetime)

		periods := planPeriodHistory(t, fixture.periodRepo, fixture.ctx, fixture.userID)
		if len(periods) != 3 {
			t.Fatalf("history has %d interval(s), want 3: %s", len(periods), planPeriodSummary(periods))
		}
		planRecorderAssertInterval(t, periods[0], lifetime.Slug, t0, &overlayAt)
		planRecorderAssertInterval(t, periods[1], premium.Slug, overlayAt, &overlayExpiry)
		planRecorderAssertInterval(t, periods[2], lifetime.Slug, restoreAt, nil)
		planRecorderAssertContiguous(t, periods)
		assertNoPlanPeriodOverlap(t, fixture.db, fixture.userID)

		current, err := fixture.periodRepo.GetCurrentByUserIDWithLock(fixture.ctx, fixture.userID)
		if err != nil {
			t.Fatalf("read current interval: %v", err)
		}
		if current == nil || current.PlanCode != lifetime.Slug || current.ValidTo != nil {
			t.Fatalf("current interval after restore = %v, want the open lifetime row", current)
		}
	})
}

// TestIntegrationPlanPeriodRecorderConcurrentChanges is T030: two simultaneous
// changes for one user must leave exactly one current interval and no overlap.
//
// Both transactions are released together and then contend on the per-user
// advisory lock. The winner closes the old interval and writes its own row; the
// loser acquires the lock afterwards, sees the committed row and - because both
// events share the same effective instant - replaces it in place. Both changes
// succeed and the history stays contiguous, with zero overlap. The exclusion
// constraint (where installed) never has to reject anything.
func TestIntegrationPlanPeriodRecorderConcurrentChanges(t *testing.T) {
	fixture := newPlanRecorderFixture(t)
	if !planRecorderHasExclusionConstraint(t, fixture.db) {
		// T030 runs here in the exclusion variant. The fallback variant has its
		// own harness (plan_period_recorder_fallback_integration_test.go), which
		// creates a scratch database without the constraint.
		t.Skip("ex_user_plan_periods_no_overlap is absent; this database uses the partial-unique fallback")
	}

	premium := fixture.plan("premium", "Premium", plankind.Finite, 1)
	first := fixture.plan("first", "First", plankind.Finite, 2)
	second := fixture.plan("second", "Second", plankind.Finite, 3)

	t0 := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	initialExpiry := t0.Add(60 * 24 * time.Hour)
	changedAt := t0.Add(10 * 24 * time.Hour)
	changedExpiry := t0.Add(40 * 24 * time.Hour)

	fixture.mustRecord(subscriptioneventtype.ACTIVATED, "concurrent:activate", t0,
		planRecorderSub(fixture.userID, premium, t0, &initialExpiry), premium)

	candidates := []struct {
		plan  *entities.SubscriptionPlan
		sub   *entities.UserSubscription
		event *entities.UserSubscriptionEvent
		code  string
	}{
		{
			plan:  first,
			sub:   planRecorderSub(fixture.userID, first, changedAt, &changedExpiry),
			event: recorderTestEvent(fixture.userID, "031-recorder:"+fixture.userID.String()+":concurrent:first", subscriptioneventtype.UPGRADED, changedAt),
			code:  first.Slug,
		},
		{
			plan:  second,
			sub:   planRecorderSub(fixture.userID, second, changedAt, &changedExpiry),
			event: recorderTestEvent(fixture.userID, "031-recorder:"+fixture.userID.String()+":concurrent:second", subscriptioneventtype.UPGRADED, changedAt),
			code:  second.Slug,
		},
	}

	// Deterministic contention: both transactions are open and both recorders
	// have reached the advisory lock before either is allowed to proceed. After
	// release one acquires it and runs to COMMIT; the other blocks on the lock,
	// then reads the committed row and replaces it in place because both events
	// share the same effective instant.
	ready := make(chan struct{}, len(candidates))
	release := make(chan struct{})
	recorders := make([]PlanPeriodRecorder, len(candidates))
	for i := range candidates {
		recorders[i] = NewPlanPeriodRecorder(&barrierPlanPeriodRepo{
			IUserPlanPeriodRepository: fixture.periodRepo,
			ready:                     ready,
			release:                   release,
		})
	}

	results := make([]error, len(candidates))
	var wg sync.WaitGroup
	for i := range candidates {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = fixture.uow.Execute(context.Background(), func(txCtx context.Context) error {
				return recorders[index].Record(txCtx, candidates[index].event, candidates[index].sub, candidates[index].plan)
			})
		}(i)
	}

	for i := 0; i < len(candidates); i++ {
		select {
		case <-ready:
		case <-time.After(10 * time.Second):
			t.Fatal("timed out waiting for both transactions to reach the recorder")
		}
	}
	close(release)
	wg.Wait()

	for i, err := range results {
		if err != nil {
			t.Errorf("change %d failed: %v; the advisory lock must serialize the writers, not reject one", i, err)
		}
	}

	periods := planPeriodHistory(t, fixture.periodRepo, fixture.ctx, fixture.userID)
	if len(periods) != 2 {
		t.Fatalf("history has %d interval(s), want 2 (the initial holding and the last committed change): %s",
			len(periods), planPeriodSummary(periods))
	}
	planRecorderAssertInterval(t, periods[0], premium.Slug, t0, &changedAt)
	planRecorderAssertContiguous(t, periods)
	assertNoPlanPeriodOverlap(t, fixture.db, fixture.userID)

	// "One current interval" means exactly one row can be read as current:
	// the greatest valid_from. The last writer replaced the winner's row in
	// place, so the plan code is whichever transaction acquired the lock last.
	current, err := fixture.periodRepo.GetCurrentByUserIDWithLock(fixture.ctx, fixture.userID)
	if err != nil {
		t.Fatalf("read current interval after the race: %v", err)
	}
	if current == nil || !current.ValidFrom.Equal(changedAt) {
		t.Fatalf("current interval after the race = %v, want a row starting at %s", current, changedAt)
	}
	lastWriter := current.PlanCode
	if lastWriter != candidates[0].code && lastWriter != candidates[1].code {
		t.Fatalf("current plan after the race = %q, want one of the two candidates", lastWriter)
	}
	planRecorderAssertInterval(t, periods[1], lastWriter, changedAt, &changedExpiry)
	var openRows int64
	if err := fixture.db.Raw(
		`SELECT COUNT(*) FROM subscription.user_plan_periods WHERE user_id = ? AND valid_to IS NULL`,
		fixture.userID,
	).Scan(&openRows).Error; err != nil {
		t.Fatalf("count open intervals: %v", err)
	}
	if openRows != 0 {
		t.Errorf("open-ended intervals after two finite changes = %d, want 0", openRows)
	}
}

// barrierPlanPeriodRepo pauses the recorder's advisory lock until both
// transactions have reached it. Everything else delegates to the real
// repository, so the recorder still exercises the production SQL.
type barrierPlanPeriodRepo struct {
	repositories.IUserPlanPeriodRepository
	ready   chan<- struct{}
	release <-chan struct{}
	once    sync.Once
}

func (r *barrierPlanPeriodRepo) LockUser(ctx context.Context, userID uuid.UUID) error {
	r.once.Do(func() { r.ready <- struct{}{} })
	<-r.release
	return r.IUserPlanPeriodRepository.LockUser(ctx, userID)
}

// planRecorderHasExclusionConstraint reports whether the migration installed
// the full non-overlap guard rather than the partial-unique fallback.
func planRecorderHasExclusionConstraint(t *testing.T, db *gorm.DB) bool {
	t.Helper()
	var count int64
	if err := db.Raw(`SELECT COUNT(*) FROM pg_constraint WHERE conname = 'ex_user_plan_periods_no_overlap'`).Scan(&count).Error; err != nil {
		t.Fatalf("read exclusion constraint: %v", err)
	}
	return count > 0
}
