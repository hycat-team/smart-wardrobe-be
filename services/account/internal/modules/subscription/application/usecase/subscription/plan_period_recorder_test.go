package subscription

import (
	"context"
	"sort"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"

	"github.com/google/uuid"
)

// 031 T027/T031 — unit coverage for the recorder transition table
// (data-model §3). Fixed clocks and a fake repository make every branch
// observable without a database: which rows were read, closed, replaced or
// created, at which instant, and which event types must not touch history at
// all.
//
// T028/T030 prove the same rules against Postgres; this file is the fast,
// deterministic half of that pair.

// fakePlanPeriodRepo is an in-memory IUserPlanPeriodRepository. It hands out
// copies, like a database would, so the recorder cannot mutate a row without
// going through Update/CloseCurrent/UpdateValidTo.
type fakePlanPeriodRepo struct {
	periods []*entities.UserPlanPeriod

	lockCalls          int
	currentReads       int
	createCalls        int
	updateCalls        int
	closeCalls         int
	updateValidToCalls int

	// callOrder records the two calls whose relative order is a correctness
	// property: the advisory lock must come before the current-row read.
	callOrder []string

	lastCreated *entities.UserPlanPeriod
	lastUpdated *entities.UserPlanPeriod
	lastCloseID uuid.UUID
	lastCloseAt time.Time
	lastValidTo *time.Time
}

var _ repositories.IUserPlanPeriodRepository = (*fakePlanPeriodRepo)(nil)

func (f *fakePlanPeriodRepo) GetDistribution(context.Context, time.Time, time.Time) ([]repositories.SubscriptionDistributionDTO, error) {
	return nil, nil
}

func (f *fakePlanPeriodRepo) LockUser(_ context.Context, _ uuid.UUID) error {
	f.lockCalls++
	f.callOrder = append(f.callOrder, "lock")
	return nil
}

func (f *fakePlanPeriodRepo) GetCurrentByUserIDWithLock(_ context.Context, userID uuid.UUID) (*entities.UserPlanPeriod, error) {
	f.currentReads++
	f.callOrder = append(f.callOrder, "current")
	var current *entities.UserPlanPeriod
	for _, period := range f.periods {
		if period.UserID != userID {
			continue
		}
		if current == nil || period.ValidFrom.After(current.ValidFrom) {
			current = period
		}
	}
	return clonePlanPeriod(current), nil
}

func (f *fakePlanPeriodRepo) CloseCurrent(_ context.Context, periodID uuid.UUID, closeAt time.Time) error {
	f.closeCalls++
	f.lastCloseID = periodID
	f.lastCloseAt = closeAt
	if period := f.find(periodID); period != nil {
		closedAt := closeAt
		period.ValidTo = &closedAt
		period.UpdatedAt = time.Now().UTC()
	}
	return nil
}

func (f *fakePlanPeriodRepo) Create(_ context.Context, period *entities.UserPlanPeriod) error {
	f.createCalls++
	created := clonePlanPeriod(period)
	if created.ID == uuid.Nil {
		created.ID = uuid.New()
	}
	f.lastCreated = clonePlanPeriod(created)
	f.periods = append(f.periods, created)
	return nil
}

func (f *fakePlanPeriodRepo) Update(_ context.Context, period *entities.UserPlanPeriod) error {
	f.updateCalls++
	updated := clonePlanPeriod(period)
	f.lastUpdated = clonePlanPeriod(updated)
	for i, existing := range f.periods {
		if existing.ID == updated.ID {
			f.periods[i] = updated
			return nil
		}
	}
	f.periods = append(f.periods, updated)
	return nil
}

func (f *fakePlanPeriodRepo) UpdateValidTo(_ context.Context, periodID uuid.UUID, validTo *time.Time) error {
	f.updateValidToCalls++
	f.lastValidTo = clonePlanPeriodTime(validTo)
	if period := f.find(periodID); period != nil {
		period.ValidTo = clonePlanPeriodTime(validTo)
		period.UpdatedAt = time.Now().UTC()
	}
	return nil
}

func (f *fakePlanPeriodRepo) GetByUserID(_ context.Context, userID uuid.UUID) ([]*entities.UserPlanPeriod, error) {
	var periods []*entities.UserPlanPeriod
	for _, period := range f.periods {
		if period.UserID == userID {
			periods = append(periods, clonePlanPeriod(period))
		}
	}
	sort.Slice(periods, func(i, j int) bool {
		return periods[i].ValidFrom.Before(periods[j].ValidFrom)
	})
	return periods, nil
}

func (f *fakePlanPeriodRepo) GetHistoryCutover(context.Context) (*entities.PlanHistoryMeta, error) {
	return nil, nil
}

