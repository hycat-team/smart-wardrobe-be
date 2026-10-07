package grant

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/aipolicygrantstatus"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// 027 US2 integration tests, against a real Postgres (TEST_DATABASE_URL; skipped
// when unset). They reuse the harness from grant_concurrency_integration_test.go
// (openGrantTestDB, cleanupGrantTables, seedGrantPolicy) and the fixtures from
// grant_window_resolution_integration_test.go (newGrantIntegrationUC,
// planChangedPayload, ptrTime, insertGrantRow, seedUnreadableActiveGrant,
// countOverlappingGrantWindows, describeGrantRows). US2 is the same code path as
// US1; what it adds is the REPAIR scenario - an account whose stored grant is
// broken and which is then brought back to a readable state by one valid event.
//
// These tests are integration tests because the properties they assert are
// stored-state properties: a snapshot the reader cannot decode, a window whose
// cutoff would coincide with its own start (SQLSTATE 23514), and several
// historical rows that must all sort before the new one. A fake cannot express
// any of them, and a fake has no CHECK constraint to violate.
// ---------------------------------------------------------------------------

// grantWindowCheckName is the name this test gives the window CHECK it installs.
// It is deliberately NOT the production name (user_ai_policy_grants_check): this
// test drops and recreates its own constraint around a single run, and reusing
// the production name would make the cleanup reach into a schema the test does
// not own.
const grantWindowCheckName = "ai_policy_grants_test_window_check"

// installGrantWindowCheck adds the window CHECK the production schema carries
// but AutoMigrate does not create.
//
// The AIPolicyGrant entity has no `check:` tag on EffectiveTo, so AutoMigrate
// creates the column without the constraint the shipped migration
// (20260622131631_add_ai_cost_policy.sql) adds. Without it a 23514 could never
// fire here, and AS1's "no constraint violation" assertion would be vacuous -
// it would pass on a broken row that production would reject.
//
// The name is left unqualified on purpose: the entity's TableName() is
// `ai_policy_grants` with no schema, so the test connection decides where the
// table lives. The constraint is dropped again in cleanup so the shared test
// database is left as it was found.
func installGrantWindowCheck(t *testing.T, conn *gorm.DB) {
	t.Helper()

	drop := "ALTER TABLE ai_policy_grants DROP CONSTRAINT IF EXISTS " + grantWindowCheckName
	add := "ALTER TABLE ai_policy_grants ADD CONSTRAINT " + grantWindowCheckName +
		" CHECK (effective_to IS NULL OR effective_to > effective_from)"

	if err := conn.Exec(drop).Error; err != nil {
		t.Fatalf("drop stale repair-test window CHECK: %v", err)
	}
	if err := conn.Exec(add).Error; err != nil {
		t.Fatalf("install repair-test window CHECK: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Exec(drop).Error; err != nil {
			t.Errorf("drop the repair-test window CHECK: %v", err)
		}
	})
}

// seedClosedGrantRow writes one historical, already-closed grant row. AS3 needs
// more than one prior row, and none of the existing fixtures produces a closed
// row with a known window.
func seedClosedGrantRow(t *testing.T, conn *gorm.DB, userID, policyID uuid.UUID, planCode string, from, to time.Time) entities.AIPolicyGrant {
	t.Helper()

	row := entities.AIPolicyGrant{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{ID: uuid.New(), CreatedAt: from},
			UpdatedAt:  from,
		},
		UserID:         userID,
		PolicyID:       policyID,
		PlanID:         uuid.New(),
		PlanCode:       planCode,
		TierRank:       1,
		PolicySnapshot: entities.JSONDocument(`{}`),
		EffectiveFrom:  from,
		EffectiveTo:    ptrTime(to),
		Status:         aipolicygrantstatus.Closed,
	}
	insertGrantRow(t, conn, &row)
	return row
}

