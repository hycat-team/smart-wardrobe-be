// Package campaigndb serializes the Postgres-backed campaign fixtures that more
// than one package seeds into the same TEST_DATABASE_URL.
//
// WHY THIS EXISTS. `subscription.campaigns` is a registry with a single
// selection rule: FR-040 picks the campaign with the LARGEST starts_at. Every
// fixture package writes its own `verify-026-*` rows there, and `go test
// ./internal/modules/subscription/...` runs those package binaries IN PARALLEL
// against one database. Without a shared lock, the campaign use-case fixtures
// see the persistence package's rows as "foreign" and fail their isolation
// precondition, or - worse - grant from the other suite's campaign and assert
// against the wrong row.
//
// `make test-db-integration` runs with `-p 1`, which hides this; ad-hoc scoped
// runs do not. The lock makes the suites correct regardless of how they are
// invoked, instead of documenting an invocation flag every future reader has to
// remember.
//
// It is deliberately a PostgreSQL SESSION advisory lock on a pinned connection,
// not an in-process mutex: the contention is across `go test` package binaries,
// which share nothing but the database.
package campaigndb

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"
)

// FixturePrefix is the code prefix every campaign fixture in this repository
// uses. Rows outside it (notably the seeded `launch-2026-10`) are never purged.
const FixturePrefix = "verify-026-"

// advisoryLockKey namespaces this lock. `hashtextextended(..., 0)` gives a
// 64-bit key; the string prefix keeps it disjoint from any other advisory lock
// the service may take with a raw integer key.
const advisoryLockKey = "closy.campaign-registry-fixtures"

// lockWait bounds how long a fixture waits for another package's suite. A wait
// longer than this means a suite crashed while holding the session lock and its
// connection was not closed; failing loudly is better than hanging the run.
const lockWait = 60 * time.Second

// LockAndPurge takes the cross-package advisory lock for the duration of the
// test and then removes stale fixture rows left by earlier failed runs.
//
// Call it in the fixture opener AFTER migrations and BEFORE seeding or
// asserting registry isolation. It registers its own `t.Cleanup`, and because
// cleanups run last-in-first-out, that release runs after the caller's own
// prefix cleanup. Purge is safe under the lock: every cooperating suite holds
// the same lock while it reads or writes the registry.
func LockAndPurge(tb testing.TB, db *gorm.DB) {
	tb.Helper()

	release, err := acquire(context.Background(), tb, db)
	if err != nil {
		tb.Fatalf("acquire campaign fixture advisory lock: %v", err)
	}
	tb.Cleanup(release)

	purged, err := purge(context.Background(), db)
	if err != nil {
		tb.Fatalf("purge stale campaign fixtures: %v", err)
	}
	if purged > 0 {
		tb.Logf("purged %d stale campaign fixture row(s) with prefix %q", purged, FixturePrefix)
	}
}

// acquire pins one connection and takes the session advisory lock on it.
//
// The pin matters: `pg_advisory_lock` belongs to the SESSION, so unlocking from
// a different pooled connection would not release it and would leave every
// later suite waiting for lockWait.
func acquire(ctx context.Context, tb testing.TB, db *gorm.DB) (func(), error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("pin a connection: %w", err)
	}

	if _, err := conn.ExecContext(ctx, fmt.Sprintf("SET lock_timeout = '%dms'", lockWait.Milliseconds())); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("set lock_timeout: %w", err)
	}
	if _, err := conn.ExecContext(ctx,
		"SELECT pg_advisory_lock(hashtextextended($1, 0))", advisoryLockKey); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("take advisory lock %q: %w", advisoryLockKey, err)
	}

	return func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := conn.ExecContext(unlockCtx,
			"SELECT pg_advisory_unlock(hashtextextended($1, 0))", advisoryLockKey); err != nil {
			// The connection close below releases the session lock even if the
			// explicit unlock fails, so this is a diagnostic rather than a leak.
			tb.Logf("release campaign fixture advisory lock: %v", err)
		}
		if err := conn.Close(); err != nil {
			tb.Logf("close the campaign fixture lock connection: %v", err)
		}
	}, nil
}

// purge removes every stale fixture row. Claims are named by campaign code and
// carry no foreign key, so they are deleted first; campaigns follow.
func purge(ctx context.Context, db *gorm.DB) (int64, error) {
	if err := db.WithContext(ctx).
		Exec("DELETE FROM subscription.campaign_claims WHERE campaign_code LIKE ?", FixturePrefix+"%").
		Error; err != nil {
		return 0, fmt.Errorf("delete stale campaign claims: %w", err)
	}
	result := db.WithContext(ctx).
		Exec("DELETE FROM subscription.campaigns WHERE code LIKE ?", FixturePrefix+"%")
	if result.Error != nil {
		return 0, fmt.Errorf("delete stale campaigns: %w", result.Error)
	}
	return result.RowsAffected, nil
}