func (f *fakePlanPeriodRepo) find(periodID uuid.UUID) *entities.UserPlanPeriod {
	for _, period := range f.periods {
		if period.ID == periodID {
			return period
		}
	}
	return nil
}

func (f *fakePlanPeriodRepo) writeCalls() int {
	return f.createCalls + f.updateCalls + f.closeCalls + f.updateValidToCalls
}

func clonePlanPeriod(period *entities.UserPlanPeriod) *entities.UserPlanPeriod {
	if period == nil {
		return nil
	}
	cloned := *period
	cloned.ValidTo = clonePlanPeriodTime(period.ValidTo)
	if period.SourceEventKey != nil {
		key := *period.SourceEventKey
		cloned.SourceEventKey = &key
	}
	return &cloned
}

func clonePlanPeriodTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func recorderTestPlan(slug, name string, kind plankind.PlanKind, tierRank int) *entities.SubscriptionPlan {
	return &entities.SubscriptionPlan{
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		Slug:            slug,
		Name:            name,
		IsActive:        true,
		PlanKind:        kind,
		TierRank:        tierRank,
	}
}

func recorderTestSub(userID uuid.UUID, plan *entities.SubscriptionPlan, startedAt time.Time, expiresAt *time.Time) *entities.UserSubscription {
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

func recorderTestEvent(userID uuid.UUID, key string, eventType subscriptioneventtype.Type, effectiveAt time.Time) *entities.UserSubscriptionEvent {
	return &entities.UserSubscriptionEvent{
		EventKey:    key,
		UserID:      userID,
		EventType:   string(eventType),
		OccurredAt:  effectiveAt,
		EffectiveAt: effectiveAt,
	}
}

func recorderTestPeriod(userID uuid.UUID, plan *entities.SubscriptionPlan, validFrom time.Time, validTo *time.Time, key string) *entities.UserPlanPeriod {
	period := &entities.UserPlanPeriod{
		UserID:             userID,
		SubscriptionPlanID: plan.ID,
		PlanCode:           plan.Slug,
		PlanName:           plan.Name,
		TierRank:           plan.TierRank,
		PlanKind:           plan.PlanKind,
		ValidFrom:          validFrom,
		ValidTo:            clonePlanPeriodTime(validTo),
	}
	if key != "" {
		period.SourceEventKey = &key
	}
	return period
}

func recordPlanPeriod(t *testing.T, recorder PlanPeriodRecorder, event *entities.UserSubscriptionEvent, sub *entities.UserSubscription, plan *entities.SubscriptionPlan) {
	t.Helper()
	if err := recorder.Record(context.Background(), event, sub, plan); err != nil {
		t.Fatalf("Record(%s) failed: %v", event.EventType, err)
	}
}

// TestPlanPeriodRecorderTransitionTable covers the six event types that change
// the interval and the one that must not. Each case asserts the persisted
// rows, not only the branch taken.
func TestPlanPeriodRecorderTransitionTable(t *testing.T) {
	userID := uuid.New()
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	effective := time.Date(2026, 1, 10, 8, 0, 0, 0, time.UTC)
	expiry := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)

	t.Run("ACTIVATED creates the first interval", func(t *testing.T) {
		repo := &fakePlanPeriodRepo{}
		recorder := NewPlanPeriodRecorder(repo)
		premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
		sub := recorderTestSub(userID, premium, start, &expiry)
		event := recorderTestEvent(userID, "T027:activated", subscriptioneventtype.ACTIVATED, start)

		recordPlanPeriod(t, recorder, event, sub, premium)

		if repo.currentReads != 1 {
			t.Errorf("current reads = %d, want 1: the switch path must look for the user's current interval", repo.currentReads)
		}
		if repo.createCalls != 1 || repo.writeCalls() != 1 {
			t.Fatalf("writes = %d (create=%d), want exactly one create", repo.writeCalls(), repo.createCalls)
		}
		created := repo.lastCreated
		if created.PlanCode != premium.Slug || created.PlanName != premium.Name || created.TierRank != premium.TierRank || created.PlanKind != premium.PlanKind {
			t.Errorf("created snapshot = (%s, %s, %d, %d), want (%s, %s, %d, %d)",
				created.PlanCode, created.PlanName, created.TierRank, created.PlanKind,
				premium.Slug, premium.Name, premium.TierRank, premium.PlanKind)
		}
		if !created.ValidFrom.Equal(start) || created.ValidTo == nil || !created.ValidTo.Equal(expiry) {
			t.Errorf("created interval = [%s, %v), want [%s, %s)", created.ValidFrom, created.ValidTo, start, expiry)
		}
		if created.SourceEventKey == nil || *created.SourceEventKey != event.EventKey {
			t.Errorf("created source_event_key = %v, want %q", created.SourceEventKey, event.EventKey)
		}
	})

	t.Run("UPGRADED closes at the effective instant and opens the new plan", func(t *testing.T) {
		repo := &fakePlanPeriodRepo{}
		recorder := NewPlanPeriodRecorder(repo)
		premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
		pro := recorderTestPlan("pro", "Pro", plankind.Finite, 2)
		repo.periods = []*entities.UserPlanPeriod{
			recorderTestPeriod(userID, premium, start, &expiry, "T027:upgrade:old"),
		}
		sub := recorderTestSub(userID, pro, effective, pointerToTime(effective.Add(30*24*time.Hour)))
		event := recorderTestEvent(userID, "T027:upgrade", subscriptioneventtype.UPGRADED, effective)

		recordPlanPeriod(t, recorder, event, sub, pro)

		if repo.closeCalls != 1 || !repo.lastCloseAt.Equal(effective) {
			t.Fatalf("close calls = %d at %s, want one close at %s", repo.closeCalls, repo.lastCloseAt, effective)
		}
		if repo.createCalls != 1 {
			t.Fatalf("create calls = %d, want 1", repo.createCalls)
		}
		periods, _ := repo.GetByUserID(context.Background(), userID)
		if len(periods) != 2 {
			t.Fatalf("history has %d interval(s), want 2", len(periods))
		}
		if periods[0].ValidTo == nil || !periods[0].ValidTo.Equal(effective) {
			t.Errorf("closed interval valid_to = %v, want %s", periods[0].ValidTo, effective)
		}
		if !periods[1].ValidFrom.Equal(effective) || periods[1].PlanCode != pro.Slug {
			t.Errorf("new interval = (%s, from %s), want (%s, from %s)", periods[1].PlanCode, periods[1].ValidFrom, pro.Slug, effective)
		}
	})

	t.Run("OVERLAID closes a lifetime interval and opens a finite one", func(t *testing.T) {
		repo := &fakePlanPeriodRepo{}
		recorder := NewPlanPeriodRecorder(repo)
		lifetime := recorderTestPlan("lifetime", "Lifetime", plankind.Lifetime, 9)
		premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
		repo.periods = []*entities.UserPlanPeriod{
			recorderTestPeriod(userID, lifetime, start, nil, "T027:overlay:old"),
		}
		sub := recorderTestSub(userID, premium, effective, &expiry)
		event := recorderTestEvent(userID, "T027:overlay", subscriptioneventtype.OVERLAID, effective)

		recordPlanPeriod(t, recorder, event, sub, premium)

		periods, _ := repo.GetByUserID(context.Background(), userID)
		if len(periods) != 2 {
			t.Fatalf("history has %d interval(s), want 2", len(periods))
		}
		if periods[0].PlanCode != lifetime.Slug || periods[0].ValidTo == nil || !periods[0].ValidTo.Equal(effective) {
			t.Errorf("lifetime interval = (%s, to %v), want (%s, to %s)", periods[0].PlanCode, periods[0].ValidTo, lifetime.Slug, effective)
		}
		if periods[1].PlanCode != premium.Slug || !periods[1].ValidFrom.Equal(effective) || periods[1].ValidTo == nil || !periods[1].ValidTo.Equal(expiry) {
			t.Errorf("overlay interval = (%s, from %s, to %v), want (%s, from %s, to %s)",
				periods[1].PlanCode, periods[1].ValidFrom, periods[1].ValidTo, premium.Slug, effective, expiry)
		}
	})

	t.Run("EXTENDED moves valid_to and never inserts a row", func(t *testing.T) {
		repo := &fakePlanPeriodRepo{}
		recorder := NewPlanPeriodRecorder(repo)
		premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
		repo.periods = []*entities.UserPlanPeriod{
			recorderTestPeriod(userID, premium, start, &expiry, "T027:extend:old"),
		}
		extendedExpiry := expiry.Add(30 * 24 * time.Hour)
		sub := recorderTestSub(userID, premium, start, &extendedExpiry)
		event := recorderTestEvent(userID, "T027:extend", subscriptioneventtype.EXTENDED, expiry)

		recordPlanPeriod(t, recorder, event, sub, premium)

		if repo.updateValidToCalls != 1 || repo.createCalls != 0 || repo.closeCalls != 0 {
			t.Fatalf("extend calls = (updateValidTo=%d, create=%d, close=%d), want (1, 0, 0)",
				repo.updateValidToCalls, repo.createCalls, repo.closeCalls)
		}
		if repo.lastValidTo == nil || !repo.lastValidTo.Equal(extendedExpiry) {
			t.Fatalf("valid_to written = %v, want %s", repo.lastValidTo, extendedExpiry)
		}
		periods, _ := repo.GetByUserID(context.Background(), userID)
		if len(periods) != 1 {
			t.Fatalf("history has %d interval(s), want 1: a renewal must not split one holding", len(periods))
		}
		if periods[0].ValidTo == nil || !periods[0].ValidTo.Equal(extendedExpiry) {
			t.Errorf("interval valid_to = %v, want %s", periods[0].ValidTo, extendedExpiry)
		}
	})

	t.Run("SUBSCRIPTION_DOWNGRADED_TO_FREE closes at expiry and opens free at the same instant", func(t *testing.T) {
		repo := &fakePlanPeriodRepo{}
		recorder := NewPlanPeriodRecorder(repo)
		premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
		free := recorderTestPlan("free", "Free", plankind.DefaultFree, 0)
		repo.periods = []*entities.UserPlanPeriod{
			recorderTestPeriod(userID, premium, start, &expiry, "T027:downgrade:old"),
		}
		late := expiry.Add(10 * 24 * time.Hour)
		sub := recorderTestSub(userID, free, late, nil)
		event := recorderTestEvent(userID, "T027:downgrade", subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE, late)

		recordPlanPeriod(t, recorder, event, sub, free)

		if repo.closeCalls != 1 || !repo.lastCloseAt.Equal(expiry) {
			t.Fatalf("paid interval closed at %s, want its contractual expiry %s (min(valid_to, effective))", repo.lastCloseAt, expiry)
		}
		periods, _ := repo.GetByUserID(context.Background(), userID)
		if len(periods) != 2 {
			t.Fatalf("history has %d interval(s), want 2", len(periods))
		}
		if periods[1].PlanCode != free.Slug || !periods[1].ValidFrom.Equal(expiry) || periods[1].ValidTo != nil {
			t.Errorf("free interval = (%s, from %s, to %v), want (%s, from %s, open)",
				periods[1].PlanCode, periods[1].ValidFrom, periods[1].ValidTo, free.Slug, expiry)
		}
		if periods[0].ValidTo == nil || !periods[1].ValidFrom.Equal(*periods[0].ValidTo) {
			t.Error("free interval must start exactly where the paid interval ended; the downgrade must not leave a gap")
		}
	})

	t.Run("LIFETIME_FALLBACK_RESTORED closes the finite plan and restores lifetime", func(t *testing.T) {
		repo := &fakePlanPeriodRepo{}
		recorder := NewPlanPeriodRecorder(repo)
		lifetime := recorderTestPlan("lifetime", "Lifetime", plankind.Lifetime, 9)
		premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
		repo.periods = []*entities.UserPlanPeriod{
			recorderTestPeriod(userID, premium, start, &expiry, "T027:fallback:old"),
		}
		sub := recorderTestSub(userID, lifetime, expiry, nil)
		event := recorderTestEvent(userID, "T027:fallback", subscriptioneventtype.LIFETIME_FALLBACK_RESTORED, expiry)

		recordPlanPeriod(t, recorder, event, sub, lifetime)

		periods, _ := repo.GetByUserID(context.Background(), userID)
		if len(periods) != 2 {
			t.Fatalf("history has %d interval(s), want 2", len(periods))
		}
		if periods[1].PlanCode != lifetime.Slug || !periods[1].ValidFrom.Equal(expiry) || periods[1].ValidTo != nil {
			t.Errorf("restored lifetime interval = (%s, from %s, to %v), want (%s, from %s, open)",
				periods[1].PlanCode, periods[1].ValidFrom, periods[1].ValidTo, lifetime.Slug, expiry)
		}
	})

	t.Run("AUTO_RENEW_UPDATED is a no-op and does not even read history", func(t *testing.T) {
		repo := &fakePlanPeriodRepo{}
		recorder := NewPlanPeriodRecorder(repo)
		premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
		repo.periods = []*entities.UserPlanPeriod{
			recorderTestPeriod(userID, premium, start, &expiry, "T027:auto-renew:old"),
		}
		sub := recorderTestSub(userID, premium, start, &expiry)
		event := recorderTestEvent(userID, "T027:auto-renew", subscriptioneventtype.AUTO_RENEW_UPDATED, effective)

		recordPlanPeriod(t, recorder, event, sub, premium)

		if repo.lockCalls != 0 || repo.currentReads != 0 || repo.writeCalls() != 0 {
			t.Errorf("auto-renew touched history: locks=%d reads=%d writes=%d, want 0/0/0",
				repo.lockCalls, repo.currentReads, repo.writeCalls())
		}
	})
}

