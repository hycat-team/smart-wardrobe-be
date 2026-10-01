package subscription

import (
	"context"
	"sync"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure/persistence"
	"smart-wardrobe-be/shared/infrastructure/db"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// concurrencyRacers is wide enough that the "everyone read absent, everyone
// inserted" window is reliably open when the goroutines start. Twelve matches
// the request burst that reproduced this on the development database.
const concurrencyRacers = 12

// TestIntegrationConcurrentQuotaProvisioningCreatesOneRow is the regression test
// for the HTTP 500 that feature 024 T148 fixed.
//
// `user_daily_quotas.user_id` is the primary key, and GetOrCreateUserDailyQuota
// used to read-then-insert. On a brand-new account several handlers ask for the
// quota at the same time - the profile response and the AI path both need it -
// so they all read "absent" and all insert. The losers failed with
// `duplicate key value violates unique constraint "user_daily_quotas_pkey"`.
//
// A unit test with a fake cannot catch this: the fake has no unique index and no
// concurrent transactions, so it cannot produce the race. The failure needs a real
// database, which is why this lives next to the other Postgres integration tests
// and skips when TEST_DATABASE_URL is not set.
func TestIntegrationConcurrentQuotaProvisioningCreatesOneRow(t *testing.T) {
	dbConn := openSubscriptionTestDB(t)
	cleanupSubscriptionTables(t, dbConn)

	quotaRepo := persistence.NewUserDailyQuotaRepository(dbConn)
	// The barrier is what makes this a regression test rather than a flake hunt:
	// without it the racers happen to interleave loosely, the reads are spread out,
	// and each one may land after the previous insert has committed - which is
	// exactly why the plain read-then-insert survived a test run here even though
	// it fails under a real request burst. Holding every read until all of them have
	// completed forces all 12 inserts into the window the fix is about.
	barrierQuota := newBarrierQuotaRepo(quotaRepo, concurrencyRacers)
	support := NewSubscriptionStateSupport(
		nil,
		nil,
		barrierQuota,
		nil,
		db.NewGormUnitOfWork(dbConn),
	)

	userID := uuid.New()
	results := make([]*entities.UserDailyQuota, concurrencyRacers)
	errs := make([]error, concurrencyRacers)

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < concurrencyRacers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			results[idx], errs[idx] = support.GetOrCreateUserDailyQuota(context.Background(), userID)
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("racer %d failed: %v", i, err)
		}
	}

	var count int64
	if err := dbConn.Model(&entities.UserDailyQuota{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count quota rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 quota row for the account, got %d", count)
	}

	// Every racer must observe the same row, not just avoid erroring. Returning a
	// per-caller in-memory struct would reset the customer's counters, which is
	// worse than the failure this replaced.
	for i, quota := range results {
		if quota == nil {
			t.Fatalf("racer %d received a nil quota", i)
		}
		if quota.UserID != userID {
			t.Fatalf("racer %d received a quota for user %s, want %s", i, quota.UserID, userID)
		}
		if quota.OutfitRecommendCount != 0 || quota.AiUsageCount != 0 {
			t.Fatalf("racer %d received non-zero counters: outfit=%d ai=%d",
				i, quota.OutfitRecommendCount, quota.AiUsageCount)
		}
	}

	// The stored row must be the one every racer saw, i.e. the read-back is real
	// and not an optimisation that skipped it. `user_id` is the only key here, so
	// `last_reset_date` stands in for row identity: a per-caller in-memory struct
	// would carry each goroutine's own timestamp.
	var stored entities.UserDailyQuota
	if err := dbConn.Where("user_id = ?", userID).First(&stored).Error; err != nil {
		t.Fatalf("reload quota: %v", err)
	}
	for i, quota := range results {
		if !quota.LastResetDate.Equal(stored.LastResetDate) {
			t.Fatalf("racer %d observed last_reset_date %s but the stored row has %s",
				i, quota.LastResetDate, stored.LastResetDate)
		}
	}
}