// TestIntegrationRepairClosesOldAndCreatesReadable is AS1 and the point of US2:
// an account whose live grant carries an UNREADABLE snapshot (cause 1) AND whose
// window starts exactly at the billing period start (cause 2) is repaired by one
// valid change event.
//
// It asserts all three outcomes the specification names at once:
//   - the old row is closed, and closed with a non-empty window so the CHECK
//     (effective_to IS NULL OR effective_to > effective_from) holds - the 23514
//     that killed the first production run must not recur;
//   - the new row is created with a snapshot the real readers can decode;
//   - no two live windows overlap.
//
// The old snapshot is asserted to be unreadable first, so the test cannot pass
// on a fixture that never reproduced the production lock-up.
func TestIntegrationRepairClosesOldAndCreatesReadable(t *testing.T) {
	conn := openGrantTestDB(t)
	cleanupGrantTables(t, conn)
	installGrantWindowCheck(t, conn)
	uc := newGrantIntegrationUC(conn)

	now := time.Now().UTC().Truncate(time.Microsecond)
	start := now.Add(-24 * time.Hour) // the billing period start
	userID := uuid.New()
	policy := seedGrantPolicy(t, conn, "premium")

	// The production shape: the live grant was written before the snapshot
	// envelope existed, so ExtractOccurredAt cannot read it (cause 1), and its
	// effective_from is exactly the billing period start, so closing it at that
	// instant would give effective_to == effective_from (cause 2).
	prior := seedUnreadableActiveGrant(t, conn, userID, policy.ID, start, start)
	if _, ok := ExtractOccurredAt(&prior); ok {
		t.Fatal("test premise broken: the prior snapshot must be unreadable by ExtractOccurredAt")
	}
	if !prior.EffectiveFrom.Equal(start) {
		t.Fatalf("test premise broken: prior.effective_from %v must equal the billing start %v",
			prior.EffectiveFrom, start)
	}

	// The repair event. effectiveAt equals the prior start, which is what the old
	// code used as the close cutoff and what produced the 23514.
	repair := planChangedPayload(userID, policy.ID, uuid.New(),
		subscriptioneventtype.ACTIVATED,
		now,                            // occurredAt: newer than prior.created_at
		start,                          // effectiveAt == billing start -> the 23514 trigger
		ptrTime(now.AddDate(0, 0, 30)), // effectiveTo
		30)                             // durationDays

	if err := uc.HandlePlanChanged(context.Background(), repair); err != nil {
		t.Fatalf("AS1: the repair must apply without a window-constraint failure (23514): %v", err)
	}

	// The old row is closed at the shifted start, so its window is non-empty.
	wantStart := start.Add(time.Microsecond)

	var closed entities.AIPolicyGrant
	if err := conn.Where("id = ?", prior.ID).First(&closed).Error; err != nil {
		t.Fatalf("re-read the repaired grant: %v", err)
	}
	if closed.Status != aipolicygrantstatus.Closed {
		t.Errorf("AS1: the old grant must be closed, got status %q%s",
			closed.Status, describeGrantRows(t, conn, userID))
	}
	if closed.EffectiveTo == nil || !closed.EffectiveTo.Equal(wantStart) {
		t.Errorf("AS1: the old grant must end at the shifted start %v, got %v", wantStart, closed.EffectiveTo)
	}
	if closed.EffectiveTo != nil && !closed.EffectiveTo.After(closed.EffectiveFrom) {
		t.Errorf("AS1: effective_to %v must be strictly after effective_from %v - the CHECK production enforces",
			closed.EffectiveTo, closed.EffectiveFrom)
	}

	// The new row exists and its snapshot is readable by the real readers.
	var created entities.AIPolicyGrant
	sourceID := uuid.MustParse(repair.SourceEventID)
	if err := conn.Where("user_id = ? AND source_event_id = ?", userID, sourceID).First(&created).Error; err != nil {
		t.Fatalf("read the newly created grant: %v", err)
	}
	if created.Status != aipolicygrantstatus.Active {
		t.Errorf("AS1: the new grant must be active, got status %q", created.Status)
	}
	if !created.EffectiveFrom.Equal(wantStart) {
		t.Errorf("AS1: the new grant must start at the shifted start %v, got %v", wantStart, created.EffectiveFrom)
	}
	if _, ok := ExtractOccurredAt(&created); !ok {
		t.Errorf("AS1: the new snapshot must carry a readable envelope the ordering check can use")
	}
	entitlements, err := decodeEntitlements(&created)
	if err != nil {
		t.Fatalf("AS1: the new snapshot entitlements must decode: %v", err)
	}
	for _, op := range entitlementOperations {
		if _, ok := entitlements[string(op)]; !ok {
			t.Errorf("AS1: the new snapshot is missing entitlement %q", op)
		}
	}

	if n := countOverlappingGrantWindows(t, conn, userID); n != 0 {
		t.Errorf("AS1: the repair must not leave overlapping windows, got %d%s",
			n, describeGrantRows(t, conn, userID))
	}
}