// TestPlanPeriodRecorderTakesAdvisoryLockBeforeReadingCurrent pins the ordering
// that makes the fallback (partial-unique) variant safe: every write branch
// must take the per-user advisory lock before it reads the current interval.
// Without that order the second of two concurrent changes still reads the old
// row and can open an overlapping closed interval, because that variant has no
// exclusion constraint to reject it.
func TestPlanPeriodRecorderTakesAdvisoryLockBeforeReadingCurrent(t *testing.T) {
	userID := uuid.New()
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	expiry := start.Add(30 * 24 * time.Hour)
	effective := start.Add(10 * 24 * time.Hour)

	premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
	pro := recorderTestPlan("pro", "Pro", plankind.Finite, 2)
	free := recorderTestPlan("free", "Free", plankind.DefaultFree, 0)

	cases := []struct {
		name      string
		eventType subscriptioneventtype.Type
		sub       func() *entities.UserSubscription
		plan      *entities.SubscriptionPlan
	}{
		{
			name:      "switch",
			eventType: subscriptioneventtype.UPGRADED,
			sub:       func() *entities.UserSubscription { return recorderTestSub(userID, pro, effective, &expiry) },
			plan:      pro,
		},
		{
			name:      "extend",
			eventType: subscriptioneventtype.EXTENDED,
			sub:       func() *entities.UserSubscription { return recorderTestSub(userID, premium, start, &expiry) },
			plan:      premium,
		},
		{
			name:      "fallback",
			eventType: subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE,
			sub:       func() *entities.UserSubscription { return recorderTestSub(userID, free, effective, nil) },
			plan:      free,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakePlanPeriodRepo{}
			repo.periods = []*entities.UserPlanPeriod{
				recorderTestPeriod(userID, premium, start, &expiry, "T031:lock-order:old"),
			}
			recorder := NewPlanPeriodRecorder(repo)
			event := recorderTestEvent(userID, "T031:lock-order:"+tc.name, tc.eventType, effective)

			recordPlanPeriod(t, recorder, event, tc.sub(), tc.plan)

			if repo.lockCalls != 1 {
				t.Fatalf("advisory lock calls = %d, want 1 for a write branch", repo.lockCalls)
			}
			if len(repo.callOrder) < 2 || repo.callOrder[0] != "lock" || repo.callOrder[1] != "current" {
				t.Fatalf("call order = %v, want lock then current: the lock must serialize the read", repo.callOrder)
			}
		})
	}
}

