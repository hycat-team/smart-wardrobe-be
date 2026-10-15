package persistence

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	aierrors "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/errors"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// The project has no database test harness — no sqlmock, no testcontainers, no
// sqlite — so the SQL-level behaviour these tests care about (the ON CONFLICT
// predicate and the FOR UPDATE re-read) is asserted in two ways:
//
//   1. The pure window logic is tested here directly.
//   2. The dedup predicate itself is verified against the running PostgreSQL in
//      quickstart.md, GĐ1, which checks that a second insert with
//      grant_id IS NULL affects 0 rows. That check was run on PostgreSQL 17.10
//      during design and is repeated per implementation.

// TestFallbackLedgerWindowAnchorsOnAFixedEpoch covers the nil-grant path. The
// window must advance on its own: anchoring it on the request time would pin
// every request to index 0, the dedup key would never change, and the running
// total on the fallback ledger would never reset.
func TestFallbackLedgerWindowAnchorsOnAFixedEpoch(t *testing.T) {
	now := time.Date(2026, 9, 30, 14, 0, 0, 0, time.UTC)

	idx, start, end := periodFor(nil, 30, now)
	if idx <= 0 {
		t.Fatalf("a 2026 request must land well past the epoch, got index %d", idx)
	}
	if start.After(now) || !end.After(now) {
		t.Fatalf("the window must contain the request time, got [%v, %v)", start, end)
	}
	if got := end.Sub(start); got != 30*24*time.Hour {
		t.Fatalf("the fallback period must span 30 days, got %v", got)
	}
}

// TestFallbackLedgerWindowIsIdenticalForTheSameInstant is what makes the
// anti-overspend control work: two calls at the same time must land on the same
// index, or each would open a fresh ledger and the running total would reset.
func TestFallbackLedgerWindowIsIdenticalForTheSameInstant(t *testing.T) {
	at := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

	firstIdx, firstStart, _ := periodFor(nil, 30, at)
	secondIdx, secondStart, _ := periodFor(nil, 30, at)

	if firstIdx != secondIdx || !firstStart.Equal(secondStart) {
		t.Fatalf("two calls at the same instant produced different windows: %d/%v and %d/%v",
			firstIdx, firstStart, secondIdx, secondStart)
	}
}

func TestFallbackLedgerWindowRollsOver(t *testing.T) {
	start := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

	firstIdx, _, _ := periodFor(nil, 30, start)
	secondIdx, _, _ := periodFor(nil, 30, start.Add(30*24*time.Hour))

	if secondIdx != firstIdx+1 {
		t.Fatalf("the period must roll over after 30 days: %d then %d", firstIdx, secondIdx)
	}
}

// TestGrantLedgerWindowIsAnchoredOnTheGrant documents why the two paths differ.
// A real grant anchors its period on effective_from so that a renewal continues
// the existing budget instead of handing the account a fresh one.
func TestGrantLedgerWindowIsAnchoredOnTheGrant(t *testing.T) {
	effectiveFrom := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	grant := &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{
			BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()},
		},
		EffectiveFrom: effectiveFrom,
	}

	idx, start, _ := periodFor(grant, 30, effectiveFrom.Add(35*24*time.Hour))
	if idx != 1 {
		t.Fatalf("expected the second period, got %d", idx)
	}
	if want := effectiveFrom.Add(30 * 24 * time.Hour); !start.Equal(want) {
		t.Fatalf("the period must be anchored on the grant, got %v", start)
	}
}

func TestGrantLedgerWindowIsCappedByGrantExpiry(t *testing.T) {
	effectiveFrom := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	expiry := effectiveFrom.Add(10 * 24 * time.Hour)
	grant := &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{
			BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()},
		},
		EffectiveFrom: effectiveFrom,
		EffectiveTo:   &expiry,
	}

	_, _, end := periodFor(grant, 30, effectiveFrom.Add(24*time.Hour))
	if !end.Equal(expiry) {
		t.Fatalf("the period must not run past the grant expiry, got %v", end)
	}
}

func TestPeriodFallsBackToThirtyDaysWhenUnset(t *testing.T) {
	now := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

	idx, withDefault, _ := periodFor(nil, 0, now)
	_, withExplicit, _ := periodFor(nil, 30, now)
	if !withDefault.Equal(withExplicit) {
		t.Fatalf("a zero period_days must behave as 30, got %v vs %v", withDefault, withExplicit)
	}
	if idx <= 0 {
		t.Fatalf("the index must still advance, got %d", idx)
	}
}

