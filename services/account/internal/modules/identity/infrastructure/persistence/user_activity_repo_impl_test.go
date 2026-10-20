package persistence_test

import (
	"context"
	"io"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/useractivitytype"
	identityrepositories "smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	identitypersistence "smart-wardrobe-be/services/account/internal/modules/identity/infrastructure/persistence"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// 029 T007: the admin activity list (FR-001, FR-002, FR-005).
//
// Two halves, because either alone leaves the requirement unproven:
//
//   - SQL pinning, with the same unreachable logger-capture technique as
//     admin_stats_query_sql_test.go: GORM calls Trace on every executed
//     statement (log level does not gate the call), so a sqlite-backed run
//     captures BOTH statements the method builds. The custom ParamsFilter
//     returns no vars, which is why the bound filters show up as `?`
//     placeholders below — the assertions are on the statement shape.
//   - Behaviour on sqlite, where the same SQL runs for real: identity of a
//     soft-deleted account must survive the LEFT JOIN, the filters must narrow
//     the page, and the newest event must come first.
//
// WHAT THE PINNING STILL DOES NOT PROVE: no PostgreSQL server means no proof
// that the server accepts the timestamptz comparisons. It proves the repository
// BUILDS the query the contract describes.

// activitySQLCapture records every statement GORM hands to the driver, success
// or failure, while delegating the actual logging to a silent base.
type activitySQLCapture struct {
	gormlogger.Interface
	mu         sync.Mutex
	statements []string
}

func newActivitySQLCapture() *activitySQLCapture {
	base := gormlogger.New(log.New(io.Discard, "", 0), gormlogger.Config{LogLevel: gormlogger.Silent})
	return &activitySQLCapture{Interface: base}
}

func (l *activitySQLCapture) ParamsFilter(_ context.Context, sql string, _ ...any) (string, []any) {
	return sql, nil
}

func (l *activitySQLCapture) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	statement, _ := fc()
	l.mu.Lock()
	l.statements = append(l.statements, statement)
	l.mu.Unlock()
	l.Interface.Trace(ctx, begin, fc, err)
}

func (l *activitySQLCapture) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.statements...)
}

// reset drops what setup wrote (the two CREATE TABLEs), so the pinning test
// asserts only on the statements the repository built.
func (l *activitySQLCapture) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.statements = nil
}

var _ gormlogger.Interface = (*activitySQLCapture)(nil)

// openActivitySQLite builds the two tables the query touches and returns the
// connection. The DSN is named so the shared in-memory database cannot collide
// with the DAU test in this package.
func openActivitySQLite(t *testing.T, capture gormlogger.Interface) *gorm.DB {
	t.Helper()

	cfg := &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               gormlogger.Default.LogMode(gormlogger.Silent),
	}
	if capture != nil {
		cfg.Logger = capture
	}
	dbConn, err := gorm.Open(sqlite.Open("file:list-admin-activity?mode=memory&cache=shared&_busy_timeout=5000"), cfg)
	require.NoError(t, err)

	require.NoError(t, dbConn.Exec(`DROP TABLE IF EXISTS user_activity_events`).Error)
	require.NoError(t, dbConn.Exec(`DROP TABLE IF EXISTS users`).Error)
	require.NoError(t, dbConn.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			email TEXT NOT NULL,
			first_name TEXT,
			last_name TEXT,
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
			created_at DATETIME NOT NULL,
			device_name TEXT,
			device_ip TEXT,
			user_agent TEXT
		)`).Error)
	t.Cleanup(func() {
		_ = dbConn.Exec(`DROP TABLE IF EXISTS user_activity_events`).Error
		_ = dbConn.Exec(`DROP TABLE IF EXISTS users`).Error
	})
	return dbConn
}

func insertActivityUser(t *testing.T, dbConn *gorm.DB, name string, deleted bool, firstName, lastName *string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	err := dbConn.Exec(
		`INSERT INTO users (id, username, email, first_name, last_name, created_at, updated_at, is_deleted)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id.String(), name, name+"@example.test", firstName, lastName, time.Now(), time.Now(), deleted,
	).Error
	require.NoError(t, err)
	return id
}

