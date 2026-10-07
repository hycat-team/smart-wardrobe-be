package grant

import (
	"context"
	"errors"
	"testing"
	"time"

	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/infrastructure/persistence"
	"smart-wardrobe-be/shared/domain/constants/aipolicygrantstatus"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"
	"smart-wardrobe-be/shared/infrastructure/db"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// 027 US1 integration tests, against a real Postgres (TEST_DATABASE_URL; skipped
// when unset). They follow the harness in grant_concurrency_integration_test.go:
// openGrantTestDB owns the connection and AutoMigrate, cleanupGrantTables empties
// the three tables, and seedGrantPolicy builds a policy with all four metered
// operations so the grant path is the one exercised - not the degraded one.
//
// These tests exist because the property that matters here is a stored-state
// property: an unreadable snapshot, a `future` row whose effective_from is ahead
// of its created_at, and two concurrent writers. A fake cannot express any of
// them.
// ---------------------------------------------------------------------------

func newGrantIntegrationUC(conn *gorm.DB) *GrantUseCase {
	repo := persistence.NewAICostRepository(conn)
	log := logger.New("test", "", "info", false)
	return NewGrantUseCase(repo, db.NewGormUnitOfWork(conn), log, NewGrantMetrics(log)).(*GrantUseCase)
}

func ptrTime(t time.Time) *time.Time { return &t }

// insertGrantRow writes a grant row directly so the test controls created_at and
// effective_from exactly. It is the only way to reproduce a `future` grant whose
// created_at is in the past, which is the shape the two-floor rule exists for.
func insertGrantRow(t *testing.T, conn *gorm.DB, row *entities.AIPolicyGrant) {
	t.Helper()
	if err := conn.Create(row).Error; err != nil {
		t.Fatalf("insert grant row: %v", err)
	}
}

// seedFutureGrant inserts a `future` grant starting 30 days after it was written.
// created_at is in the past and effective_from is in the future - the exact
// combination that makes a created_at-only floor unable to see it (FR-004a, H2).
func seedFutureGrant(t *testing.T, conn *gorm.DB, userID, policyID uuid.UUID, now time.Time) entities.AIPolicyGrant {
	t.Helper()
	row := entities.AIPolicyGrant{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{ID: uuid.New(), CreatedAt: now},
			UpdatedAt:  now,
		},
		UserID:         userID,
		PolicyID:       policyID,
		PlanID:         uuid.New(),
		PlanCode:       "premium-monthly",
		TierRank:       1,
		PolicySnapshot: entities.JSONDocument(`{}`),
		EffectiveFrom:  now.AddDate(0, 0, 30),
		EffectiveTo:    ptrTime(now.AddDate(0, 0, 90)),
		Status:         aipolicygrantstatus.Future,
	}
	insertGrantRow(t, conn, &row)
	return row
}

// seedUnreadableActiveGrant inserts the production failure shape: an active grant
// whose policy snapshot has no envelope, so ExtractOccurredAt cannot read it. 027
// must order against this row anyway, using created_at.
func seedUnreadableActiveGrant(t *testing.T, conn *gorm.DB, userID, policyID uuid.UUID, effectiveFrom, createdAt time.Time) entities.AIPolicyGrant {
	t.Helper()
	row := entities.AIPolicyGrant{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{ID: uuid.New(), CreatedAt: createdAt},
			UpdatedAt:  createdAt,
		},
		UserID:         userID,
		PolicyID:       policyID,
		PlanID:         uuid.New(),
		PlanCode:       "premium-monthly",
		TierRank:       1,
		PolicySnapshot: entities.JSONDocument(`{}`),
		EffectiveFrom:  effectiveFrom,
		Status:         aipolicygrantstatus.Active,
	}
	insertGrantRow(t, conn, &row)
	return row
}

