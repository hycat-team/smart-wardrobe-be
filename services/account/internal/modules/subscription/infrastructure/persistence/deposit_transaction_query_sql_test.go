package persistence

import (
	"context"
	"io"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// 029 T018 / FR-010 / R1 + R10 — the admin deposit list period, pinned on the
// SQL and then on a database that runs it.
//
// WHAT CHANGED AND WHY THE OLD ASSERTIONS WOULD NOT HAVE CAUGHT IT. The list
// used `created_at <= ?` with a bound parsed at UTC midnight, so `toDate` cut
// off almost the whole Vietnamese final day (7 hours of it) — a filter that
// answers with fewer rows than exist is indistinguishable, on screen, from a
// quiet day. The fix is the shared half-open convention (data-model §3): the
// handler parses VN calendar days into `[from, to)` and the repository compares
// `>= from AND < to`.
//
// THE TECHNIQUE. `ListAdmin` runs a COUNT and then a page query, and without a
// server the old unreachable-dialector capture only ever reached the COUNT —
// the ORDER BY lives on the page statement and was unreachable. The activity
// repository's test (T007) sidesteps that with an in-memory sqlite database
// that runs BOTH statements, so the capture sees both; this file does the same.
// sqlite accepts `?` bindings and the plain comparisons here; the timestamptz
// idiom that sqlite rejects is not in this query.
//
// WHAT IT STILL DOES NOT PROVE: no PostgreSQL server means no proof that the
// exact statement runs there. It proves the repository BUILDS the query the
// period convention describes, and — on sqlite — that the boundary row falls on
// the side the convention says it does.

// depositSQLCapture records every statement GORM reports while delegating to a
// silent base logger. It keeps the statement's placeholders (`ParamsFilter`
// returns no vars), so the assertions below read the SHAPE of the SQL; the
// bound instants are pinned separately, at the handler (T019), where they are
// typed rather than rendered.
type depositSQLCapture struct {
	gormlogger.Interface
	mu         sync.Mutex
	statements []string
}

func newDepositSQLCapture() *depositSQLCapture {
	base := gormlogger.New(log.New(io.Discard, "", 0), gormlogger.Config{LogLevel: gormlogger.Silent})
	return &depositSQLCapture{Interface: base}
}

func (l *depositSQLCapture) ParamsFilter(_ context.Context, sql string, _ ...any) (string, []any) {
	return sql, nil
}

func (l *depositSQLCapture) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	statement, _ := fc()
	l.mu.Lock()
	l.statements = append(l.statements, statement)
	l.mu.Unlock()
	l.Interface.Trace(ctx, begin, fc, err)
}

func (l *depositSQLCapture) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.statements...)
}

// reset drops what setup wrote (the DROP + CREATE TABLE), so the pinning test
// asserts only on the statements the repository built.
func (l *depositSQLCapture) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.statements = nil
}

var _ gormlogger.Interface = (*depositSQLCapture)(nil)

// openDepositSQLite builds the table the ListAdmin queries touch. The DDL is
// written out rather than `AutoMigrate`d because AutoMigrate replays
// PostgreSQL-isms (`gen_random_uuid()`, `numeric(12,2)`) that sqlite rejects —
// the same reason the identity activity test writes its own DDL. The column set
// is the entity's full 43 columns: the page query selects them explicitly, so a
// missing column would be an error rather than a difference in the assertions.
func openDepositSQLite(t *testing.T, capture gormlogger.Interface) *gorm.DB {
	t.Helper()

	cfg := &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               gormlogger.Default.LogMode(gormlogger.Silent),
	}
	if capture != nil {
		cfg.Logger = capture
	}
	dbConn, err := gorm.Open(sqlite.Open("file:list-admin-deposits?mode=memory&cache=shared&_busy_timeout=5000"), cfg)
	if err != nil {
		t.Fatalf("open the in-memory database: %v", err)
	}

	if err := dbConn.Exec(`DROP TABLE IF EXISTS deposit_transactions`).Error; err != nil {
		t.Fatalf("drop deposit_transactions: %v", err)
	}
	if err := dbConn.Exec(`
		CREATE TABLE deposit_transactions (
			id TEXT PRIMARY KEY,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			user_id TEXT NOT NULL,
			amount NUMERIC NOT NULL,
			currency TEXT NOT NULL,
			status INTEGER NOT NULL,
			transaction_type TEXT NOT NULL,
			subscription_plan_id TEXT,
			order_code INTEGER NOT NULL,
			provider TEXT NOT NULL,
			payment_link_id TEXT,
			provider_status TEXT,
			successful_provider_reference TEXT,
			gateway_details TEXT,
			payment_url TEXT,
			expires_at DATETIME,
			next_reconciliation_at DATETIME,
			reconciliation_attempts INTEGER NOT NULL DEFAULT 0,
			processing_token TEXT,
			processing_lease_until DATETIME,
			last_provider_error_code TEXT,
			last_provider_error_at DATETIME,
			failure_reason TEXT,
			refunded_at DATETIME,
			refund_amount NUMERIC,
			refund_reason TEXT,
			cancelled_at DATETIME,
			expired_at DATETIME,
			plan_code TEXT,
			plan_name TEXT,
			tier_rank INTEGER,
			plan_kind INTEGER,
			purchased_duration_days INTEGER,
			expected_amount NUMERIC NOT NULL DEFAULT 0,
			benefit_snapshot JSON,
			pricing_version INTEGER,
			benefit_resolution TEXT,
			benefit_applied_at DATETIME,
			benefit_result_snapshot JSON,
			wallet_credit_amount NUMERIC,
			subscription_version_before INTEGER,
			subscription_version_after INTEGER
		)`).Error; err != nil {
		t.Fatalf("create deposit_transactions: %v", err)
	}
	t.Cleanup(func() {
		_ = dbConn.Exec(`DROP TABLE IF EXISTS deposit_transactions`).Error
	})
	return dbConn
}