// TestPlanPeriodRecorderMinOfValidToAndEffective pins the close boundary rule:
// closeAt = min(current.valid_to, effective). The two directions are distinct
// outcomes - a contiguous switch and an honest lapse.
func TestPlanPeriodRecorderMinOfValidToAndEffective(t *testing.T) {
	userID := uuid.New()
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

	t.Run("effective before expiry closes at the effective instant", func(t *testing.T) {
		repo := &fakePlanPeriodRepo{}
		recorder := NewPlanPeriodRecorder(repo)
		premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
		pro := recorderTestPlan("pro", "Pro", plankind.Finite, 2)
		expiry := start.Add(30 * 24 * time.Hour)
		effective := start.Add(10 * 24 * time.Hour)
		repo.periods = []*entities.UserPlanPeriod{
			recorderTestPeriod(userID, premium, start, &expiry, "T027:min:effective"),
		}
		sub := recorderTestSub(userID, pro, effective, pointerToTime(effective.Add(30*24*time.Hour)))
		event := recorderTestEvent(userID, "T027:min:effective", subscriptioneventtype.UPGRADED, effective)

		recordPlanPeriod(t, recorder, event, sub, pro)

		if !repo.lastCloseAt.Equal(effective) {
			t.Errorf("closed at %s, want the effective instant %s (min of valid_to %s and effective)", repo.lastCloseAt, effective, expiry)
		}
		periods, _ := repo.GetByUserID(context.Background(), userID)
		if len(periods) != 2 || periods[0].ValidTo == nil || !periods[0].ValidTo.Equal(effective) {
			t.Fatalf("history = %v, want the old interval closed at %s", periods, effective)
		}
	})

	t.Run("effective after expiry closes at the expiry, not at the event time", func(t *testing.T) {
		repo := &fakePlanPeriodRepo{}
		recorder := NewPlanPeriodRecorder(repo)
		premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
		pro := recorderTestPlan("pro", "Pro", plankind.Finite, 2)
		expiry := start.Add(30 * 24 * time.Hour)
		effective := expiry.Add(10 * 24 * time.Hour)
		repo.periods = []*entities.UserPlanPeriod{
			recorderTestPeriod(userID, premium, start, &expiry, "T027:min:expiry"),
		}
		sub := recorderTestSub(userID, pro, effective, pointerToTime(effective.Add(30*24*time.Hour)))
		event := recorderTestEvent(userID, "T027:min:expiry", subscriptioneventtype.UPGRADED, effective)

		recordPlanPeriod(t, recorder, event, sub, pro)

		if !repo.lastCloseAt.Equal(expiry) {
			t.Errorf("closed at %s, want the contractual expiry %s, not the late event time %s", repo.lastCloseAt, expiry, effective)
		}
		periods, _ := repo.GetByUserID(context.Background(), userID)
		if len(periods) != 2 {
			t.Fatalf("history has %d interval(s), want 2", len(periods))
		}
		if !periods[1].ValidFrom.Equal(effective) {
			t.Errorf("new interval starts at %s, want the event time %s: a later purchase leaves a gap after the old expiry %s",
				periods[1].ValidFrom, effective, expiry)
		}
	})
}