// planChangedPayload builds a wire payload for a chosen event type and clock,
// starting from validPayload() so every required field stays valid.
//
// A nil effectiveTo means OPEN-ENDED, and it must clear the field rather than leave
// validPayload()'s value in place. The previous form only wrote the field when it was
// non-nil, so every "open-ended" payload this helper built still carried an end date -
// which is why no test ever exercised EXTENDED with no effectiveTo, and why the
// production repair that depends on exactly that shape was not caught here.
func planChangedPayload(
	userID, policyID, planID uuid.UUID,
	eventType subscriptioneventtype.Type,
	occurredAt, effectiveAt time.Time,
	effectiveTo *time.Time,
	durationDays int,
) aiexecutor_dto.PlanChangedEventPayload {
	payload := validPayload()
	payload.UserID = userID.String()
	payload.AICostPolicyID = policyID.String()
	payload.ToPlanID = planID.String()
	payload.SourceEventID = uuid.New().String()
	payload.EventType = string(eventType)
	payload.OccurredAt = occurredAt.Format(time.RFC3339Nano)
	payload.EffectiveAt = effectiveAt.Format(time.RFC3339Nano)
	payload.EffectiveTo = ""
	if effectiveTo != nil {
		payload.EffectiveTo = effectiveTo.Format(time.RFC3339Nano)
	}
	payload.DurationDays = durationDays
	payload.FromPlanCode = "free"
	payload.ToPlanCode = "premium-monthly"
	return payload
}

// countOverlappingGrantWindows counts pairs of live (active or future) windows for
// one account that overlap in time. It is the direct expression of FR-008, and it
// is timing-independent: it looks at the stored intervals rather than at now().
//
// The table name is UNQUALIFIED on purpose: the grant entity's TableName() is
// "ai_policy_grants" with no schema, so AutoMigrate creates it in whatever schema
// the test connection resolves to. Prefixing it with a schema that the entity does
// not use fails with SQLSTATE 42P01, exactly as the other integration tests would
// if they hardcoded one.
func countOverlappingGrantWindows(t *testing.T, conn *gorm.DB, userID uuid.UUID) int64 {
	t.Helper()
	var n int64
	row := conn.Raw(`
		SELECT count(*)
		FROM ai_policy_grants a
		JOIN ai_policy_grants b
		  ON a.user_id = b.user_id
		 AND a.id < b.id
		 AND a.status IN ('active','future')
		 AND b.status IN ('active','future')
		 AND a.effective_from < COALESCE(b.effective_to, 'infinity'::timestamptz)
		 AND b.effective_from < COALESCE(a.effective_to, 'infinity'::timestamptz)
		WHERE a.user_id = ?`, userID).Row()
	if err := row.Scan(&n); err != nil {
		t.Fatalf("count overlapping grant windows: %v", err)
	}
	return n
}

// runLateRenewalAgainstFutureGrant is the shared fixture for the two H2 tests: it
// seeds a `future` grant and delivers a renewal whose declared window starts
// before it, but whose occurrence is newer than the future grant's created_at.
func runLateRenewalAgainstFutureGrant(t *testing.T, conn *gorm.DB, uc *GrantUseCase) (entities.AIPolicyGrant, entities.AIPolicyGrant, uuid.UUID) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	userID := uuid.New()
	policy := seedGrantPolicy(t, conn, "premium")
	future := seedFutureGrant(t, conn, userID, policy.ID, now)

	event := planChangedPayload(userID, policy.ID, uuid.New(),
		subscriptioneventtype.EXTENDED,
		now.Add(time.Minute),           // occurredAt: newer than future.created_at
		now.AddDate(0, 0, 15),          // effectiveAt: ignored for EXTENDED
		ptrTime(now.AddDate(0, 0, 45)), // effectiveTo
		30)                             // start = effectiveTo - 30d = now+15d

	if err := uc.HandlePlanChanged(context.Background(), event); err != nil {
		t.Fatalf("a late but valid event must be applied, not refused: %v", err)
	}

	var created entities.AIPolicyGrant
	if err := conn.Where("user_id = ? AND source_event_id = ?", userID, event.SourceEventID).First(&created).Error; err != nil {
		t.Fatalf("read the newly created grant: %v", err)
	}
	return future, created, userID
}