func insertActivityEvent(t *testing.T, dbConn *gorm.DB, userID uuid.UUID, activity useractivitytype.UserActivityType, at time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	err := dbConn.Exec(
		`INSERT INTO user_activity_events (id, user_id, activity_type, occurred_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		id.String(), userID.String(), string(activity), at, at,
	).Error
	require.NoError(t, err)
	return id
}

func strPtr(v string) *string { return &v }

// TestListAdminActivityEventsSQLPinsJoinOrderAndMatchingCount is the SQL-pinning
// half: the SELECT must read identity through a LEFT JOIN, order the page
// `occurred_at DESC, id DESC`, and the COUNT must carry the SAME filter set as
// the SELECT (a count that ignores a filter drifts from the page immediately,
// and pagination then lies about how many events exist).
func TestListAdminActivityEventsSQLPinsJoinOrderAndMatchingCount(t *testing.T) {
	capture := newActivitySQLCapture()
	dbConn := openActivitySQLite(t, capture)
	repo := identitypersistence.NewUserActivityRepository(dbConn)
	capture.reset() // drop the CREATE TABLEs

	userID := uuid.New()
	activity := useractivitytype.Login
	from := time.Date(2026, 10, 15, 0, 0, 0, 0, timeutils.VnLoc())
	to := time.Date(2026, 10, 16, 0, 0, 0, 0, timeutils.VnLoc())

	_, _, err := repo.ListAdmin(context.Background(), identityrepositories.AdminActivityEventFilter{
		UserID:       &userID,
		ActivityType: &activity,
		From:         from,
		To:           to,
	}, 2, 50)
	require.NoError(t, err)

	captured := capture.snapshot()
	require.Len(t, captured, 2, "ListAdmin must build exactly a COUNT and a SELECT: %v", captured)

	var countStmt, selectStmt string
	for _, stmt := range captured {
		if strings.Contains(stmt, "count(*)") {
			countStmt = stmt
		} else {
			selectStmt = stmt
		}
	}
	require.NotEmpty(t, countStmt, "one statement must be the COUNT: %v", captured)
	require.NotEmpty(t, selectStmt, "one statement must be the SELECT: %v", captured)

	// The table names stay unqualified: identity repositories read through the
	// module search_path, not through `identity.` prefixes.
	for name, stmt := range map[string]string{"count": countStmt, "select": selectStmt} {
		if !strings.Contains(stmt, "user_activity_events") {
			t.Errorf("%s must read user_activity_events, got: %s", name, stmt)
		}
		if strings.Contains(stmt, "identity.") {
			t.Errorf("%s must not qualify the table with the schema name, got: %s", name, stmt)
		}
		for _, fragment := range []string{
			"e.user_id = ",
			"e.activity_type = ",
			"e.occurred_at >= ",
			"e.occurred_at < ",
		} {
			if !strings.Contains(stmt, fragment) {
				t.Errorf("%s must carry the filter %q, got: %s", name, fragment, stmt)
			}
		}
	}
	// The two statements walk different logger paths (Count keeps `?`
	// placeholders, Scan records the interpolated statement), so the SAME
	// predicates are pinned through the value path here: the bound user and
	// type must have reached the page query.
	if !strings.Contains(selectStmt, userID.String()) {
		t.Errorf("the SELECT must bind the filtered user, got: %s", selectStmt)
	}
	if !strings.Contains(selectStmt, string(activity)) {
		t.Errorf("the SELECT must bind the filtered activity type, got: %s", selectStmt)
	}

	if strings.Contains(countStmt, "JOIN") {
		t.Errorf("the COUNT needs no join and must not carry one, got: %s", countStmt)
	}
	if strings.Contains(countStmt, "LIMIT") {
		t.Errorf("the COUNT must not be paginated, got: %s", countStmt)
	}

	if !strings.Contains(selectStmt, "LEFT JOIN users u ON u.id = e.user_id") {
		t.Errorf("the SELECT must keep identity through a LEFT JOIN on users, got: %s", selectStmt)
	}
	for _, column := range []string{
		"e.id", "e.user_id", "e.activity_type", "e.occurred_at", "e.created_at",
		"e.device_name", "e.device_ip", "e.user_agent",
		"u.email", "u.first_name", "u.last_name",
	} {
		if !strings.Contains(selectStmt, column) {
			t.Errorf("the SELECT must read %s, got: %s", column, selectStmt)
		}
	}
	if !strings.Contains(selectStmt, "ORDER BY e.occurred_at") || !strings.Contains(selectStmt, "e.id DESC") {
		t.Errorf("the page must be ordered occurred_at DESC, id DESC, got: %s", selectStmt)
	}
	if !strings.Contains(selectStmt, "LIMIT 50") || !strings.Contains(selectStmt, "OFFSET 50") {
		t.Errorf("page 2 limit 50 must render LIMIT 50 OFFSET 50, got: %s", selectStmt)
	}
}

// TestListAdminActivityEventsBehaviour pins what the SQL MEANS on a database
// that runs it: newest first, filters honoured, and a soft-deleted account's
// identity still visible (data-model §1 — the event must not disappear because
// the account was deleted; the row survives soft deletion and the LEFT JOIN
// reads it).
func TestListAdminActivityEventsBehaviour(t *testing.T) {
	dbConn := openActivitySQLite(t, nil)
	repo := identitypersistence.NewUserActivityRepository(dbConn)

	active := insertActivityUser(t, dbConn, "activity-active", false, strPtr("An"), strPtr("Nguyễn"))
	softDeleted := insertActivityUser(t, dbConn, "activity-deleted", true, strPtr("Dũng"), nil)

	// Outside the day window below.
	insertActivityEvent(t, dbConn, active, useractivitytype.Login, time.Date(2026, 10, 14, 20, 0, 0, 0, timeutils.VnLoc()))
	loginEventID := insertActivityEvent(t, dbConn, active, useractivitytype.Login, time.Date(2026, 10, 15, 8, 0, 0, 0, timeutils.VnLoc()))
	refreshEventID := insertActivityEvent(t, dbConn, active, useractivitytype.Refresh, time.Date(2026, 10, 15, 9, 0, 0, 0, timeutils.VnLoc()))
	deletedLoginEventID := insertActivityEvent(t, dbConn, softDeleted, useractivitytype.Login, time.Date(2026, 10, 15, 7, 30, 0, 0, timeutils.VnLoc()))

	dayFrom := time.Date(2026, 10, 15, 0, 0, 0, 0, timeutils.VnLoc())
	dayTo := time.Date(2026, 10, 16, 0, 0, 0, 0, timeutils.VnLoc())

	t.Run("no filter returns everything newest first", func(t *testing.T) {
		rows, total, err := repo.ListAdmin(context.Background(), identityrepositories.AdminActivityEventFilter{}, 1, 20)
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		require.Len(t, rows, 4)
		assert.Equal(t, refreshEventID, rows[0].ID, "the newest event must come first")
		assert.Equal(t, useractivitytype.Refresh, useractivitytype.UserActivityType(rows[0].ActivityType))
		assert.Equal(t, "activity-active@example.test", rows[0].Email)
	})

	t.Run("the day window and activity type narrow the page", func(t *testing.T) {
		activity := useractivitytype.Login
		rows, total, err := repo.ListAdmin(context.Background(), identityrepositories.AdminActivityEventFilter{
			ActivityType: &activity,
			From:         dayFrom,
			To:           dayTo,
		}, 1, 20)
		require.NoError(t, err)
		assert.Equal(t, int64(2), total, "one login inside the window per account; the refresh and the previous day stay out")
		require.Len(t, rows, 2)
		assert.Equal(t, loginEventID, rows[0].ID)
		assert.Equal(t, deletedLoginEventID, rows[1].ID)
	})

	t.Run("a soft-deleted account keeps its identity", func(t *testing.T) {
		activity := useractivitytype.Login
		rows, _, err := repo.ListAdmin(context.Background(), identityrepositories.AdminActivityEventFilter{
			UserID:       &softDeleted,
			ActivityType: &activity,
			From:         dayFrom,
			To:           dayTo,
		}, 1, 20)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		row := rows[0]
		assert.Equal(t, deletedLoginEventID, row.ID)
		assert.Equal(t, softDeleted, row.UserID)
		assert.Equal(t, "activity-deleted@example.test", row.Email)
		require.NotNil(t, row.FirstName)
		assert.Equal(t, "Dũng", *row.FirstName)
		assert.Nil(t, row.LastName, "a NULL last_name must reach the read model as nil")
	})

	t.Run("the user filter alone selects that account's events", func(t *testing.T) {
		rows, total, err := repo.ListAdmin(context.Background(), identityrepositories.AdminActivityEventFilter{
			UserID: &active,
		}, 1, 20)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		require.Len(t, rows, 3)
		for _, row := range rows {
			assert.Equal(t, active, row.UserID)
		}
	})

	t.Run("one open bound leaves the range open on the other side", func(t *testing.T) {
		rows, total, err := repo.ListAdmin(context.Background(), identityrepositories.AdminActivityEventFilter{
			From: dayFrom,
		}, 1, 20)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total, "only the previous-day event stays out")
		require.Len(t, rows, 3)

		rows, total, err = repo.ListAdmin(context.Background(), identityrepositories.AdminActivityEventFilter{
			To: dayFrom,
		}, 1, 20)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total, "the upper bound excludes the whole day it starts")
		require.Len(t, rows, 1)
	})
}
