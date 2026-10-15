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
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Admin-stats SQL pinning (fix false admin metrics).
//
// Three repository queries were silently wrong and every behavioural test
// stayed green, because all of them went through fakes or needed a live
// database. This file pins the STATEMENTS the repositories build, using the
// same unreachable-dialector + statement-capture technique as
// campaign_repo_list_query_sql_test.go: gorm.Open with DisableAutomaticPing
// builds the statement and fails on execution, and the logger captures what
// would have been handed to the driver.
//
// WHAT IT STILL DOES NOT PROVE. No server means no proof that PostgreSQL
// accepts the statement or that the numbers come back right. It proves the
// repository BUILDS the query the fix describes. The behavioural half is
// covered where sqlite can run the same SQL; FILTER and timestamptz idioms
// that sqlite rejects need TEST_DATABASE_URL.

const unreachableStatsDSN = "host=127.0.0.1 port=1 user=none dbname=none sslmode=disable"

type statsSQLCapture struct {
	gormlogger.Interface
	mu         sync.Mutex
	statements []string
}

func newStatsSQLCapture() *statsSQLCapture {
	base := gormlogger.New(log.New(io.Discard, "", 0), gormlogger.Config{LogLevel: gormlogger.Silent})
	return &statsSQLCapture{Interface: base}
}

func (l *statsSQLCapture) ParamsFilter(_ context.Context, sql string, _ ...any) (string, []any) {
	return sql, nil
}

func (l *statsSQLCapture) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	statement, _ := fc()
	l.mu.Lock()
	l.statements = append(l.statements, statement)
	l.mu.Unlock()
	l.Interface.Trace(ctx, begin, fc, err)
}

var _ gormlogger.Interface = (*statsSQLCapture)(nil)