// TestIntegrationLateEventShiftsPastFutureGrant is AS1 against stored state: a
// valid event that lands after a `future` grant is APPLIED (027 turns the old
// permanent refusal into a shift), and its window starts strictly after the
// future grant rather than before it.
func TestIntegrationLateEventShiftsPastFutureGrant(t *testing.T) {
	conn := openGrantTestDB(t)
	cleanupGrantTables(t, conn)
	uc := newGrantIntegrationUC(conn)

	future, created, userID := runLateRenewalAgainstFutureGrant(t, conn, uc)

	wantStart := future.EffectiveFrom.Add(time.Microsecond)
	if !created.EffectiveFrom.Equal(wantStart) {
		t.Errorf("FR-004: expected the new window to start at %v (future.effective_from+1us), got %v",
			wantStart, created.EffectiveFrom)
	}
	if !created.EffectiveFrom.After(future.EffectiveFrom) {
		t.Errorf("FR-004: the new window must start STRICTLY after the future grant, got %v",
			created.EffectiveFrom)
	}
	if n := countOverlappingGrantWindows(t, conn, userID); n != 0 {
		t.Errorf("FR-008: an account must not hold overlapping windows, got %d%s",
			n, describeGrantRows(t, conn, userID))
	}
}

// TestIntegrationFutureGrantIsTruncated is the H2 assertion: with the order floor
// taken from the future grant's effective_from, TruncateGrantsBefore is handed a
// cutoff that actually spans the future row, so the row is closed at that cutoff
// instead of being left to overlap the new one.
func TestIntegrationFutureGrantIsTruncated(t *testing.T) {
	conn := openGrantTestDB(t)
	cleanupGrantTables(t, conn)
	uc := newGrantIntegrationUC(conn)

	future, _, userID := runLateRenewalAgainstFutureGrant(t, conn, uc)

	var after entities.AIPolicyGrant
	if err := conn.Where("id = ?", future.ID).First(&after).Error; err != nil {
		t.Fatalf("re-read the future grant: %v", err)
	}
	wantCutoff := future.EffectiveFrom.Add(time.Microsecond)
	if after.Status != aipolicygrantstatus.Closed {
		t.Errorf("the future grant must be closed by the truncation, got status %q%s",
			after.Status, describeGrantRows(t, conn, userID))
	}
	if after.EffectiveTo == nil || !after.EffectiveTo.Equal(wantCutoff) {
		t.Errorf("the future grant must be truncated at %v, got %v", wantCutoff, after.EffectiveTo)
	}
	if n := countOverlappingGrantWindows(t, conn, userID); n != 0 {
		t.Errorf("FR-008: the future grant must not overlap the new one, got %d overlapping pairs", n)
	}
}

