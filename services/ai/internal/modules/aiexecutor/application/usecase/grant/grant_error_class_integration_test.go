package grant

import (
	"errors"
	"os"
	"testing"
	"time"

	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/aipolicygrantstatus"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// 027 US3 (T016): the classification against a REAL Postgres, through REAL GORM.
//
// This is the one test that closes research.md R-001 / plan.md Risk 1: nobody in
// the repository has ever read a *pgconn.PgError back through GORM, so it is not
// known whether GORM preserves it under errors.As. A hand-built PgError cannot
// answer that question - it would be green even if the integration were broken,
// which is exactly the "false sense of safety" the contract warns about.
//
// Every case below therefore FORCES the real SQLSTATE through GORM:
//   - 23514 by a CHECK constraint installed here (the entity has no `check:` tag,
//     so AutoMigrate does not create the one production carries);
//   - 23505 by inserting a duplicate primary key;
//   - 40P01 by a genuine deadlock between two transactions that take
//     `pg_advisory_xact_lock` in opposite orders.
// ---------------------------------------------------------------------------

// errorClassWindowCheckName is the name THIS test gives its CHECK. It is
// deliberately different from the production name and from the name the Phase 4
// repair test installs (grantWindowCheckName): each test owns and drops its own
// constraint, and sharing a name would make one test's cleanup reach into
// another test's run.
const errorClassWindowCheckName = "ai_policy_grants_error_class_check"

// pgUniqueViolationCode and pgDeadlockCode are the two post-027 codes that must
// STAY on the transient path. They are named here so the assertions read as the
// rule they enforce rather than as magic strings.
const (
	pgUniqueViolationCode = "23505"
	pgDeadlockCode        = "40P01"
)

// installErrorClassWindowCheck adds the window CHECK production has but
// AutoMigrate does not, so a real 23514 can be produced. See D-06 in deferred.md:
// the constraint lives in the Account service's migration history, and the
// entity's missing `check:` tag is why the test must install it itself.
func installErrorClassWindowCheck(t *testing.T, conn *gorm.DB) {
	t.Helper()

	drop := "ALTER TABLE ai_policy_grants DROP CONSTRAINT IF EXISTS " + errorClassWindowCheckName
	add := "ALTER TABLE ai_policy_grants ADD CONSTRAINT " + errorClassWindowCheckName +
		" CHECK (effective_to IS NULL OR effective_to > effective_from)"

	if err := conn.Exec(drop).Error; err != nil {
		t.Fatalf("drop stale error-class window CHECK: %v", err)
	}
	if err := conn.Exec(add).Error; err != nil {
		t.Fatalf("install error-class window CHECK: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Exec(drop).Error; err != nil {
			t.Errorf("drop the error-class window CHECK: %v", err)
		}
	})
}

// seedErrorClassGrant inserts one fully-populated grant row. It exists so the
// violating statement is the ONLY thing that can fail: if any NOT NULL column
// were missing, a 23502 would fire first and the test would prove nothing about
// 23514.
func seedErrorClassGrant(t *testing.T, conn *gorm.DB) entities.AIPolicyGrant {
	t.Helper()

	now := time.Now().UTC().Truncate(time.Microsecond)
	policy := seedGrantPolicy(t, conn, "premium")
	row := entities.AIPolicyGrant{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{ID: uuid.New(), CreatedAt: now},
			UpdatedAt:  now,
		},
		UserID:         uuid.New(),
		PolicyID:       policy.ID,
		PlanID:         uuid.New(),
		PlanCode:       "premium-monthly",
		TierRank:       1,
		PolicySnapshot: entities.JSONDocument(`{}`),
		EffectiveFrom:  now,
		EffectiveTo:    ptrTime(now.Add(time.Hour)),
		Status:         aipolicygrantstatus.Active,
	}
	insertGrantRow(t, conn, &row)
	return row
}