// TestPlanPeriodRecorderReplacesZeroLengthWindows covers the two ways a change
// can land at (or before) the current interval's start: provisioning plus a
// first grant in one transaction, and clock skew. Both must replace the row
// instead of writing a zero-length interval the CHECK constraint rejects.
func TestPlanPeriodRecorderReplacesZeroLengthWindows(t *testing.T) {
	userID := uuid.New()
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

	t.Run("provisioning and grant at the same instant replace the row", func(t *testing.T) {
		repo := &fakePlanPeriodRepo{}
		recorder := NewPlanPeriodRecorder(repo)
		free := recorderTestPlan("free", "Free", plankind.DefaultFree, 0)
		premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
		expiry := start.Add(30 * 24 * time.Hour)
		repo.periods = []*entities.UserPlanPeriod{
			recorderTestPeriod(userID, free, start, nil, "T031:provision:free"),
		}
		sub := recorderTestSub(userID, premium, start, &expiry)
		event := recorderTestEvent(userID, "T031:grant:premium", subscriptioneventtype.ACTIVATED, start)

		recordPlanPeriod(t, recorder, event, sub, premium)

		if repo.updateCalls != 1 || repo.createCalls != 0 || repo.closeCalls != 0 {
			t.Fatalf("calls = (update=%d, create=%d, close=%d), want (1, 0, 0): replacing avoids a zero-length interval",
				repo.updateCalls, repo.createCalls, repo.closeCalls)
		}
		periods, _ := repo.GetByUserID(context.Background(), userID)
		if len(periods) != 1 {
			t.Fatalf("history has %d interval(s), want exactly 1 after replacement", len(periods))
		}
		period := periods[0]
		if period.PlanCode != premium.Slug || !period.ValidFrom.Equal(start) || period.ValidTo == nil || !period.ValidTo.Equal(expiry) {
			t.Errorf("replaced interval = (%s, %s, %v), want (%s, %s, %s)",
				period.PlanCode, period.ValidFrom, period.ValidTo, premium.Slug, start, expiry)
		}
		if period.SourceEventKey == nil || *period.SourceEventKey != event.EventKey {
			t.Errorf("replaced source_event_key = %v, want %q", period.SourceEventKey, event.EventKey)
		}
	})

	t.Run("clock skew clamps valid_from to the current interval start", func(t *testing.T) {
		repo := &fakePlanPeriodRepo{}
		recorder := NewPlanPeriodRecorder(repo)
		premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
		pro := recorderTestPlan("pro", "Pro", plankind.Finite, 2)
		currentStart := start.Add(5 * 24 * time.Hour)
		currentEnd := currentStart.Add(30 * 24 * time.Hour)
		skewedEffective := start // earlier than the current interval's start
		proExpiry := skewedEffective.Add(60 * 24 * time.Hour)
		repo.periods = []*entities.UserPlanPeriod{
			recorderTestPeriod(userID, premium, currentStart, &currentEnd, "T031:skew:old"),
		}
		sub := recorderTestSub(userID, pro, skewedEffective, &proExpiry)
		event := recorderTestEvent(userID, "T031:skew:upgrade", subscriptioneventtype.UPGRADED, skewedEffective)

		recordPlanPeriod(t, recorder, event, sub, pro)

		periods, _ := repo.GetByUserID(context.Background(), userID)
		if len(periods) != 1 {
			t.Fatalf("history has %d interval(s), want 1 after replacement", len(periods))
		}
		period := periods[0]
		if !period.ValidFrom.Equal(currentStart) {
			t.Errorf("valid_from = %s, want the clamp %s (max(effective, current.valid_from))", period.ValidFrom, currentStart)
		}
		if !period.ValidFrom.Before(*period.ValidTo) {
			t.Errorf("interval [%s, %s) is not positive after clamping", period.ValidFrom, *period.ValidTo)
		}
		if period.PlanCode != pro.Slug {
			t.Errorf("plan_code = %q, want the replacement plan %q", period.PlanCode, pro.Slug)
		}
	})

	t.Run("downgrade at the interval start replaces the paid row", func(t *testing.T) {
		repo := &fakePlanPeriodRepo{}
		recorder := NewPlanPeriodRecorder(repo)
		premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
		free := recorderTestPlan("free", "Free", plankind.DefaultFree, 0)
		expiry := start.Add(30 * 24 * time.Hour)
		repo.periods = []*entities.UserPlanPeriod{
			recorderTestPeriod(userID, premium, start, &expiry, "T031:fallback-same-instant:old"),
		}
		// The downgrade lands exactly at the current interval's start (clock
		// skew): close+insert would write a zero-length paid interval and fail
		// the CHECK constraint.
		sub := recorderTestSub(userID, free, start, nil)
		event := recorderTestEvent(userID, "T031:fallback-same-instant", subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE, start)

		recordPlanPeriod(t, recorder, event, sub, free)

		if repo.updateCalls != 1 || repo.createCalls != 0 || repo.closeCalls != 0 {
			t.Fatalf("calls = (update=%d, create=%d, close=%d), want (1, 0, 0): replacing avoids the invalid range",
				repo.updateCalls, repo.createCalls, repo.closeCalls)
		}
		periods, _ := repo.GetByUserID(context.Background(), userID)
		if len(periods) != 1 {
			t.Fatalf("history has %d interval(s), want 1 after replacement", len(periods))
		}
		period := periods[0]
		if period.PlanCode != free.Slug || !period.ValidFrom.Equal(start) || period.ValidTo != nil {
			t.Errorf("replaced interval = (%s, %s, %v), want (%s, %s, open)",
				period.PlanCode, period.ValidFrom, period.ValidTo, free.Slug, start)
		}
		if period.SourceEventKey == nil || *period.SourceEventKey != event.EventKey {
			t.Errorf("replaced source_event_key = %v, want %q", period.SourceEventKey, event.EventKey)
		}
	})
}