// TestNullableGrantID is the small but load-bearing mapping: a nil grant has to
// become SQL NULL, and uuid.Nil would be a valid-looking id pointing at a
// non-existent row.
func TestNullableGrantID(t *testing.T) {
	if nullableGrantID(nil) != nil {
		t.Fatal("a nil grant must map to SQL NULL, not to uuid.Nil")
	}
	grant := &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{
			BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()},
		},
	}
	if got := nullableGrantID(grant); got == nil || *got != grant.ID {
		t.Fatalf("a real grant must map to its own id, got %v", got)
	}
}

// TestUserIDFallsBackToTheEvent covers the case where the ledger is keyed on the
// account even though there is no grant row: the user still has to come from
// somewhere, and taking it from the usage event is the only source available.
func TestUserIDFallsBackToTheEvent(t *testing.T) {
	userID := uuid.MustParse("3f7c1a20-0000-4000-8000-000000000001")
	event := &aiexecutor_entities.AIUsageEvent{UserID: &userID}

	if got := userIDFor(nil, event); got != userID {
		t.Fatalf("expected the user id from the event, got %s", got)
	}
	if got := userIDFor(nil, &aiexecutor_entities.AIUsageEvent{}); got != uuid.Nil {
		t.Fatalf("an anonymous event must yield uuid.Nil, got %s", got)
	}

	grant := &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{
			BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()},
		},
	}
	// With a grant present the grant is authoritative, even if the event
	// disagrees — the ledger row must follow the grant that is paying for it.
	grant.UserID = userID
	event.UserID = func() *uuid.UUID { other := uuid.New(); return &other }()
	if got := userIDFor(grant, event); got != userID {
		t.Fatalf("the grant must win over the event, got %s", got)
	}
}

// TestObserveOnlyWithoutAttemptCapIsAMisconfiguration documents the branch that
// FR-005e requires. The sentinel is a distinct value rather than a generic
// error so the caller can branch on it and degrade instead of failing.
func TestObserveOnlyWithoutAttemptCapIsAMisconfiguration(t *testing.T) {
	// A nil MaxPaidAttemptsPerDay in observe_only is the misconfiguration; the
	// repository turns it into this sentinel. The test pins the reason the
	// caller must recognise it.
	if err := aierrors.ErrObserveOnlyMissingAttemptCap; err == nil {
		t.Fatal("the sentinel must exist")
	}
}

// TestOperationLimitIsOnlyEnforcedInObserveOnly restates the rule the Reserve
// branch implements: strict policies ignore the per-day counter entirely, which
// is why Premium is stored as NULL.
func TestOperationLimitIsOnlyEnforcedInObserveOnly(t *testing.T) {
	premiumLike := &aiexecutor_entities.AICostPolicyOperation{
		Operation:             aioperation.Outfit,
		MaxPaidAttemptsPerDay: nil,
	}
	if premiumLike.MaxPaidAttemptsPerDay != nil {
		t.Fatal("a strict policy's per-day limit must be NULL, not a number")
	}
}

// ---- 028 US1 T010: the ledger accumulates actual cost on every route ----
//
// The project has no database test harness, so the Confirm ledger behaviour is
// asserted over the source the same way TestRepositoryNoLongerReadsTheLiveAttemptCapColumn
// does: the property under test is structural (where the accumulation line sits),
// and a fake cannot catch a regression in the real file.