// TestIntegrationRepairDoesNotShiftNormalWindow is AS2: a healthy account - a
// readable live snapshot and an event newer than everything stored - must keep
// the event's declared start to the microsecond. The 1µs shift exists only to
// avoid an overlap; applying it to a normal change would move every window on
// every plan change for no reason, which FR-007 forbids.
func TestIntegrationRepairDoesNotShiftNormalWindow(t *testing.T) {
	conn := openGrantTestDB(t)
	cleanupGrantTables(t, conn)
	installGrantWindowCheck(t, conn)
	uc := newGrantIntegrationUC(conn)

	now := time.Now().UTC().Truncate(time.Microsecond)
	start := now.Add(-10 * 24 * time.Hour)
	userID := uuid.New()
	policy := seedGrantPolicy(t, conn, "premium")

	// A normal, healthy live grant. Its snapshot is readable - the control
	// fixture from grant_testhelpers_test.go - and its window began well before
	// the event, so the ordering floor sits far in the past.
	readable := grantWithReadableSnapshot(t).PolicySnapshot
	prior := entities.AIPolicyGrant{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{ID: uuid.New(), CreatedAt: start},
			UpdatedAt:  start,
		},
		UserID:         userID,
		PolicyID:       policy.ID,
		PlanID:         uuid.New(),
		PlanCode:       "premium-monthly",
		TierRank:       1,
		PolicySnapshot: readable,
		EffectiveFrom:  start,
		EffectiveTo:    ptrTime(now.Add(20 * 24 * time.Hour)),
		Status:         aipolicygrantstatus.Active,
	}
	insertGrantRow(t, conn, &prior)

	repair := planChangedPayload(userID, policy.ID, uuid.New(),
		subscriptioneventtype.ACTIVATED,
		now, // occurredAt: newer than the prior write
		now, // effectiveAt: already past the ordering floor
		ptrTime(now.AddDate(0, 0, 30)),
		30)

	if err := uc.HandlePlanChanged(context.Background(), repair); err != nil {
		t.Fatalf("AS2: a normal repair must apply: %v", err)
	}

	var created entities.AIPolicyGrant
	sourceID := uuid.MustParse(repair.SourceEventID)
	if err := conn.Where("user_id = ? AND source_event_id = ?", userID, sourceID).First(&created).Error; err != nil {
		t.Fatalf("read the newly created grant: %v", err)
	}
	// FR-007 / AS2: the declared start is already past every stored row, so it is
	// used verbatim - no microsecond is added.
	if !created.EffectiveFrom.Equal(now) {
		t.Errorf("AS2 (FR-007): the normal window must NOT be shifted; expected %v, got %v",
			now, created.EffectiveFrom)
	}
	if !created.EffectiveFrom.Equal(EventGrantWindow(&PlanChangedEvent{
		EventType:    subscriptioneventtype.ACTIVATED,
		EffectiveAt:  now,
		EffectiveTo:  ptrTime(now.AddDate(0, 0, 30)),
		DurationDays: 30,
	}).EffectiveFrom) {
		t.Errorf("AS2: expected the event's own window start %v, got %v", now, created.EffectiveFrom)
	}

	var closed entities.AIPolicyGrant
	if err := conn.Where("id = ?", prior.ID).First(&closed).Error; err != nil {
		t.Fatalf("re-read the prior grant: %v", err)
	}
	if closed.Status != aipolicygrantstatus.Closed || closed.EffectiveTo == nil || !closed.EffectiveTo.Equal(now) {
		t.Errorf("AS2: the prior grant must be closed at the unshifted start %v, got status %q / effective_to %v",
			now, closed.Status, closed.EffectiveTo)
	}

	if n := countOverlappingGrantWindows(t, conn, userID); n != 0 {
		t.Errorf("AS2: no overlap expected, got %d%s", n, describeGrantRows(t, conn, userID))
	}
}