// TestPlanPeriodRecorderAdminRestoreNeverWrites pins the absolute no-op that
// protects a concurrent purchase. The restore path emits ACTIVATED with
// source=admin_restore; it repairs AI grants, not the subscription window.
func TestPlanPeriodRecorderAdminRestoreNeverWrites(t *testing.T) {
	userID := uuid.New()
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	expiry := start.Add(30 * 24 * time.Hour)

	t.Run("same plan activation is a no-op", func(t *testing.T) {
		repo := &fakePlanPeriodRepo{}
		recorder := NewPlanPeriodRecorder(repo)
		premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
		repo.periods = []*entities.UserPlanPeriod{
			recorderTestPeriod(userID, premium, start, &expiry, "T031:restore:same-plan"),
		}
		sub := recorderTestSub(userID, premium, start, &expiry)
		event := recorderTestEvent(userID, "T031:restore:same-plan", subscriptioneventtype.ACTIVATED, sub.StartedAt)

		recordPlanPeriod(t, recorder, event, sub, premium)

		if repo.writeCalls() != 0 {
			t.Errorf("same-plan ACTIVATED wrote %d change(s), want 0: the plan and window did not change", repo.writeCalls())
		}
	})

	t.Run("source=admin_restore overrides a concurrent purchase without reading it", func(t *testing.T) {
		repo := &fakePlanPeriodRepo{}
		recorder := NewPlanPeriodRecorder(repo)
		pro := recorderTestPlan("pro", "Pro", plankind.Finite, 2)
		lifetime := recorderTestPlan("lifetime", "Lifetime", plankind.Lifetime, 9)
		purchasedAt := start.Add(20 * 24 * time.Hour)
		purchaseExpiry := purchasedAt.Add(30 * 24 * time.Hour)
		// A purchase already committed a newer interval. The restore event is
		// older and names a different plan; without the metadata guard it would
		// close or replace this row.
		repo.periods = []*entities.UserPlanPeriod{
			recorderTestPeriod(userID, pro, purchasedAt, &purchaseExpiry, "T031:restore:purchase"),
		}
		before, _ := repo.GetByUserID(context.Background(), userID)
		sub := recorderTestSub(userID, lifetime, start, nil)
		event := recorderTestEvent(userID, "T031:restore:admin", subscriptioneventtype.ACTIVATED, start)
		event.Metadata = entities.JSONDocument(`{"source":"admin_restore"}`)

		recordPlanPeriod(t, recorder, event, sub, lifetime)

		if repo.lockCalls != 0 || repo.currentReads != 0 || repo.writeCalls() != 0 {
			t.Errorf("admin restore touched history: locks=%d reads=%d writes=%d, want 0/0/0",
				repo.lockCalls, repo.currentReads, repo.writeCalls())
		}
		after, _ := repo.GetByUserID(context.Background(), userID)
		if len(after) != len(before) || after[0].ID != before[0].ID || after[0].PlanCode != before[0].PlanCode {
			t.Errorf("admin restore changed the interval: before=%v after=%v", before[0], after[0])
		}
		if after[0].ValidTo == nil || !after[0].ValidTo.Equal(purchaseExpiry) {
			t.Errorf("admin restore moved the purchase's valid_to to %v, want %s", after[0].ValidTo, purchaseExpiry)
		}
	})
}

