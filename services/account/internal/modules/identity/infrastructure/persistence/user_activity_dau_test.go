package persistence_test

import (
	"context"
	"testing"
	"time"

	identitypersistence "smart-wardrobe-be/services/account/internal/modules/identity/infrastructure/persistence"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DAU is distinct users with a real activity event in the Vietnam calendar
// day. The three traps this pins:
//
//  1. VN-day boundary: an event at 23:30 UTC belongs to the NEXT VN day
//     (06:30 +07). A UTC-midnight window would have put it on the wrong day.
//  2. Distinctness: two events from one user count once.
//  3. Deleted accounts: activity from a soft-deleted account is not DAU.
func TestAdminUserStatsDAUUsesVietnamDayActivity(t *testing.T) {
	dbConn, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_busy_timeout=5000"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, dbConn.Exec(`DROP TABLE IF EXISTS users`).Error)
	require.NoError(t, dbConn.Exec(`DROP TABLE IF EXISTS user_activity_events`).Error)
	require.NoError(t, dbConn.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			email TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME,
			is_deleted BOOLEAN NOT NULL DEFAULT 0
		)`).Error)
	require.NoError(t, dbConn.Exec(`
		CREATE TABLE user_activity_events (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			activity_type TEXT NOT NULL,
			occurred_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL
		)`).Error)
	t.Cleanup(func() {
		_ = dbConn.Exec(`DROP TABLE IF EXISTS user_activity_events`).Error
		_ = dbConn.Exec(`DROP TABLE IF EXISTS users`).Error
	})

	vn := timeutils.VnLoc()
	mkUser := func(name string, created time.Time, deleted bool) uuid.UUID {
		id := uuid.New()
		require.NoError(t, dbConn.Exec(
			`INSERT INTO users (id, username, email, created_at, updated_at, is_deleted) VALUES (?, ?, ?, ?, ?, ?)`,
			id.String(), name, name+"@example.test", created, created, deleted,
		).Error)
		return id
	}
	mkEvent := func(userID uuid.UUID, at time.Time) {
		require.NoError(t, dbConn.Exec(
			`INSERT INTO user_activity_events (id, user_id, activity_type, occurred_at, created_at) VALUES (?, ?, 'login', ?, ?)`,
			uuid.New().String(), userID.String(), at, at,
		).Error)
	}

	// u1 registered on the 10th (VN), u2 on the 11th (VN) → one new user.
	u1 := mkUser("dau-one", time.Date(2026, 10, 10, 9, 0, 0, 0, vn), false)
	u2 := mkUser("dau-two", time.Date(2026, 10, 11, 8, 30, 0, 0, vn), false)
	u3 := mkUser("dau-deleted", time.Date(2026, 10, 10, 9, 0, 0, 0, vn), true)

	// u1 active on the 10th (VN) — must NOT count for the 11th.
	mkEvent(u1, time.Date(2026, 10, 10, 20, 0, 0, 0, vn))
	// u1 active at 23:30 UTC = 06:30 +07 on the 11th — counts, and proves the
	// window is VN, not UTC.
	mkEvent(u1, time.Date(2026, 10, 10, 23, 30, 0, 0, time.UTC))
	// u1 active again on the 11th — still one user.
	mkEvent(u1, time.Date(2026, 10, 11, 9, 0, 0, 0, vn))
	// u2 active on the 11th.
	mkEvent(u2, time.Date(2026, 10, 11, 8, 45, 0, 0, vn))
	// u3 (deleted) active on the 11th — must not inflate DAU.
	mkEvent(u3, time.Date(2026, 10, 11, 10, 0, 0, 0, vn))

	repo := identitypersistence.NewUserRepository(dbConn, identitypersistence.NewUserActivityRepository(dbConn))

	snapshot := time.Date(2026, 10, 11, 9, 0, 0, 0, vn)
	stats, err := repo.GetAdminUserStats(context.Background(), snapshot)
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.Equal(t, 2, stats.TotalUsers, "deleted accounts are not users")
	assert.Equal(t, 1, stats.NewUsers, "only u2 registered on the 11th (VN)")
	assert.Equal(t, 2, stats.ActiveUsersDau, "u1 and u2; u1 once despite two events; u3 deleted")
}
