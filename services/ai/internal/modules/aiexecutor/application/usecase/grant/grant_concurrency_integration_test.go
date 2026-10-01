package grant

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/infrastructure/persistence"
	"smart-wardrobe-be/shared/domain/constants/aipolicygrantstatus"
	"smart-wardrobe-be/shared/infrastructure/db"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// racersForGrantLock is wide enough that the check-then-write window is reliably
// open, and small enough that a lock serialisation stays quick.
const racersForGrantLock = 8

// TestIntegrationConcurrentSameEventCreatesOneGrant is the regression test for
// 024 T163 (FR-033).
//
// The outbox guarantees at-least-once delivery, so two deliveries of the SAME event
// are the normal case rather than an exotic one. Before this task the duplicate
// check sat outside the transaction on a plain SELECT, and `source_event_id` carried
// only a non-unique index, so both deliveries read "absent" and both inserted.
//
// FR-034 forbids a unique constraint at the data layer - an account may hold several
// `active` rows as long as they do not overlap in time - so the guarantee has to come
// from the application: a per-account transaction lock plus a re-check inside it.
//
// A fake cannot demonstrate this. It has no unique index and no concurrent
// transactions, so the window never opens - the same trap T148 hit, which is why this
// runs against a real Postgres with a barrier.
func TestIntegrationConcurrentSameEventCreatesOneGrant(t *testing.T) {
	dbConn := openGrantTestDB(t)
	cleanupGrantTables(t, dbConn)

	repo := persistence.NewAICostRepository(dbConn)
	log := logger.New("test", "", "info", false)
	uc := NewGrantUseCase(repo, db.NewGormUnitOfWork(dbConn), log, NewGrantMetrics(log)).(*GrantUseCase)

	userID := uuid.New()
	policy := seedGrantPolicy(t, dbConn, "free-default")
	planID := uuid.New()
	sourceEventID := uuid.New()

	payload := validPayload()
	payload.UserID = userID.String()
	payload.AICostPolicyID = policy.ID.String()
	payload.ToPlanID = planID.String()
	payload.SourceEventID = sourceEventID.String()
	payload.EventType = "ACTIVATED"
	payload.FromPlanCode = ""

	errs := runConcurrently(racersForGrantLock, func(idx int) error {
		return uc.HandlePlanChanged(context.Background(), payload)
	})
	for i, err := range errs {
		if err != nil {
			t.Fatalf("racer %d failed: %v", i, err)
		}
	}

	var count int64
	if err := dbConn.Model(&entities.AIPolicyGrant{}).
		Where("source_event_id = ?", sourceEventID).Count(&count).Error; err != nil {
		t.Fatalf("count grants by source event: %v", err)
	}
	if count != 1 {
		t.Errorf("FR-033: one sourceEventID must create exactly ONE grant, got = %d", count)
	}
}