// TestIntegrationStaleEventIsRefusedWithoutClosing is AS2: an event that really is
// older than the account's newest grant is refused, and the refusal has no side
// effects - nothing closed, nothing written (FR-012).
//
// The existing grant's snapshot is deliberately unreadable. Under the old code that
// alone caused the permanent refusal; 027 must refuse this event because of its
// timestamp, not because the snapshot cannot be read.
func TestIntegrationStaleEventIsRefusedWithoutClosing(t *testing.T) {
	conn := openGrantTestDB(t)
	cleanupGrantTables(t, conn)
	uc := newGrantIntegrationUC(conn)

	now := time.Now().UTC().Truncate(time.Microsecond)
	userID := uuid.New()
	policy := seedGrantPolicy(t, conn, "premium")
	prior := seedUnreadableActiveGrant(t, conn, userID, policy.ID, now.Add(-time.Hour), now)

	stale := planChangedPayload(userID, policy.ID, uuid.New(),
		subscriptioneventtype.ACTIVATED,
		now.Add(-2*time.Hour), // occurredAt: older than the stored created_at
		now.Add(-2*time.Hour),
		ptrTime(now.AddDate(0, 0, 30)),
		30)

	err := uc.HandlePlanChanged(context.Background(), stale)
	var rejection *RejectionError
	if !errors.As(err, &rejection) {
		t.Fatalf("a truly old event must be refused, got %T: %v", err, err)
	}
	if rejection.Reason != ReasonStaleEvent {
		t.Errorf("expected reason %q, got %q", ReasonStaleEvent, rejection.Reason)
	}

	var after entities.AIPolicyGrant
	if err := conn.Where("id = ?", prior.ID).First(&after).Error; err != nil {
		t.Fatalf("re-read the existing grant: %v", err)
	}
	if after.Status != aipolicygrantstatus.Active {
		t.Errorf("FR-012: the refusal must not close the existing grant, got status %q", after.Status)
	}
	if after.EffectiveTo != nil {
		t.Errorf("FR-012: the refusal must not move the existing grant's end, got %v", after.EffectiveTo)
	}

	sourceID := uuid.MustParse(stale.SourceEventID)
	var written int64
	if err := conn.Model(&entities.AIPolicyGrant{}).
		Where("source_event_id = ?", sourceID).Count(&written).Error; err != nil {
		t.Fatalf("count grants for the refused event: %v", err)
	}
	if written != 0 {
		t.Errorf("FR-012: the refusal must not write a grant, got %d", written)
	}
}

// TestIntegrationConcurrentEventsLeaveOneActiveGrant is AS3 against stored state:
// two change events for one account, processed at the same time, must leave at
// most one live window - FR-008 / FR-034 of spec 024.
//
// The account starts with an unreadable active grant, which is the state the
// production repair operated on. Whichever event wins is free to close it; the
// loser is expected to be refused as stale, which is a correct outcome rather
// than a failure.
func TestIntegrationConcurrentEventsLeaveOneActiveGrant(t *testing.T) {
	conn := openGrantTestDB(t)
	cleanupGrantTables(t, conn)
	uc := newGrantIntegrationUC(conn)

	now := time.Now().UTC().Truncate(time.Microsecond)
	userID := uuid.New()
	policy := seedGrantPolicy(t, conn, "free-default")
	seedUnreadableActiveGrant(t, conn, userID, policy.ID, now.Add(-time.Hour), now.Add(-time.Hour))

	first := planChangedPayload(userID, policy.ID, uuid.New(),
		subscriptioneventtype.ACTIVATED, now, now, ptrTime(now.AddDate(0, 0, 30)), 30)
	second := first
	second.SourceEventID = uuid.New().String()
	second.OccurredAt = now.Add(time.Second).Format(time.RFC3339Nano)
	second.EffectiveAt = now.Add(time.Second).Format(time.RFC3339Nano)

	errs := runConcurrently(racersForGrantLock*2, func(idx int) error {
		if idx%2 == 0 {
			return uc.HandlePlanChanged(context.Background(), first)
		}
		return uc.HandlePlanChanged(context.Background(), second)
	})
	for i, err := range errs {
		if err == nil {
			continue
		}
		var rejection *RejectionError
		if !errors.As(err, &rejection) || rejection.Reason != ReasonStaleEvent {
			t.Fatalf("racer %d failed: %v", i, err)
		}
	}

	if n := countOverlappingGrantWindows(t, conn, userID); n != 0 {
		t.Errorf("FR-008: concurrent events must not leave overlapping windows, got %d%s",
			n, describeGrantRows(t, conn, userID))
	}

	var active int64
	if err := conn.Model(&entities.AIPolicyGrant{}).
		Where("user_id = ? AND status = ?", userID, aipolicygrantstatus.Active).
		Count(&active).Error; err != nil {
		t.Fatalf("count active grants: %v", err)
	}
	if active != 1 {
		t.Errorf("FR-034: an account must be left with exactly one active grant, got %d%s",
			active, describeGrantRows(t, conn, userID))
	}
}

