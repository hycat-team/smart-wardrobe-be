package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Job advisory-lock IDs (FR-045/SC-019). Advisory locks are cluster-wide
// int64, so these are namespaced far from Goose's migration lock and from
// each other. One ID per non-idempotent scheduled job.
const (
	// SubscriptionRenewalLockID guards subscription renewal + expiry runs.
	SubscriptionRenewalLockID int64 = 8_200_101
	// LoyaltyExpiryLockID guards loyalty point expiry runs.
	LoyaltyExpiryLockID int64 = 8_200_102
)

// JobLocker serializes non-idempotent scheduled jobs across instances with a
// Postgres session-level advisory lock. It holds its own small direct
// connection pool (never the transaction pooler, FR-046) and pins one session
// per held lock.
type JobLocker struct {
	db *sql.DB
}

// NewPostgresJobLocker opens the locker's dedicated direct connection. It
// returns (nil, nil) when locking is unreachable so boot never fails; callers
// treat a nil locker as "locking unavailable" and run unguarded.
func NewPostgresJobLocker(cfg *config.Config, l logger.Interface) (*JobLocker, error) {
	dsn := DirectDSN(cfg)

	opened, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open job locker database: %w", err)
	}
	sqlDB, err := opened.DB()
	if err != nil {
		return nil, fmt.Errorf("resolve job locker database: %w", err)
	}
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(2)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping job locker database: %w", err)
	}

	l.Info("postgres job locker ready (direct connection, advisory locks)")
	return &JobLocker{db: sqlDB}, nil
}

// TryLock attempts a non-blocking session advisory lock. When acquired, the
// caller MUST call release (usually deferred) to free the pinned session.
// A false acquired with nil error means another instance holds the lock.
func (l *JobLocker) TryLock(ctx context.Context, lockID int64) (release func(), acquired bool, err error) {
	if l == nil || l.db == nil {
		return func() {}, false, nil
	}

	conn, err := l.db.Conn(ctx)
	if err != nil {
		return func() {}, false, err
	}

	var ok bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lockID).Scan(&ok); err != nil {
		_ = conn.Close()
		return func() {}, false, err
	}
	if !ok {
		_ = conn.Close()
		return func() {}, false, nil
	}

	return func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", lockID)
		_ = conn.Close()
	}, true, nil
}

// Close releases the locker's connection pool.
func (l *JobLocker) Close() error {
	if l == nil || l.db == nil {
		return nil
	}
	return l.db.Close()
}