// insertDepositRow writes one deposit transaction at a given instant.
func insertDepositRow(t *testing.T, dbConn *gorm.DB, createdAt time.Time, status depositstatus.DepositStatus) uuid.UUID {
	t.Helper()
	id := uuid.New()
	err := dbConn.Exec(`
		INSERT INTO deposit_transactions
			(id, created_at, updated_at, user_id, amount, currency, status, transaction_type,
			 order_code, provider, reconciliation_attempts, expected_amount)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id.String(), createdAt, createdAt, uuid.New().String(), 100000, "VND", int16(status),
		"direct_purchase", createdAt.UnixNano(), "PAYOS", 0, 100000,
	).Error
	if err != nil {
		t.Fatalf("insert deposit transaction: %v", err)
	}
	return id
}

// TestListAdminDepositTransactionsSQLPinsTheHalfOpenPeriod pins the statement
// both halves of the repository build: the COUNT and the page must carry
// `created_at >= ? AND created_at < ?` — never `<=` — and the page must be
// ordered `created_at DESC` like before.
func TestListAdminDepositTransactionsSQLPinsTheHalfOpenPeriod(t *testing.T) {
	capture := newDepositSQLCapture()
	dbConn := openDepositSQLite(t, capture)
	repo := NewDepositTransactionRepository(dbConn)
	capture.reset() // drop the DDL

	status := depositstatus.FailedLegacy
	from := time.Date(2026, 10, 15, 0, 0, 0, 0, timeutils.VnLoc())
	to := time.Date(2026, 10, 16, 0, 0, 0, 0, timeutils.VnLoc())

	_, _, err := repo.ListAdmin(context.Background(), repositories.AdminDepositTransactionFilter{
		Status:   &status,
		FromDate: &from,
		ToDate:   &to,
	}, 2, 50)
	if err != nil {
		t.Fatalf("ListAdmin against sqlite: %v", err)
	}

	captured := capture.snapshot()
	if len(captured) != 2 {
		t.Fatalf("ListAdmin must build exactly a COUNT and a page query, got %d: %v", len(captured), captured)
	}
	var countStmt, pageStmt string
	for _, stmt := range captured {
		if strings.Contains(stmt, "count(*)") {
			countStmt = stmt
		} else {
			pageStmt = stmt
		}
	}
	if countStmt == "" || pageStmt == "" {
		t.Fatalf("one statement must be the COUNT and one the page: %v", captured)
	}

	for name, stmt := range map[string]string{"count": countStmt, "page": pageStmt} {
		if !strings.Contains(stmt, "created_at >= ?") {
			t.Errorf("%s must lower-bound the period with `created_at >= ?`, got: %s", name, stmt)
		}
		if !strings.Contains(stmt, "created_at < ?") {
			t.Errorf("%s must use the half-open upper bound `created_at < ?` (R1) — with `<=` the "+
				"filter includes an instant the next period also claims, got: %s", name, stmt)
		}
		if strings.Contains(stmt, "created_at <= ") {
			t.Errorf("%s still carries the old inclusive upper bound `created_at <=`; the ToDate "+
				"filter is now the exclusive end of [from, to), so a toDate row would be counted "+
				"twice across two periods, got: %s", name, stmt)
		}
		if !strings.Contains(stmt, "status = ?") {
			t.Errorf("%s must keep the status filter (FR-010), got: %s", name, stmt)
		}
	}
	if strings.Contains(countStmt, "ORDER BY") {
		t.Errorf("the COUNT must not carry an ORDER BY; only the page query orders, got: %s", countStmt)
	}

	if !strings.Contains(pageStmt, "ORDER BY created_at DESC") {
		t.Errorf("the page keeps the newest-first order (FR-010), got: %s", pageStmt)
	}
	if !strings.Contains(pageStmt, "LIMIT 50") || !strings.Contains(pageStmt, "OFFSET 50") {
		t.Errorf("page 2 limit 50 must render LIMIT 50 OFFSET 50, got: %s", pageStmt)
	}
}

// TestListAdminDepositTransactionsExcludesTheUpperBoundary runs the period the
// handler now builds (VN calendar days, `[from, to)`) against rows placed on
// the boundaries themselves.
//
// THE BOUNDARY ROW IS THE WHOLE TEST. A row created at exactly 00:00 of the day
// after `toDate` belongs to the NEXT Vietnamese day, so it must not appear in
// this period; with the old `<=` it did. The 23:30 row is the quickstart's
// scenario-2 case: it is inside the 15th in Vietnam, and under the old UTC
// parse (toDate = midnight UTC) it used to be dropped.
func TestListAdminDepositTransactionsExcludesTheUpperBoundary(t *testing.T) {
	dbConn := openDepositSQLite(t, nil)
	repo := NewDepositTransactionRepository(dbConn)

	// Out of the period: the day before, and exactly the exclusive upper bound.
	insertDepositRow(t, dbConn, time.Date(2026, 10, 14, 23, 59, 0, 0, timeutils.VnLoc()), depositstatus.Success)
	insertDepositRow(t, dbConn, time.Date(2026, 10, 16, 0, 0, 0, 0, timeutils.VnLoc()), depositstatus.Success)
	// Inside the period: the quickstart's 23:30 VN boundary case.
	lateOnTheDayAt := time.Date(2026, 10, 15, 23, 30, 0, 0, timeutils.VnLoc())
	lateOnTheDay := insertDepositRow(t, dbConn, lateOnTheDayAt, depositstatus.Success)

	from, to := timeutils.StartOfDayInVN(lateOnTheDayAt), timeutils.EndOfDayInVN(lateOnTheDayAt)
	if from.Day() != 15 || to.Day() != 16 {
		t.Fatalf("sanity: the period must be the 15th VN day, got [%s, %s)", from, to)
	}

	rows, total, err := repo.ListAdmin(context.Background(), repositories.AdminDepositTransactionFilter{
		FromDate: &from,
		ToDate:   &to,
	}, 1, 20)
	if err != nil {
		t.Fatalf("ListAdmin against sqlite: %v", err)
	}

	if total != 1 || len(rows) != 1 {
		t.Fatalf("the period [%s, %s) must contain exactly the 23:30 row, got total=%d rows=%d",
			from, to, total, len(rows))
	}
	if rows[0].ID != lateOnTheDay {
		t.Errorf("row 1 = %s, want the 23:30 VN row %s (the previous-day and next-midnight rows must stay out)",
			rows[0].ID, lateOnTheDay)
	}

	// The status filter is unchanged by the period fix: it must still select by
	// the numeric status, and `failed_legacy` must be one of the accepted inputs
	// via the same numeric value the column stores.
	failedLegacy := insertDepositRow(t, dbConn, time.Date(2026, 10, 15, 9, 0, 0, 0, timeutils.VnLoc()), depositstatus.FailedLegacy)
	status := depositstatus.FailedLegacy
	rows, total, err = repo.ListAdmin(context.Background(), repositories.AdminDepositTransactionFilter{
		Status:   &status,
		FromDate: &from,
		ToDate:   &to,
	}, 1, 20)
	if err != nil {
		t.Fatalf("ListAdmin with the status filter: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].ID != failedLegacy {
		t.Errorf("status=failed_legacy must select exactly the failed_legacy row, got total=%d rows=%v",
			total, rows)
	}
}
