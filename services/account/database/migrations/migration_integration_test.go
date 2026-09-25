package migrations_test

import (
	"context"
	"os"
	"sync"
	"testing"

	accountmigrations "smart-wardrobe-be/services/account/database/migrations"
	shareddb "smart-wardrobe-be/shared/infrastructure/db"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestIntegrationRunMigrationsConcurrent proves the Postgres session-level
// advisory lock serializes concurrent boot migrations: two replicas starting
// at the same time must both succeed without schema conflicts. It lives in the
// owning service (Account holds the history — FR-035) because shared must not
// import service modules (Go module cycle).
func TestIntegrationRunMigrationsConcurrent(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping migration integration test")
	}
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}

	const replicas = 2
	var wg sync.WaitGroup
	errs := make([]error, replicas)
	for i := 0; i < replicas; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = shareddb.RunMigrations(context.Background(), gormDB, accountmigrations.EmbedFS, "goose_db_version", true, zap.NewNop())
		}(i)
	}
	wg.Wait()

	for idx, err := range errs {
		assert.NoError(t, err, "replica %d migration failed", idx)
	}
}

// TestIntegrationGoogleAuthColumns verifies feature 021 adds the identity
// provider columns and makes password_hash nullable. Skipped without a DB.
func TestIntegrationGoogleAuthColumns(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping migration integration test")
	}
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}

	if err := shareddb.RunMigrations(context.Background(), gormDB, accountmigrations.EmbedFS, "goose_db_version", true, zap.NewNop()); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	var columnCount int64
	if err := gormDB.Raw(`SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = 'identity' AND table_name = 'users'
		  AND column_name IN ('provider', 'provider_id', 'email_verified')`).Scan(&columnCount).Error; err != nil {
		t.Fatalf("query columns: %v", err)
	}
	assert.EqualValues(t, 3, columnCount, "expected provider/provider_id/email_verified columns")

	var nullable string
	if err := gormDB.Raw(`SELECT is_nullable FROM information_schema.columns
		WHERE table_schema = 'identity' AND table_name = 'users' AND column_name = 'password_hash'`).Scan(&nullable).Error; err != nil {
		t.Fatalf("query password_hash nullability: %v", err)
	}
	assert.Equal(t, "YES", nullable, "password_hash must be nullable")
}