// statsStatement runs op against an unreachable dialector and returns the ONE
// statement it built. A success would mean the repository never reached the
// driver, so it is a failure of the test, not of the environment.
func statsStatement(t *testing.T, op func(dbConn *gorm.DB) error) string {
	t.Helper()

	capture := newStatsSQLCapture()
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

// The revenue column must filter on success: the old query summed every
// PayOS status (pending, failed, cancelled, expired, refunded), so the
// reported revenue was larger than the money that actually moved.
//
// 029 T026 extends the pinning to the period: the stats query must honour the
// half-open VN range [from, to) (R1/data-model §3) instead of deriving a day
// range from a snapshot date internally.
func TestAdminStatsRevenueSumsSuccessOnly(t *testing.T) {
	from := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	stmt := statsStatement(t, func(dbConn *gorm.DB) error {
		_, err := NewDepositTransactionRepository(dbConn).GetAdminSubscriptionStats(context.Background(), from, to)
		return err
	})

	if !strings.Contains(stmt, "SUM(amount) FILTER (WHERE status") {
		t.Fatalf("revenue must be a status-filtered SUM, got: %s", stmt)
	}
	// The logger inlines bound values, so the success constant renders as
	// `status = 1` here; the production code binds depositstatus.Success.
	for _, alias := range []string{"total_revenue", "success_count", "pending_count", "failed_count", "voided_count"} {
		if !strings.Contains(stmt, alias) {
			t.Fatalf("expected the status breakdown alias %q in the statement, got: %s", alias, stmt)
		}
	}
	if !strings.Contains(stmt, "created_at >= ") || !strings.Contains(stmt, "created_at < ") {
		t.Fatalf("029 R1: the stats query must filter the half-open range created_at >= from AND created_at < to, got: %s", stmt)
	}
}

// 029 T037/R6 (data-model §4) pins the two scopes the stats statement must
// carry at once:
//   - the five payOS counters cover EVERY payOS money line (direct purchase +
//     wallet top-up), so no transaction_type condition may sit in the WHERE;
//   - total_subscription_revenue stays direct-purchase-only and moves its type
//     filter INSIDE the status FILTER;
//   - the new total_payos_success_value sums the successful amount over both
//     types, filtered by status only.
func TestAdminStatsPayOSScopeCoversBothTransactionTypes(t *testing.T) {
	from := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	stmt := statsStatement(t, func(dbConn *gorm.DB) error {
		_, err := NewDepositTransactionRepository(dbConn).GetAdminSubscriptionStats(context.Background(), from, to)
		return err
	})

	if strings.Contains(stmt, "WHERE transaction_type") {
		t.Fatalf("029 R6: the payOS counters must span both transaction types, "+
			"so no WHERE-level transaction_type filter may exist, got: %s", stmt)
	}
	if !strings.Contains(stmt, "AND transaction_type = ") {
		t.Fatalf("029 R6: total_subscription_revenue must keep its direct-purchase scope "+
			"inside the FILTER (status = success AND transaction_type = direct_purchase), got: %s", stmt)
	}
	if strings.Count(stmt, "transaction_type") != 1 {
		t.Fatalf("029 R6: only the revenue scope may narrow by transaction_type, got: %s", stmt)
	}

	// Two status-filtered SUMs: the revenue and the new payOS success value.
	if strings.Count(stmt, "SUM(amount) FILTER (WHERE status") != 2 {
		t.Fatalf("029 R6: expected the revenue SUM and a SUM for total_payos_success_value, got: %s", stmt)
	}
	if !strings.Contains(stmt, "AS total_payos_success_value") {
		t.Fatalf("029 R6: total_payos_success_value must be part of the stats statement, got: %s", stmt)
	}
	if !strings.Contains(stmt, "SUM(amount) FILTER (WHERE status = 1), 0) AS total_payos_success_value") {
		t.Fatalf("029 R6: total_payos_success_value = SUM(amount) over status=success with NO "+
			"transaction_type filter, got: %s", stmt)
	}
	if !strings.Contains(stmt, "SUM(amount) FILTER (WHERE status = 1 AND transaction_type = 'direct_purchase')") {
		t.Fatalf("029 R6: total_subscription_revenue = SUM(amount) over status=success AND "+
			"transaction_type=direct_purchase, got: %s", stmt)
	}
}

// insertAdminStatsDepositRow writes one deposit transaction with an explicit
// transaction type and amount, the two axes the payOS scope changes turn on.
func insertAdminStatsDepositRow(t *testing.T, dbConn *gorm.DB, createdAt time.Time, status depositstatus.DepositStatus, txType string, amount int64) {
	t.Helper()
	err := dbConn.Exec(`
		INSERT INTO deposit_transactions
			(id, created_at, updated_at, user_id, amount, currency, status, transaction_type,
			 order_code, provider, reconciliation_attempts, expected_amount)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), createdAt, createdAt, uuid.New().String(), amount, "VND", int16(status),
		txType, createdAt.UnixNano(), "PAYOS", 0, amount,
	).Error
	if err != nil {
		t.Fatalf("insert deposit transaction: %v", err)
	}
}

// TestAdminStatsScopesRunOnSQLite is the behavioural half of T037: the capture
// test above proves the statement text; this one runs it on sqlite and reads
// the DTO, which is also the only thing that can catch a column-alias →
// struct-field mapping mismatch (the scan silently leaves the field zero).
//
// Fixture: in period, one successful direct purchase (1,000), one failed
// direct purchase (2,000), one successful wallet top-up (500) and one pending
// wallet top-up (700); out of period, a 9,999 success that must be ignored.
func TestAdminStatsScopesRunOnSQLite(t *testing.T) {
	dbConn := openDepositSQLite(t, nil)
	repo := NewDepositTransactionRepository(dbConn)

	day := time.Date(2026, 10, 15, 12, 0, 0, 0, timeutils.VnLoc())
	outside := time.Date(2026, 10, 13, 12, 0, 0, 0, timeutils.VnLoc())

	insertAdminStatsDepositRow(t, dbConn, day, depositstatus.Success, "direct_purchase", 1000)
	insertAdminStatsDepositRow(t, dbConn, day, depositstatus.FailedLegacy, "direct_purchase", 2000)
	insertAdminStatsDepositRow(t, dbConn, day, depositstatus.Success, "wallet_topup", 500)
	insertAdminStatsDepositRow(t, dbConn, day, depositstatus.Pending, "wallet_topup", 700)
	insertAdminStatsDepositRow(t, dbConn, outside, depositstatus.Success, "direct_purchase", 9999)

	from, to := timeutils.StartOfDayInVN(day), timeutils.EndOfDayInVN(day)
	stats, err := repo.GetAdminSubscriptionStats(context.Background(), from, to)
	if err != nil {
		t.Fatalf("GetAdminSubscriptionStats against sqlite: %v", err)
	}

	// The counters span both transaction types (029 R6/data-model §4).
	if stats.PayOSTransactionCount != 4 {
		t.Errorf("total count must span direct purchase + wallet top-up, got %d want 4", stats.PayOSTransactionCount)
	}
	if stats.PayOSSuccessCount != 2 {
		t.Errorf("success count must span both types (direct 1,000 + top-up 500), got %d want 2", stats.PayOSSuccessCount)
	}
	if stats.PayOSPendingCount != 1 {
		t.Errorf("pending count must include the wallet top-up, got %d want 1", stats.PayOSPendingCount)
	}
	if stats.PayOSFailedCount != 1 {
		t.Errorf("failed count must include the direct purchase, got %d want 1", stats.PayOSFailedCount)
	}
	if stats.PayOSVoidedCount != 0 {
		t.Errorf("voided count must be 0, got %d", stats.PayOSVoidedCount)
	}

	// Revenue stays subscriptions-only: the successful direct purchase.
	if !stats.TotalSubscriptionRevenue.Equal(decimal.NewFromInt(1000)) {
		t.Errorf("total_subscription_revenue must stay direct_purchase-only (1,000, failed line excluded), got %s",
			stats.TotalSubscriptionRevenue)
	}
	// The new value covers successful money on BOTH types.
	if !stats.TotalPayOSSuccessValue.Equal(decimal.NewFromInt(1500)) {
		t.Errorf("total_payos_success_value must sum success over both types (1,000 + 500), got %s",
			stats.TotalPayOSSuccessValue)
	}
}

// The distribution must count active subscriptions only: the old GROUP BY
// counted expired rows, so a lapsed premium user kept showing up in
// "premium-monthly".
func TestDistributionCountsActiveOnly(t *testing.T) {
	stmt := statsStatement(t, func(dbConn *gorm.DB) error {
		_, err := NewUserSubscriptionRepository(dbConn).GetDistributionByPlanCode(context.Background(), time.Time{}, time.Time{})
		return err
	})

	if !strings.Contains(stmt, "expires_at IS NULL OR expires_at >") {
		t.Fatalf("distribution must filter on active subscriptions, got: %s", stmt)
	}
}

// 029 T026/R7: with a period, the distribution counts users whose current
// plan window OVERLAPS the requested [from, to) — the window started before
// the period ends and has not expired before it starts (data-model §5). The
// no-period branch above stays as it was.
func TestDistributionOverlapsTheRequestedPeriod(t *testing.T) {
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	stmt := statsStatement(t, func(dbConn *gorm.DB) error {
		_, err := NewUserSubscriptionRepository(dbConn).GetDistributionByPlanCode(context.Background(), from, to)
		return err
	})

	if !strings.Contains(stmt, "started_at < ") {
		t.Fatalf("029 R7: the bounded distribution must drop windows that start at/after to, got: %s", stmt)
	}
	if !strings.Contains(stmt, "expires_at IS NULL OR expires_at >= ") {
		t.Fatalf("029 R7: the bounded distribution must keep windows alive at or after from (NULL = never expires), got: %s", stmt)
	}
}

// Total attempts must exclude in-flight rows: an attempt still in
// `processing` has not completed a renewal, so counting it deflates the
// success rate while every retry stays open.
//
// 029 T026/R8: with a period, the same rule applies inside
// created_at >= from AND created_at < to (the attempt belongs to the period
// it was created in), and `status <> 'processing'` must survive the filter.
func TestRenewalStatsExcludesProcessing(t *testing.T) {
	stmt := statsStatement(t, func(dbConn *gorm.DB) error {
		_, err := NewSubscriptionRenewalAttemptRepository(dbConn).GetRenewalStats(context.Background(), time.Time{}, time.Time{})
		return err
	})

	if !strings.Contains(stmt, "status <> 'processing'") {
		t.Fatalf("renewal stats must exclude in-flight attempts, got: %s", stmt)
	}
}

// 029 T026/R8: the all-time query above has no created_at filter at all; the
// bounded one must add the half-open range without dropping the processing
// exclusion.
func TestRenewalStatsUsesTheCreationPeriod(t *testing.T) {
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	stmt := statsStatement(t, func(dbConn *gorm.DB) error {
		_, err := NewSubscriptionRenewalAttemptRepository(dbConn).GetRenewalStats(context.Background(), from, to)
		return err
	})

	if !strings.Contains(stmt, "created_at >= ") || !strings.Contains(stmt, "created_at < ") {
		t.Fatalf("029 R8: renewal stats must filter attempts created in [from, to), got: %s", stmt)
	}
	if !strings.Contains(stmt, "status <> 'processing'") {
		t.Fatalf("029 R8: the period filter must keep excluding in-flight attempts, got: %s", stmt)
	}
}