// TestIntegrationOlderEventAfterScheduledGrantIsRefused is the deterministic test for
// 024 T164. It is the one that actually pins the defect, and it needs no concurrency
// at all.
//
// The sequence is ordinary customer behaviour, not a race:
//
//  1. The customer renews early, so the new period starts in the future. The grant is
//     written with status `future` — correct, and required, because the old period is
//     still the one they are using.
//  2. A delivery that was already in flight arrives afterwards. It is older than the
//     renewal.
//
// Before the fix, step 2 was accepted and the account ended up with two `active`
// grants. The ordering check read GetCurrentGrant, which filters `effective_from <=
// now`, so it could not see the `future` grant the renewal had just written. It found
// nothing to compare against, declared itself fresh, and wrote. Its CloseActiveGrants
// then used a cutoff taken from its own earlier timestamp — which by design does not
// touch a future grant — so it closed nothing either.
//
// Two guards, both necessary and neither sufficient alone: the account lock serialises
// the writers, and the ordering rule decides which of them wins. This test is what
// proves the second one looks at the right row.
func TestIntegrationOlderEventAfterScheduledGrantIsRefused(t *testing.T) {
	dbConn := openGrantTestDB(t)
	cleanupGrantTables(t, dbConn)

	repo := persistence.NewAICostRepository(dbConn)
	log := logger.New("test", "", "info", false)
	uc := NewGrantUseCase(repo, db.NewGormUnitOfWork(dbConn), log, NewGrantMetrics(log)).(*GrantUseCase)

	userID := uuid.New()
	policy := seedGrantPolicy(t, dbConn, "premium")
	planID := uuid.New()
	now := time.Now().UTC()

	// The renewal: its paid stretch starts in the future, so the row is `future`.
	//
	// For EXTENDED the window comes from EffectiveTo minus DurationDays, not from
	// EffectiveAt, so the future start has to be expressed that way. Getting this
	// wrong produced an `active` row and the premise check below caught it - which is
	// what that check is for.
	renewal := validPayload()
	renewal.UserID = userID.String()
	renewal.AICostPolicyID = policy.ID.String()
	renewal.ToPlanID = planID.String()
	renewal.SourceEventID = uuid.New().String()
	renewal.EventType = "EXTENDED"
	renewal.OccurredAt = now.Format(time.RFC3339Nano)
	renewal.DurationDays = 30
	renewal.EffectiveTo = now.AddDate(0, 0, 60).Format(time.RFC3339Nano)
	renewal.EffectiveAt = now.AddDate(0, 0, 30).Format(time.RFC3339Nano)
	renewal.FromPlanCode = "premium"

	if err := uc.HandlePlanChanged(context.Background(), renewal); err != nil {
		t.Fatalf("the renewal must be accepted: %v", err)
	}

	// Pin the premise. If the renewal did not produce a `future` row then this test
	// is not exercising the defect at all, and a green result would mean nothing.
	var scheduled entities.AIPolicyGrant
	if err := dbConn.Where("user_id = ? AND source_event_id = ?", userID, renewal.SourceEventID).
		First(&scheduled).Error; err != nil {
		t.Fatalf("read the renewal grant: %v", err)
	}
	if scheduled.Status != aipolicygrantstatus.Future {
		t.Fatalf("test premise broken: the renewal grant must be `future`, got = %q", scheduled.Status)
	}
	if !scheduled.EffectiveFrom.After(now) {
		t.Fatalf("test premise broken: effective_from must be in the future, got = %s", scheduled.EffectiveFrom)
	}

	// The straggler: older, delivered after. Its own window is entirely reasonable,
	// which is the point - only the ordering rule can refuse it.
	straggler := renewal
	straggler.SourceEventID = uuid.New().String()
	straggler.OccurredAt = now.Add(-time.Minute).Format(time.RFC3339Nano)

	err := uc.HandlePlanChanged(context.Background(), straggler)
	var rejection *RejectionError
	if !errors.As(err, &rejection) {
		t.Fatalf("an event older than the stored record must be refused, got: %v", err)
	}
	if rejection.Reason != ReasonStaleEvent {
		t.Errorf("the rejection reason must be %q, got = %q", ReasonStaleEvent, rejection.Reason)
	}

	var total int64
	if err := dbConn.Model(&entities.AIPolicyGrant{}).
		Where("user_id = ?", userID).Count(&total).Error; err != nil {
		t.Fatalf("count grant rows: %v", err)
	}
	if total != 1 {
		t.Errorf("the older event must not add a row, total = %d%s", total, describeGrantRows(t, dbConn, userID))
	}

	var active int64
	if err := dbConn.Model(&entities.AIPolicyGrant{}).
		Where("user_id = ? AND status = ?", userID, aipolicygrantstatus.Active).
		Count(&active).Error; err != nil {
		t.Fatalf("count active grants: %v", err)
	}
	// A future grant is not active, so the account correctly holds zero active grants
	// right now and will hold exactly one when the period starts.
	if active != 0 {
		t.Errorf("an account whose grant is `future` future must have no active grant, got = %d", active)
	}
}

