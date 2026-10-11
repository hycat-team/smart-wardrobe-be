package usecase_test

import (
	"context"
	"testing"

	"smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"
	identityuc "smart-wardrobe-be/services/account/internal/modules/identity/application/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/authprovider"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/identity/infrastructure/persistence"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func newActivityFixtureDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbConn, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_busy_timeout=5000"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, dbConn.Exec(`DROP TABLE IF EXISTS refresh_tokens`).Error)
	require.NoError(t, dbConn.Exec(`DROP TABLE IF EXISTS user_activity_events`).Error)
	require.NoError(t, dbConn.Exec(`
		CREATE TABLE refresh_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			is_revoked BOOLEAN NOT NULL DEFAULT 0,
			device_name TEXT,
			device_ip TEXT,
			user_agent TEXT,
			last_active_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`).Error)
	require.NoError(t, dbConn.Exec(`
		CREATE TABLE user_activity_events (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			activity_type TEXT NOT NULL,
			occurred_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`).Error)
	t.Cleanup(func() {
		_ = dbConn.Exec(`DROP TABLE IF EXISTS user_activity_events`).Error
		_ = dbConn.Exec(`DROP TABLE IF EXISTS refresh_tokens`).Error
	})
	return dbConn
}

func newSessionTestConfig() *config.Config {
	return &config.Config{Jwt: config.Jwt{
		Secret:                  "test-secret-that-is-long-enough",
		Issuer:                  "closy-test",
		Audience:                "closy-test",
		AccessExpirationMinutes: 15,
		RefreshExpirationDays:   30,
	}}
}

func newActivityTestUser() *entities.User {
	user := &entities.User{
		Username: "activity-user",
		Email:    "activity@example.com",
		Provider: authprovider.Local,
		RoleSlug: roleslug.RoleSlug("user"),
		Status:   userstatus.Active,
	}
	user.ID = uuid.New()
	return user
}

// A login must leave exactly one `login` activity event behind: that row is
// the raw material the admin dashboard DAU is counted from. If issuing a
// session stopped writing it, the DAU would silently decay — and every stats
// test above would stay green.
func TestIssueSessionRecordsLoginActivity(t *testing.T) {
	dbConn := newActivityFixtureDB(t)
	user := newActivityTestUser()

	uc := identityuc.NewSessionUseCase(
		&loginUserRepo{user: user},
		persistence.NewRefreshTokenRepository(dbConn),
		persistence.NewUserActivityRepository(dbConn),
		&exactHasher{password: "correct-horse"},
		&fakeBlacklist{},
		newSessionTestConfig(),
		nil,
	)

	res, err := uc.IssueSession(context.Background(), user, usecase.SessionDevice{})
	require.NoError(t, err)
	require.NotNil(t, res)

	var rows []struct {
		ActivityType string
		UserID       string
	}
	require.NoError(t, dbConn.Raw(
		`SELECT activity_type, user_id FROM user_activity_events`,
	).Scan(&rows).Error)
	require.Len(t, rows, 1, "one login must write exactly one activity event")
	assert.Equal(t, "login", rows[0].ActivityType)
	assert.Equal(t, user.ID.String(), rows[0].UserID)
}