// TestIntegrationLateDowngradeClosesPriorAtShiftedCutoff is the regression for the
// close-branch defect: the prior active row's effective_from is LATER than the
// late downgrade's event.EffectiveAt, so resolveGrantWindow shifts the new window
// up to orderFloor+1us. The prior row must still be closed, and closed at the
// RESOLVED cutoff - the instant the new row starts, not the unshifted event time.
//
// Before the fix, retirePreviousGrants passed event.EffectiveAt (the past instant)
// to CloseActiveGrants, whose filter is `effective_from <= cutoff`. The prior row
// started after that cutoff, so it was never matched, stayed active, and overlapped
// the new row: two active windows at once, which FR-008 forbids.
//
// This is only reachable for a ClosesPriorGrants type, and only from a fake if the
// fake has a filter to miss - which is why it needs a real database.
func TestIntegrationLateDowngradeClosesPriorAtShiftedCutoff(t *testing.T) {
	conn := openGrantTestDB(t)
	cleanupGrantTables(t, conn)
	uc := newGrantIntegrationUC(conn)

	now := time.Now().UTC().Truncate(time.Microsecond)
	userID := uuid.New()
	policy := seedGrantPolicy(t, conn, "premium")

	// A genuinely active row already in force, starting 5 minutes ago. The late
	// downgrade's effectiveAt is 10 minutes ago - BEFORE that start - which is the
	// condition that makes the new window shift past it.
	prior := entities.AIPolicyGrant{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{ID: uuid.New(), CreatedAt: now.Add(-time.Hour)},
			UpdatedAt:  now.Add(-time.Hour),
		},
		UserID:         userID,
		PolicyID:       policy.ID,
		PlanID:         uuid.New(),
		PlanCode:       "premium-monthly",
		TierRank:       2,
		PolicySnapshot: entities.JSONDocument(`{}`),
		EffectiveFrom:  now.Add(-5 * time.Minute),
		EffectiveTo:    ptrTime(now.Add(time.Hour)),
		Status:         aipolicygrantstatus.Active,
	}
	insertGrantRow(t, conn, &prior)

	downgrade := planChangedPayload(userID, policy.ID, uuid.New(),
		subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE,
		now.Add(time.Minute),     // occurredAt: newer than prior.created_at, so not stale
		now.Add(-10*time.Minute), // effectiveAt: BEFORE prior.effective_from -> shift
		ptrTime(now.AddDate(0, 0, 30)),
		30)

	if err := uc.HandlePlanChanged(context.Background(), downgrade); err != nil {
		t.Fatalf("the late downgrade must be applied, not refused: %v", err)
	}

	// The resolved cutoff is orderFloor + 1us, where orderFloor is the prior row's
	// effective_from (it is later than the stale floor).
	wantCutoff := prior.EffectiveFrom.Add(time.Microsecond)

	var after entities.AIPolicyGrant
	if err := conn.Where("id = ?", prior.ID).First(&after).Error; err != nil {
		t.Fatalf("re-read the prior grant: %v", err)
	}
	if after.Status != aipolicygrantstatus.Closed {
		t.Errorf("the prior active grant must be closed at the shifted cutoff, got status %q%s",
			after.Status, describeGrantRows(t, conn, userID))
	}
	if after.EffectiveTo == nil || !after.EffectiveTo.Equal(wantCutoff) {
		t.Errorf("the prior grant must be closed at %v, got %v", wantCutoff, after.EffectiveTo)
	}

	sourceID := uuid.MustParse(downgrade.SourceEventID)
	var created entities.AIPolicyGrant
	if err := conn.Where("user_id = ? AND source_event_id = ?", userID, sourceID).First(&created).Error; err != nil {
		t.Fatalf("read the newly created grant: %v", err)
	}
	if !created.EffectiveFrom.Equal(wantCutoff) {
		t.Errorf("the new grant must start at the resolved cutoff %v, got %v", wantCutoff, created.EffectiveFrom)
	}

	if n := countOverlappingGrantWindows(t, conn, userID); n != 0 {
		t.Errorf("FR-008: the late downgrade must not leave overlapping windows, got %d%s",
			n, describeGrantRows(t, conn, userID))
	}
}