// TestIntegrationConcurrentPlanChangePublishesOneEvent covers the other half of
// T148: the `event_key` UNIQUE index.
//
// Default provisioning derives the key as `DEFAULT_PROVISION:{userID}:{version}`,
// so two concurrent first-provisions of one account compute the SAME key. The
// check-then-insert in PublishPlanChange has a window between the check and the
// insert, and the loser used to return a unique-violation - an HTTP 500 for an
// account whose subscription had been created correctly.
//
// The assertion that matters most is the publish count. Refusing the duplicate
// insert is only half the fix: if the loser published anyway, the receiver would
// see two events for one change and have to de-duplicate them itself.
func TestIntegrationConcurrentPlanChangePublishesOneEvent(t *testing.T) {
	dbConn := openSubscriptionTestDB(t)
	cleanupSubscriptionTables(t, dbConn)

	eventRepo := persistence.NewUserSubscriptionEventRepository(dbConn)
	// Same reasoning as the quota test: the reads have to overlap for the writes to
	// race, so the barrier holds every GetByKey until all racers have one.
	publisher := &countingPublisher{}
	changePublisher := NewPlanChangePublisher(newBarrierEventRepo(eventRepo, concurrencyRacers), publisher)

	userID := uuid.New()
	eventKey := "DEFAULT_PROVISION:" + userID.String() + ":1"
	now := time.Now()
	tierRank := 0
	plan := &entities.SubscriptionPlan{
		// 024 T155: a plan row read from the database always carries an id, and it
		// travels to the AI receiver as toPlanId. A fixture without one produces an
		// event the receiver refuses by design, which is not what this test is about.
		AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}},
		Slug:            "free",
		Name:            "Free",
		Price:           decimal.Zero,
		PlanKind:        plankind.DefaultFree,
		TierRank:        tierRank,
		AICostPolicyID:  uuid.New(),
	}
	sub := &entities.UserSubscription{
		UserID:          userID,
		CurrentPlanCode: plan.Slug,
		CurrentTierRank: tierRank,
		CurrentPlanKind: plan.PlanKind,
		StartedAt:       now,
	}

	returned := make([]*entities.UserSubscriptionEvent, concurrencyRacers)
	errs := make([]error, concurrencyRacers)

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < concurrencyRacers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			event := &entities.UserSubscriptionEvent{
				EventKey:    eventKey,
				UserID:      userID,
				EventType:   "DEFAULT_PROVISION",
				ToPlanCode:  &plan.Slug,
				ToTierRank:  &plan.TierRank,
				OccurredAt:  now,
				EffectiveAt: now,
			}
			returned[idx], errs[idx] = changePublisher.PublishPlanChange(context.Background(), event, sub, plan)
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("racer %d failed: %v", i, err)
		}
	}

	var count int64
	if err := dbConn.Model(&entities.UserSubscriptionEvent{}).Where("event_key = ?", eventKey).Count(&count).Error; err != nil {
		t.Fatalf("count event rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 history row for event_key %q, got %d", eventKey, count)
	}

	if got := publisher.count(); got != 1 {
		t.Fatalf("expected exactly 1 published event, got %d - the duplicate insert is being published too", got)
	}

	// Every racer must get a usable event back, not nil: the caller uses the
	// returned id when it builds its own response.
	var firstID uuid.UUID
	for i, event := range returned {
		if event == nil {
			t.Fatalf("racer %d received a nil event", i)
		}
		if i == 0 {
			firstID = event.ID
			continue
		}
		if event.ID != firstID {
			t.Fatalf("racer %d observed event id %s, want the stored %s", i, event.ID, firstID)
		}
	}
}