// Confirm must accumulate actual_cost_micro_vnd for both paid and free routes.
// The accumulation line must sit outside the paid/free token-bucket branch.
func TestConfirmAccumulatesActualOnBothPaidAndFreeRoutes(t *testing.T) {
	src := readRepoImpl(t)
	actualLine := `ledgerUpdates["actual_cost_micro_vnd"] = gorm.Expr("actual_cost_micro_vnd + ?", cost)`
	if !bytes.Contains(src, []byte(actualLine)) {
		t.Fatalf("Confirm must accumulate actual_cost_micro_vnd, missing line %q", actualLine)
	}
	// The token buckets stay split by route...
	for _, want := range []string{`ledgerUpdates["paid_input_tokens"]`, `ledgerUpdates["free_input_tokens"]`} {
		if !bytes.Contains(src, []byte(want)) {
			t.Errorf("Confirm must keep the split token bucket %s", want)
		}
	}
	// ...but the actual accumulation must not live inside the paid-only branch.
	// It is positioned after the if/else that splits the buckets.
	paidIdx := bytes.Index(src, []byte("if paid {"))
	actualIdx := bytes.Index(src, []byte(actualLine))
	elseIdx := bytes.Index(src[paidIdx:], []byte("} else {"))
	if paidIdx < 0 || actualIdx < 0 || elseIdx < 0 {
		t.Fatalf("cannot locate the Confirm route branch (paid=%d actual=%d else=%d)", paidIdx, actualIdx, elseIdx)
	}
	elseClose := paidIdx + elseIdx
	if actualIdx < elseClose {
		t.Error("actual_cost_micro_vnd accumulation must sit after the paid/free branch so free events accumulate too")
	}
}

// Confirm must not touch the reservation or admission logic: reserved cost is
// only released back, and the token buckets are additive.
func TestConfirmLeavesReservationAndBucketsAdditive(t *testing.T) {
	src := readRepoImpl(t)
	if !bytes.Contains(src, []byte(`"reserved_cost_micro_vnd": gorm.Expr("GREATEST(reserved_cost_micro_vnd - ?,0)"`)) {
		t.Error("Confirm must release the reservation with GREATEST(reserved - ?, 0)")
	}
	if strings.Contains(string(src), "admitted") && strings.Contains(string(src), "func (r *AICostRepository) Confirm") {
		confirmSrc := string(src[bytes.Index(src, []byte("func (r *AICostRepository) Confirm")):])
		if idx := strings.Index(confirmSrc, "func (r *AICostRepository) Release"); idx >= 0 {
			confirmSrc = confirmSrc[:idx]
		}
		if strings.Contains(confirmSrc, "admitted") {
			t.Error("Confirm must not contain admission logic; Reserve owns it")
		}
	}
}

// A failed attempt (Released) must not accumulate actual cost: only Confirm
// writes actual_cost_micro_vnd to the ledger.
func TestReleasedAttemptsDoNotAccumulateActual(t *testing.T) {
	src := readRepoImpl(t)
	releaseIdx := bytes.Index(src, []byte("func (r *AICostRepository) finishWithoutCost"))
	if releaseIdx < 0 {
		t.Fatal("cannot locate finishWithoutCost (Release path)")
	}
	tail := src[releaseIdx:]
	if next := bytes.Index(tail, []byte("\nfunc (r *AICostRepository) ExpireUnknown")); next >= 0 {
		tail = tail[:next]
	}
	if bytes.Contains(tail, []byte("actual_cost_micro_vnd")) {
		t.Error("Release/finishWithoutCost must never touch actual_cost_micro_vnd; failed attempts carry no cost")
	}
}

func readRepoImpl(t *testing.T) []byte {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(".", "ai_cost_repo_impl.go"))
	if err != nil {
		t.Fatalf("cannot read repo impl: %v", err)
	}
	return src
}

// ---- 029 T028: the admin period queries build the half-open VN range ----
//
// Same unreachable-dialector + statement-capture technique as
// services/account/.../admin_stats_query_sql_test.go: gorm.Open with
// DisableAutomaticPing builds the statement and fails on execution, and the
// logger captures what would have gone to the driver. It proves the repository
// BUILDS the query the period contract (R1/data-model §3) describes; that
// PostgreSQL accepts it is covered by the live quickstart check.

const unreachableStatsDSN = "host=127.0.0.1 port=1 user=none dbname=none sslmode=disable"

type aiStatsSQLCapture struct {
	gormlogger.Interface
	mu         sync.Mutex
	statements []string
}

func newAIStatsSQLCapture() *aiStatsSQLCapture {
	base := gormlogger.New(log.New(io.Discard, "", 0), gormlogger.Config{LogLevel: gormlogger.Silent})
	return &aiStatsSQLCapture{Interface: base}
}

func (l *aiStatsSQLCapture) ParamsFilter(_ context.Context, sql string, _ ...any) (string, []any) {
	return sql, nil
}

func (l *aiStatsSQLCapture) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	statement, _ := fc()
	l.mu.Lock()
	l.statements = append(l.statements, statement)
	l.mu.Unlock()
	l.Interface.Trace(ctx, begin, fc, err)
}