// TestPlanPeriodRecorderLegacyUserWithoutInterval covers a user who predates
// the cutover and therefore has no current row. The recorder must create one
// instead of inventing a close, for both a switch and a renewal.
func TestPlanPeriodRecorderLegacyUserWithoutInterval(t *testing.T) {
	userID := uuid.New()
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	effective := start.Add(10 * 24 * time.Hour)
	expiry := start.Add(30 * 24 * time.Hour)

	t.Run("switch inserts the new interval from the event time", func(t *testing.T) {
		repo := &fakePlanPeriodRepo{}
		recorder := NewPlanPeriodRecorder(repo)
		premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
		sub := recorderTestSub(userID, premium, effective, &expiry)
		event := recorderTestEvent(userID, "T031:legacy:switch", subscriptioneventtype.UPGRADED, effective)

		recordPlanPeriod(t, recorder, event, sub, premium)

		if repo.createCalls != 1 || repo.closeCalls != 0 {
			t.Fatalf("calls = (create=%d, close=%d), want (1, 0): there is no row to close", repo.createCalls, repo.closeCalls)
		}
	})

	t.Run("extended inserts the current window from sub.started_at", func(t *testing.T) {
		repo := &fakePlanPeriodRepo{}
		recorder := NewPlanPeriodRecorder(repo)
		premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
		sub := recorderTestSub(userID, premium, start, &expiry)
		event := recorderTestEvent(userID, "T031:legacy:extend", subscriptioneventtype.EXTENDED, effective)

		recordPlanPeriod(t, recorder, event, sub, premium)

		if repo.createCalls != 1 || repo.updateValidToCalls != 0 {
			t.Fatalf("calls = (create=%d, updateValidTo=%d), want (1, 0)", repo.createCalls, repo.updateValidToCalls)
		}
		if repo.lastCreated.ValidFrom.Equal(effective) {
			t.Errorf("inserted interval starts at the event time %s; the legacy fallback must use sub.StartedAt %s", effective, start)
		}
		if !repo.lastCreated.ValidFrom.Equal(start) {
			t.Errorf("inserted interval starts at %s, want sub.StartedAt %s", repo.lastCreated.ValidFrom, start)
		}
	})
}