// TestIntegrationConcurrentDifferentEventsLeaveOneActiveGrant is the regression test
// for 024 T164 (FR-034).
//
// Two DIFFERENT events for one account is the ordinary case of a customer buying
// twice in quick succession. Each is individually legitimate, so the duplicate check
// cannot help: what has to hold is that they do not overlap, so the account never
// holds two `active` grants at once.
//
// Note that this test is a weaker guarantee than
// TestIntegrationOlderEventAfterScheduledGrantIsRefused above. It runs goroutines at
// one barrier and asserts the final state, but it cannot force the interleaving - the
// window between the read and the write is opened by the database scheduler, not by
// the test. Removing the account lock left this test GREEN, which is why the
// deterministic test above exists. Both are kept: this one exercises the lock under
// real contention, the other pins the rule.
func TestIntegrationConcurrentDifferentEventsLeaveOneActiveGrant(t *testing.T) {
	dbConn := openGrantTestDB(t)
	cleanupGrantTables(t, dbConn)

	repo := persistence.NewAICostRepository(dbConn)
	log := logger.New("test", "", "info", false)
	uc := NewGrantUseCase(repo, db.NewGormUnitOfWork(dbConn), log, NewGrantMetrics(log)).(*GrantUseCase)

	userID := uuid.New()
	policy := seedGrantPolicy(t, dbConn, "free-default")
	planID := uuid.New()
	now := time.Now().UTC()

	// Distinct source events, and the second is later.
	//
	// Which one lands first is NOT decided by this test, and it must not be: FR-033's
	// timestamp rule means that if the newer event is stored first, the older one is
	// then correctly refused as stale. Both orderings are legitimate. What FR-034
	// forbids is the outcome where the account is left holding two `active` grants,
	// so the assertion is on the final state, not on how many writes succeeded.
	first := validPayload()
	first.UserID = userID.String()
	first.AICostPolicyID = policy.ID.String()
	first.ToPlanID = planID.String()
	first.SourceEventID = uuid.New().String()
	first.EventType = "ACTIVATED"
	first.OccurredAt = now.Format(time.RFC3339Nano)
	first.EffectiveAt = now.Format(time.RFC3339Nano)
	first.FromPlanCode = ""

	second := first
	second.SourceEventID = uuid.New().String()
	second.OccurredAt = now.Add(time.Minute).Format(time.RFC3339Nano)
	second.EffectiveAt = now.Add(time.Minute).Format(time.RFC3339Nano)

	// Interleave the two events across one barrier, which is the real arrival
	// pattern: two plan changes for one account, processed at the same time.
	total := racersForGrantLock * 2
	errs := runConcurrently(total, func(idx int) error {
		if idx%2 == 0 {
			return uc.HandlePlanChanged(context.Background(), first)
		}
		return uc.HandlePlanChanged(context.Background(), second)
	})
	for i, err := range errs {
		if err == nil {
			continue
		}
		// A stale refusal is the correct outcome for whichever event lost the
		// ordering race. Anything else is a real failure.
		var rejection *RejectionError
		if !errors.As(err, &rejection) || rejection.Reason != ReasonStaleEvent {
			t.Fatalf("racer %d failed: %v", i, err)
		}
	}

	var active int64
	if err := dbConn.Model(&entities.AIPolicyGrant{}).
		Where("user_id = ? AND status = ?", userID, aipolicygrantstatus.Active).
		Count(&active).Error; err != nil {
		t.Fatalf("count active grants: %v", err)
	}
	if active != 1 {
		// Show the rows. "expected 1 active, got 2" says the invariant broke but not
		// how, and the how is the whole question here: which of the two events won,
		// and what did the loser believe about the account's current entitlement.
		t.Errorf("FR-034: an account may hold several active grants, got %d%s",
			active, describeGrantRows(t, dbConn, userID))
	}
}

// describeGrantRows renders an account's grant rows for a failure message.
//
// The rows are the diagnosis. "expected 1, got 2" does not say which event won or
// what the other one believed, and that difference is exactly what distinguishes one
// cause of an overlapping-entitlement bug from another - it is how the `future`
// visibility defect in 024 T164 was found in the first place.
//
// Errors are reported rather than swallowed: a diagnostic that quietly prints "no rows"
// because its own query failed is worse than no diagnostic, because it points the next
// reader at the wrong thing.
func describeGrantRows(t *testing.T, dbConn *gorm.DB, userID uuid.UUID) string {
	t.Helper()
	var rows []entities.AIPolicyGrant
	if err := dbConn.Model(&entities.AIPolicyGrant{}).
		Where("user_id = ?", userID).
		Order("effective_from").
		Find(&rows).Error; err != nil {
		return "\n      (could not read the grant rows for diagnosis: " + err.Error() + ")"
	}
	if len(rows) == 0 {
		return "\n      (the account has no grant rows)"
	}
	var b strings.Builder
	for _, r := range rows {
		source := "<nil>"
		if r.SourceEventID != nil {
			source = r.SourceEventID.String()
		}
		b.WriteString("\n      status=" + string(r.Status) +
			" effective_from=" + r.EffectiveFrom.UTC().Format(time.RFC3339Nano) +
			" source_event_id=" + source)
	}
	return b.String()
}