// ---------------------------------------------------------------------------
// T006b: the two NEGATIVE constraints 027 must not violate.
// ---------------------------------------------------------------------------

// grantTableOnlyRepo answers GetNewestGrantForOrdering - the one grant-table read
// resolveGrantWindow is allowed to make - and nothing else. Any other repository
// method is a nil embedded interface, so calling one panics and the test fails
// loudly rather than passing on a query nobody looked at.
type grantTableOnlyRepo struct {
	repositories.IAICostRepository
	prior *entities.AIPolicyGrant
	reads int
}

func (r *grantTableOnlyRepo) GetNewestGrantForOrdering(context.Context, uuid.UUID) (*entities.AIPolicyGrant, error) {
	r.reads++
	return r.prior, nil
}

// TestAC01ResolveGrantWindowOnlyReadsGrantRows pins AC-01 at the application
// boundary: resolveGrantWindow's entire database surface is the single newest-grant
// read, and that read belongs to the grant table only. The logical table is
// `ai_policy_grants`; the schema it lands in is decided by the connection
// (`search_path`) rather than by the entity, whose TableName() carries no schema.
// It does not, and cannot at this layer, prove the SQL of that read - that contract
// is owned by GetNewestGrantForOrdering in the infrastructure layer, which 027 does
// not touch.
func TestAC01ResolveGrantWindowOnlyReadsGrantRows(t *testing.T) {
	prior := priorGrantForTest(windowBase.Add(-time.Hour), windowBase.Add(-2*time.Hour))
	repo := &grantTableOnlyRepo{prior: prior}
	uc := NewGrantUseCase(repo, &recordingUow{},
		logger.New("test", "", "info", false), NewGrantMetrics(logger.New("test", "", "info", false))).(*GrantUseCase)

	event := activeEvent(windowBase, windowBase.Add(time.Hour), windowBase)
	if _, err := uc.resolveGrantWindow(context.Background(), event); err != nil {
		t.Fatalf("resolveGrantWindow: %v", err)
	}
	if repo.reads != 1 {
		t.Errorf("AC-01: resolveGrantWindow must make exactly one grant-table read, got %d", repo.reads)
	}
}

// TestAC02NoEventTypeBypassesTheOrderingCheck pins the behavioural half of AC-02:
// there is no event type for which the ordering check is skipped. Every valid
// change type is driven through resolveGrantWindow against the same stale state
// and must reach the same refusal.
//
// What this does NOT prove: the absence of a hypothetical source-scoped flag that
// no code reads. That is a property of the source, and it is stated by inspection
// in the report rather than asserted here.
func TestAC02NoEventTypeBypassesTheOrderingCheck(t *testing.T) {
	s := windowBase
	for _, eventType := range []subscriptioneventtype.Type{
		subscriptioneventtype.ACTIVATED,
		subscriptioneventtype.EXTENDED,
		subscriptioneventtype.UPGRADED,
		subscriptioneventtype.OVERLAID,
		subscriptioneventtype.LIFETIME_FALLBACK_RESTORED,
		subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE,
		subscriptioneventtype.AUTO_RENEW_UPDATED,
	} {
		t.Run(string(eventType), func(t *testing.T) {
			// stale floor = created_at = s; the event occurs exactly at s, so it is
			// not strictly newer and must be refused whatever its type.
			prior := priorGrantForTest(s, s.Add(-time.Hour))
			uc := newResolutionUC(&fakeGrantRepo{currentGrant: prior}, &recordingLogger{})

			event := &PlanChangedEvent{
				EventType:    eventType,
				UserID:       uuid.MustParse(validUserID),
				OccurredAt:   s,
				EffectiveAt:  s.Add(-time.Second),
				EffectiveTo:  ptrTime(s.Add(time.Hour)),
				DurationDays: 30,
			}
			_, err := uc.resolveGrantWindow(context.Background(), event)
			requireStaleRejection(t, err)
		})
	}
}