// TestPlanPeriodRecorderLapseLeavesGap proves the recorder does not invent
// continuity across a lapse: when the paid interval already ended before the
// new purchase, the gap is preserved (FR-002 requires contiguity only for a
// continuous switch).
func TestPlanPeriodRecorderLapseLeavesGap(t *testing.T) {
	userID := uuid.New()
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	expiry := start.Add(30 * 24 * time.Hour)
	repurchaseAt := expiry.Add(10 * 24 * time.Hour)
	repurchaseExpiry := repurchaseAt.Add(30 * 24 * time.Hour)

	repo := &fakePlanPeriodRepo{}
	recorder := NewPlanPeriodRecorder(repo)
	premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
	pro := recorderTestPlan("pro", "Pro", plankind.Finite, 2)
	repo.periods = []*entities.UserPlanPeriod{
		recorderTestPeriod(userID, premium, start, &expiry, "T031:lapse:old"),
	}
	sub := recorderTestSub(userID, pro, repurchaseAt, &repurchaseExpiry)
	event := recorderTestEvent(userID, "T031:lapse:new", subscriptioneventtype.UPGRADED, repurchaseAt)

	recordPlanPeriod(t, recorder, event, sub, pro)

	periods, _ := repo.GetByUserID(context.Background(), userID)
	if len(periods) != 2 {
		t.Fatalf("history has %d interval(s), want 2", len(periods))
	}
	if periods[0].ValidTo == nil || !periods[0].ValidTo.Equal(expiry) {
		t.Fatalf("old interval valid_to = %v, want the expiry %s", periods[0].ValidTo, expiry)
	}
	if !periods[1].ValidFrom.Equal(repurchaseAt) {
		t.Fatalf("new interval starts at %s, want the purchase time %s", periods[1].ValidFrom, repurchaseAt)
	}
	if !periods[1].ValidFrom.After(*periods[0].ValidTo) {
		t.Error("the lapse was silently closed: the new purchase must start after the old expiry, leaving an honest gap")
	}
}

func pointerToTime(value time.Time) *time.Time {
	return &value
}