// TestIntegrationGrantLockIsReleasedAtCommit is what makes the lock safe to rely on.
//
// An advisory lock that leaked past its transaction would deadlock the next event for
// that account forever, and it would do so silently - no error, just a grant that
// never arrives. So the second event has to be able to proceed after the first
// commits.
func TestIntegrationGrantLockIsReleasedAtCommit(t *testing.T) {
	dbConn := openGrantTestDB(t)
	cleanupGrantTables(t, dbConn)

	repo := persistence.NewAICostRepository(dbConn)
	userID := uuid.New()

	ctx := context.Background()
	if err := repo.LockGrantsForAccount(ctx, userID); err != nil {
		t.Fatalf("first lock: %v", err)
	}
	// A second acquire on a DIFFERENT connection is what a concurrent event would do.
	// It must not block, because the first transaction has already committed.
	done := make(chan error, 1)
	go func() {
		// A second connection is what a concurrent event would use. Calling
		// t.Fatalf from a goroutine is illegal, so failures travel on the channel.
		other, err := gorm.Open(postgres.Open(os.Getenv("TEST_DATABASE_URL")), &gorm.Config{})
		if err != nil {
			done <- err
			return
		}
		done <- persistence.NewAICostRepository(other).LockGrantsForAccount(ctx, userID)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("second lock: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the advisory lock must not survive the commit - the next event would hang forever")
	}
}

// barrierGate releases its racers at once. Without it the goroutines interleave
// loosely enough that the check-then-write window never opens, and the test would
// pass even with the fix reverted.
type barrierGate struct {
	mu       sync.Mutex
	arrived  int
	want     int
	released bool
	gate     chan struct{}
}

func newBarrierGate(want int) *barrierGate {
	return &barrierGate{want: want, gate: make(chan struct{})}
}

func (b *barrierGate) wait() {
	b.mu.Lock()
	b.arrived++
	if b.arrived >= b.want && !b.released {
		b.released = true
		close(b.gate)
	}
	b.mu.Unlock()
	<-b.gate
}

// runConcurrently starts `n` goroutines and releases them simultaneously.
func runConcurrently(n int, fn func(idx int) error) []error {
	gate := newBarrierGate(n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			gate.wait()
			errs[idx] = fn(idx)
		}(i)
	}
	wg.Wait()
	return errs
}

func openGrantTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping grant concurrency integration test")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	if err := conn.AutoMigrate(
		&entities.AICostPolicy{},
		&entities.AICostPolicyOperation{},
		&entities.AIPolicyGrant{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	// Close the pool when the test ends. Without this every test run leaks a pool,
	// and the symptom is not a leak report: `go test -count=25` exhausts the server's
	// max_connections and comes back as "FATAL: sorry, too many clients already
	// (SQLSTATE 53300)", which looks exactly like a product bug and is not one.
	// The product handled that failure correctly — it classified it transient and
	// requeued, which is what T152 promises — so the failure had to be traced to
	// this leak rather than to the code under test.
	sqlDB, err := conn.DB()
	if err != nil {
		t.Fatalf("resolve underlying pool: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close the test connection pool: %v", err)
		}
	})
	return conn
}

func cleanupGrantTables(t *testing.T, conn *gorm.DB) {
	t.Helper()
	for _, model := range []any{
		&entities.AIPolicyGrant{},
		&entities.AICostPolicyOperation{},
		&entities.AICostPolicy{},
	} {
		if err := conn.Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("cleanup table: %v", err)
		}
	}
}

// seedGrantPolicy creates a policy with all four metered operations, so the snapshot
// completeness check in FR-029b is satisfied and the test exercises the grant path
// rather than the degraded one.
func seedGrantPolicy(t *testing.T, conn *gorm.DB, code string) *entities.AICostPolicy {
	t.Helper()

	policy := &entities.AICostPolicy{
		Code:                code,
		Version:             1,
		Name:                code,
		EnforcementMode:     aienforcementmode.ObserveOnly,
		PeriodDays:          30,
		CompactThresholdBPS: 0,
		IsActive:            true,
	}
	policy.ID = uuid.New()
	for _, op := range []aioperation.AIOperation{aioperation.Chat, aioperation.Outfit, aioperation.Summary, aioperation.Rewriter} {
		policy.Operations = append(policy.Operations, &entities.AICostPolicyOperation{
			PolicyID:  policy.ID,
			Operation: op,
			IsEnabled: true,
		})
	}
	if err := conn.Create(policy).Error; err != nil {
		t.Fatalf("seed policy: %v", err)
	}
	return policy
}