// TestIsCheckViolationThroughGorm is the one that proves errors.As unwraps
// *pgconn.PgError THROUGH GORM (R-001).
//
// The real 23514 is produced by moving effective_to back onto effective_from -
// the exact shape that killed the first production repair run - and the
// assertion is on isCheckViolation, not on a hand-built error.
func TestIsCheckViolationThroughGorm(t *testing.T) {
	conn := openGrantTestDB(t)
	cleanupGrantTables(t, conn)
	installErrorClassWindowCheck(t, conn)

	row := seedErrorClassGrant(t, conn)

	// effective_to == effective_from violates CHECK (effective_to > effective_from).
	err := conn.Model(&entities.AIPolicyGrant{}).
		Where("id = ?", row.ID).
		Update("effective_to", row.EffectiveFrom).Error
	if err == nil {
		t.Fatal("test premise broken: the CHECK constraint did not fire, so no 23514 was produced")
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("R-001 CLOSED NEGATIVE: errors.As could not unwrap *pgconn.PgError through GORM, got %T: %v", err, err)
	}
	if pgErr.SQLState() != pgCheckViolationCode {
		t.Fatalf("expected SQLSTATE %s, got %s (%v)", pgCheckViolationCode, pgErr.SQLState(), err)
	}
	if !isCheckViolation(err) {
		t.Fatalf("R-001: isCheckViolation must return true for a real 23514 through GORM, got false for %v", err)
	}
}

// TestUniqueViolationStaysRetryable is FR-010's negative: a real 23505 must NOT
// be swept into the permanent class. This is the regression a widened class-23
// match would cause.
func TestUniqueViolationStaysRetryable(t *testing.T) {
	conn := openGrantTestDB(t)
	cleanupGrantTables(t, conn)

	first := seedErrorClassGrant(t, conn)

	// The same primary key twice: a real unique_violation from Postgres.
	duplicate := first
	err := conn.Create(&duplicate).Error
	if err == nil {
		t.Fatal("test premise broken: inserting a duplicate key did not violate a unique index")
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("errors.As could not unwrap *pgconn.PgError through GORM, got %T: %v", err, err)
	}
	if pgErr.SQLState() != pgUniqueViolationCode {
		t.Fatalf("expected SQLSTATE %s, got %s (%v)", pgUniqueViolationCode, pgErr.SQLState(), err)
	}
	if isCheckViolation(err) {
		t.Fatalf("FR-010: a real 23505 must stay retryable, but isCheckViolation returned true for %v", err)
	}
}

// TestDeadlockStaysRetryable is the other FR-010 negative, and the hardest one to
// force honestly.
//
// It opens two independent connections and has each transaction take two
// advisory locks in OPPOSITE order. Both acquire their first lock, then block on
// the second - a genuine deadlock that Postgres detects and resolves by aborting
// one transaction with SQLSTATE 40P01. The test first proves the deadlock
// actually happened (otherwise a green bar would rest on nothing), then asserts
// the abort is not classified as a CHECK violation.
func TestDeadlockStaysRetryable(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping grant error-class integration test")
	}

	connA := openGrantTestDB(t)
	connB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open the second connection for the deadlock: %v", err)
	}
	sqlB, err := connB.DB()
	if err != nil {
		t.Fatalf("resolve the second connection pool: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlB.Close(); err != nil {
			t.Errorf("close the second connection pool: %v", err)
		}
	})

	// Distinct, fixed keys. Each transaction takes its own first, then the other's.
	const keyA int64 = 827364511
	const keyB int64 = 827364512

	readyA := make(chan struct{})
	readyB := make(chan struct{})
	results := make(chan error, 2)

	// lockInOrder takes `first`, announces it, waits for the peer's first lock,
	// then requests `second` - which the peer is holding. Deadlock.
	lockInOrder := func(conn *gorm.DB, first, second int64, readyFirst, readySecond chan struct{}) {
		tx := conn.Begin()
		if tx.Error != nil {
			results <- tx.Error
			return
		}
		defer func() { _ = tx.Rollback().Error }()

		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", first).Error; err != nil {
			results <- err
			return
		}
		close(readyFirst)
		<-readySecond
		results <- tx.Exec("SELECT pg_advisory_xact_lock(?)", second).Error
	}

	go lockInOrder(connA, keyA, keyB, readyA, readyB)
	go lockInOrder(connB, keyB, keyA, readyB, readyA)

	var deadlock error
	for i := 0; i < 2; i++ {
		if err := <-results; err != nil {
			deadlock = err
		}
	}
	if deadlock == nil {
		t.Fatal("test premise broken: the two transactions did not deadlock, so this case proves nothing")
	}

	var pgErr *pgconn.PgError
	if !errors.As(deadlock, &pgErr) {
		t.Fatalf("expected a PostgreSQL deadlock error, got %T: %v", deadlock, deadlock)
	}
	if pgErr.SQLState() != pgDeadlockCode {
		t.Fatalf("expected SQLSTATE %s (deadlock_detected), got %s (%v)", pgDeadlockCode, pgErr.SQLState(), deadlock)
	}
	if isCheckViolation(deadlock) {
		t.Fatalf("FR-010: a real deadlock must stay retryable, but isCheckViolation returned true for %v", deadlock)
	}
}