// TestIntegrationRepairStartsAfterAllPrior is AS3: with a layered history - two
// closed rows and the live row, whose snapshot is unreadable - the repaired
// grant must start strictly after every prior row, and the account must be left
// with no two live windows overlapping.
//
// This is the ordering half of the repair. A created_at-only floor would miss
// nothing here because the live row is in the past, but the assertion is on
// stored state rather than on a computed value: it re-reads every prior row and
// compares intervals, so a regression in the shift shows up as a real overlap.
func TestIntegrationRepairStartsAfterAllPrior(t *testing.T) {
	conn := openGrantTestDB(t)
	cleanupGrantTables(t, conn)
	installGrantWindowCheck(t, conn)
	uc := newGrantIntegrationUC(conn)

	now := time.Now().UTC().Truncate(time.Microsecond)
	userID := uuid.New()
	policy := seedGrantPolicy(t, conn, "premium")

	// A layered history: two closed rows, then the live row as the newest by
	// effective_from. Only the live row is in force; the closed ones must stay
	// untouched.
	seedClosedGrantRow(t, conn, userID, policy.ID, "premium-monthly",
		now.Add(-40*24*time.Hour), now.Add(-25*24*time.Hour))
	seedClosedGrantRow(t, conn, userID, policy.ID, "premium-monthly",
		now.Add(-25*24*time.Hour), now.Add(-15*24*time.Hour))
	live := seedUnreadableActiveGrant(t, conn, userID, policy.ID,
		now.Add(-15*24*time.Hour), now.Add(-15*24*time.Hour))

	// The repair event declares a start equal to the live row's start, so the
	// resolved start must move forward rather than collide with it.
	repair := planChangedPayload(userID, policy.ID, uuid.New(),
		subscriptioneventtype.ACTIVATED,
		now,
		live.EffectiveFrom,
		ptrTime(now.AddDate(0, 0, 15)),
		30)

	if err := uc.HandlePlanChanged(context.Background(), repair); err != nil {
		t.Fatalf("AS3: the repair must apply: %v", err)
	}

	var created entities.AIPolicyGrant
	sourceID := uuid.MustParse(repair.SourceEventID)
	if err := conn.Where("user_id = ? AND source_event_id = ?", userID, sourceID).First(&created).Error; err != nil {
		t.Fatalf("read the newly created grant: %v", err)
	}

	var priors []entities.AIPolicyGrant
	if err := conn.Where("user_id = ? AND (source_event_id IS NULL OR source_event_id <> ?)", userID, sourceID).
		Order("effective_from").Find(&priors).Error; err != nil {
		t.Fatalf("read the prior grants: %v", err)
	}
	if len(priors) != 3 {
		t.Fatalf("test premise broken: expected 3 prior rows, got %d%s",
			len(priors), describeGrantRows(t, conn, userID))
	}
	for _, p := range priors {
		if !created.EffectiveFrom.After(p.EffectiveFrom) {
			t.Errorf("AS3: the new window must start strictly after every prior row; %v is not after %v",
				created.EffectiveFrom, p.EffectiveFrom)
		}
	}

	// The live row is closed; the historical rows keep their status.
	var closedLive entities.AIPolicyGrant
	if err := conn.Where("id = ?", live.ID).First(&closedLive).Error; err != nil {
		t.Fatalf("re-read the live grant: %v", err)
	}
	if closedLive.Status != aipolicygrantstatus.Closed {
		t.Errorf("AS3: the live grant must be closed, got %q", closedLive.Status)
	}

	if n := countOverlappingGrantWindows(t, conn, userID); n != 0 {
		t.Errorf("AS3: no overlapping windows expected, got %d%s", n, describeGrantRows(t, conn, userID))
	}

	var activeCount int64
	if err := conn.Model(&entities.AIPolicyGrant{}).
		Where("user_id = ? AND status = ?", userID, aipolicygrantstatus.Active).
		Count(&activeCount).Error; err != nil {
		t.Fatalf("count active grants: %v", err)
	}
	if activeCount != 1 {
		t.Errorf("AS3: exactly one active grant expected, got %d%s",
			activeCount, describeGrantRows(t, conn, userID))
	}
}