// TestIntegrationPlanChangeEventKeyIsNotOverwrittenOnRetry pins the difference
// between "refuse the duplicate" and "silently accept it": a later write carrying
// an existing key must leave the original row untouched, not replace it.
//
// Without the ON CONFLICT clause this would either error (the old behaviour) or,
// if the conflict were resolved with DO UPDATE, quietly rewrite a customer's plan
// history to whatever the retry happened to carry.
func TestIntegrationPlanChangeEventKeyIsNotOverwrittenOnRetry(t *testing.T) {
	dbConn := openSubscriptionTestDB(t)
	cleanupSubscriptionTables(t, dbConn)

	eventRepo := persistence.NewUserSubscriptionEventRepository(dbConn)
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()
	firstPlanCode := "free"
	secondPlanCode := "premium"

	original := &entities.UserSubscriptionEvent{
		EventKey:    "MANUAL_GRANT:" + userID.String() + ":1",
		UserID:      userID,
		EventType:   "MANUAL_GRANT",
		ToPlanCode:  &firstPlanCode,
		OccurredAt:  now,
		EffectiveAt: now,
	}
	inserted, err := eventRepo.CreateIfAbsent(ctx, original)
	if err != nil {
		t.Fatalf("insert original: %v", err)
	}
	if !inserted {
		t.Fatalf("first insert reported no insert for a key that does not exist")
	}

	retry := &entities.UserSubscriptionEvent{
		EventKey:    original.EventKey,
		UserID:      userID,
		EventType:   "MANUAL_GRANT",
		ToPlanCode:  &secondPlanCode,
		OccurredAt:  now,
		EffectiveAt: now,
	}
	inserted, err = eventRepo.CreateIfAbsent(ctx, retry)
	if err != nil {
		t.Fatalf("retry should be refused, not errored: %v", err)
	}
	if inserted {
		t.Fatalf("retry reported an insert for a key that already exists")
	}

	stored, err := eventRepo.GetByKey(ctx, original.EventKey)
	if err != nil {
		t.Fatalf("reload event: %v", err)
	}
	if stored == nil {
		t.Fatalf("event row disappeared")
	}
	if stored.ToPlanCode == nil || *stored.ToPlanCode != firstPlanCode {
		got := "<nil>"
		if stored.ToPlanCode != nil {
			got = *stored.ToPlanCode
		}
		t.Fatalf("retry overwrote the stored plan code: got %q, want %q", got, firstPlanCode)
	}
	if stored.ID != original.ID {
		t.Fatalf("retry replaced the row: got id %s, want %s", stored.ID, original.ID)
	}
}

// countingPublisher counts successful publishes. It stands in for the broker so
// the test can assert the event was published exactly once.
type countingPublisher struct {
	mu     sync.Mutex
	events []any
}

func (p *countingPublisher) Publish(ctx context.Context, topic string, payload any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, payload)
	return nil
}

func (p *countingPublisher) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.events)
}

// readBarrier releases its readers only once `want` of them have arrived. Readers
// that arrive afterwards - the read-back after a lost race, for instance - pass
// straight through, so the gate is closed exactly once.
type readBarrier struct {
	want     int
	gate     chan struct{}
	mu       sync.Mutex
	arrived  int
	released bool
}

func newReadBarrier(want int) *readBarrier {
	return &readBarrier{want: want, gate: make(chan struct{})}
}

// wait blocks the caller until every racer has reached this point.
func (b *readBarrier) wait() {
	b.mu.Lock()
	b.arrived++
	if b.arrived >= b.want && !b.released {
		b.released = true
		close(b.gate)
	}
	b.mu.Unlock()
	<-b.gate
}

// barrierQuotaRepo wraps the real repository and holds every read until all racers
// have one, forcing the read-then-write window wide open.
type barrierQuotaRepo struct {
	repositories.IUserDailyQuotaRepository
	barrier *readBarrier
}

func newBarrierQuotaRepo(inner repositories.IUserDailyQuotaRepository, want int) *barrierQuotaRepo {
	return &barrierQuotaRepo{IUserDailyQuotaRepository: inner, barrier: newReadBarrier(want)}
}

func (r *barrierQuotaRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	quota, err := r.IUserDailyQuotaRepository.GetByUserID(ctx, userID)
	r.barrier.wait()
	return quota, err
}

// barrierEventRepo does the same for the event_key lookup.
//
// It gates the FIRST `want` lookups rather than "the first lookup per key". The
// two are not equivalent: every racer uses the same key here, so a per-key flag
// would let the second racer straight through and leave the barrier permanently
// short of its quota.
//
// Gating the first `want` calls in total is still exact. A read-back only happens
// on the conflict path, which requires an insert, which requires having passed the
// barrier - so no read-back can be among the first `want` calls. They are exactly
// the racers' existence checks.
type barrierEventRepo struct {
	repositories.IUserSubscriptionEventRepository
	barrier *readBarrier
	mu      sync.Mutex
	seen    int
}

func newBarrierEventRepo(inner repositories.IUserSubscriptionEventRepository, want int) *barrierEventRepo {
	return &barrierEventRepo{
		IUserSubscriptionEventRepository: inner,
		barrier:                          newReadBarrier(want),
	}
}

func (r *barrierEventRepo) GetByKey(ctx context.Context, key string) (*entities.UserSubscriptionEvent, error) {
	event, err := r.IUserSubscriptionEventRepository.GetByKey(ctx, key)
	r.mu.Lock()
	gate := r.seen < r.barrier.want
	if gate {
		r.seen++
	}
	r.mu.Unlock()
	if gate {
		r.barrier.wait()
	}
	return event, err
}