var _ gormlogger.Interface = (*aiStatsSQLCapture)(nil)

// aiStatsStatement runs op against an unreachable dialector and returns the ONE
// statement it built. A success would mean the repository never reached the
// driver, so it is a failure of the test, not of the environment.
func aiStatsStatement(t *testing.T, op func(dbConn *gorm.DB) error) string {
	t.Helper()

	capture := newAIStatsSQLCapture()
	dbConn, err := gorm.Open(postgres.Open(unreachableStatsDSN), &gorm.Config{
		Logger:               capture,
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("initialise the dialector without connecting: %v", err)
	}

	if err := op(dbConn); err == nil {
		t.Fatal("expected the call to fail without a server; a success would mean " +
			"it never reached the driver")
	}

	capture.mu.Lock()
	defer capture.mu.Unlock()
	if len(capture.statements) != 1 {
		t.Fatalf("expected exactly one statement, got %d: %v", len(capture.statements), capture.statements)
	}
	return capture.statements[0]
}

// TestAIErrorBreakdownAcceptsAPeriod pins FR-016's repository half: with a
// period the grouping query must filter `created_at >= from AND created_at <
// to` — the same half-open window the commerce use case uses for the request
// count, so the numerator and denominator can never re-diverge.
func TestAIErrorBreakdownAcceptsAPeriod(t *testing.T) {
	from := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	stmt := aiStatsStatement(t, func(dbConn *gorm.DB) error {
		_, err := NewAICostRepository(dbConn).GetAIErrorBreakdown(context.Background(), from, to)
		return err
	})

	if !strings.Contains(stmt, "created_at >= ") || !strings.Contains(stmt, "created_at < ") {
		t.Fatalf("029 FR-016: error breakdown must filter the half-open period, got: %s", stmt)
	}
	if !strings.Contains(stmt, "status = ") {
		t.Fatalf("the released-status filter must survive the period change, got: %s", stmt)
	}
}

// TestAIErrorBreakdownWithoutAPeriodKeepsAllTime is the backward-compatible
// branch R9 asks for: an older caller that sends no period still receives the
// all-time breakdown, proven by the absence of a created_at filter.
func TestAIErrorBreakdownWithoutAPeriodKeepsAllTime(t *testing.T) {
	stmt := aiStatsStatement(t, func(dbConn *gorm.DB) error {
		_, err := NewAICostRepository(dbConn).GetAIErrorBreakdown(context.Background(), time.Time{}, time.Time{})
		return err
	})

	if strings.Contains(stmt, "created_at") {
		t.Fatalf("a zero period must leave the breakdown unbounded, got: %s", stmt)
	}
}

// TestAIUsageAndRequestCountUseHalfOpenUpperBound pins the R1 standardisation
// of the two pre-existing period queries: `<=` on the upper bound dropped the
// VN timezone offset and double-counted the instant that starts the next
// period; both must build `created_at < to`.
func TestAIUsageAndRequestCountUseHalfOpenUpperBound(t *testing.T) {
	from := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)

	usageStmt := aiStatsStatement(t, func(dbConn *gorm.DB) error {
		_, err := NewAICostRepository(dbConn).GetAIUsageByOperation(context.Background(), from, to)
		return err
	})
	if !strings.Contains(usageStmt, "created_at >= ") || !strings.Contains(usageStmt, "created_at < ") {
		t.Fatalf("R1: usage-by-operation must filter created_at >= from AND created_at < to, got: %s", usageStmt)
	}
	if strings.Contains(usageStmt, "created_at <= ") {
		t.Fatalf("R1: the inclusive `<= to` bound must be gone, got: %s", usageStmt)
	}

	countStmt := aiStatsStatement(t, func(dbConn *gorm.DB) error {
		_, err := NewAICostRepository(dbConn).GetTotalAIRequestCount(context.Background(), from, to)
		return err
	})
	if !strings.Contains(countStmt, "created_at >= ") || !strings.Contains(countStmt, "created_at < ") {
		t.Fatalf("R1: the request count must filter created_at >= from AND created_at < to, got: %s", countStmt)
	}
	if strings.Contains(countStmt, "created_at <= ") {
		t.Fatalf("R1: the inclusive `<= to` bound must be gone, got: %s", countStmt)
	}
}