// TestIntegrationRepairOfOpenEndedPlanIsNotRefused is the third cause of the blocked
// repair, and the one the production run hit.
//
// The repair grants the plan the account is ALREADY on, because that is the plan whose
// entitlement is missing. For an open-ended plan - `free` has duration_days NULL - the
// account service publishes EXTENDED with no effectiveTo and durationDays 0. The parser
// refused that combination unconditionally, so every repair event for the seven free
// accounts was dead-lettered as missing_required_field, the account row was updated, and
// no grant was ever written. The admin response said success the whole time.
//
// It goes through HandlePlanChanged, which parses the payload itself, so the assertion
// is about the real entry point the consumer uses rather than about a hand-built event
// that skips the rule.
func TestIntegrationRepairOfOpenEndedPlanIsNotRefused(t *testing.T) {
	conn := openGrantTestDB(t)
	cleanupGrantTables(t, conn)
	installGrantWindowCheck(t, conn)
	uc := newGrantIntegrationUC(conn)

	now := time.Now().UTC().Truncate(time.Microsecond)
	start := now.Add(-24 * time.Hour)
	userID := uuid.New()
	policy := seedGrantPolicy(t, conn, "premium")

	prior := seedUnreadableActiveGrant(t, conn, userID, policy.ID, start, start)

	// The production shape for a free account: same plan, open-ended, durationDays 0.
	repair := planChangedPayload(userID, policy.ID, uuid.New(),
		subscriptioneventtype.EXTENDED,
		now, // occurredAt: newer than prior.created_at, so not stale
		now, // effectiveAt
		nil, // effectiveTo: an open-ended plan has no end date
		0)   // durationDays: 0 for an open-ended plan

	if err := uc.HandlePlanChanged(context.Background(), repair); err != nil {
		t.Fatalf("an open-ended repair must not be refused as missing_required_field: %v", err)
	}

	var closed entities.AIPolicyGrant
	if err := conn.Where("id = ?", prior.ID).First(&closed).Error; err != nil {
		t.Fatalf("re-read the repaired grant: %v", err)
	}
	if closed.Status != aipolicygrantstatus.Closed {
		t.Errorf("the broken grant must be closed, got status %q%s",
			closed.Status, describeGrantRows(t, conn, userID))
	}

	sourceID := uuid.MustParse(repair.SourceEventID)
	var created entities.AIPolicyGrant
	if err := conn.Where("user_id = ? AND source_event_id = ?", userID, sourceID).First(&created).Error; err != nil {
		t.Fatalf("the open-ended repair must write a new grant: %v%s", err, describeGrantRows(t, conn, userID))
	}
	if created.Status != aipolicygrantstatus.Active {
		t.Errorf("the new grant must be active, got status %q", created.Status)
	}
	if created.EffectiveTo != nil {
		t.Errorf("an open-ended plan must produce an open-ended window, got effective_to %v", created.EffectiveTo)
	}
	if _, ok := ExtractOccurredAt(&created); !ok {
		t.Errorf("the new snapshot must carry a readable envelope the ordering check can use")
	}
	if n := countOverlappingGrantWindows(t, conn, userID); n != 0 {
		t.Errorf("no two live windows may overlap, got %d%s", n, describeGrantRows(t, conn, userID))
	}
}
