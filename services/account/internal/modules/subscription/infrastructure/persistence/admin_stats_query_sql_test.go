package persistence

import (
	"context"
	"io"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

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
func TestAdminStatsRevenueSumsSuccessOnly(t *testing.T) {
	snapshot := time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)
	stmt := statsStatement(t, func(dbConn *gorm.DB) error {
		_, err := NewDepositTransactionRepository(dbConn).GetAdminSubscriptionStats(context.Background(), snapshot)
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
}

// The distribution must count active subscriptions only: the old GROUP BY
// counted expired rows, so a lapsed premium user kept showing up in
// "premium-monthly".
func TestDistributionCountsActiveOnly(t *testing.T) {
	stmt := statsStatement(t, func(dbConn *gorm.DB) error {
		_, err := NewUserSubscriptionRepository(dbConn).GetDistributionByPlanCode(context.Background())
		return err
	})

	if !strings.Contains(stmt, "expires_at IS NULL OR expires_at >") {
		t.Fatalf("distribution must filter on active subscriptions, got: %s", stmt)
	}
}

// Total attempts must exclude in-flight rows: an attempt still in
// `processing` has not completed a renewal, so counting it deflates the
// success rate while every retry stays open.
func TestRenewalStatsExcludesProcessing(t *testing.T) {
	stmt := statsStatement(t, func(dbConn *gorm.DB) error {
		_, err := NewSubscriptionRenewalAttemptRepository(dbConn).GetRenewalStats(context.Background())
		return err
	})

	if !strings.Contains(stmt, "status <> 'processing'") {
		t.Fatalf("renewal stats must exclude in-flight attempts, got: %s", stmt)
	}
}
