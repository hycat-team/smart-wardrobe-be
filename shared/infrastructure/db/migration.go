package db

import (
	"context"
	"fmt"
	"io/fs"

	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RunMigrationsIfEnabled runs embedded Goose migrations only when enabled
// (FR-011/FR-035). Disabled profiles must migrate via controlled channels
// (`make migrate-up svc=<name>`) before deploy instead.
//
// migrationsFS is the owning service's embedded SQL tree (e.g. Account holds
// the 24 historical migrations; other services pass their own directory).
// Taking the FS as a parameter keeps shared free of service imports —
// reversing that edge would create a Go module cycle.
func RunMigrationsIfEnabled(ctx context.Context, cfg *config.Config, dbConn *gorm.DB, migrationsFS fs.FS, log logger.Interface) error {
	if !cfg.Database.MigrateOnStart {
		log.Info("Skipping automatic database migrations (database.migrate_on_start=false)")
		return nil
	}
	if migrationsFS == nil {
		log.Info("Skipping automatic database migrations (no migration tree for this service)")
		return nil
	}
	table := cfg.Database.MigrationTable
	if table == "" {
		table = "goose_db_version"
	}
	return RunMigrations(ctx, dbConn, migrationsFS, table, cfg.Database.MigrationAllowOutOfOrder, log)
}

// RunMigrations executes embedded Goose SQL migrations using a Postgres session-level advisory lock
// to prevent concurrent migration execution in multi-replica deployments.
// Each service uses its own version table (FR-035) so ordering never collides
// on the shared database; allowOutOfOrder tolerates late-merged migrations.
func RunMigrations(ctx context.Context, db *gorm.DB, migrationsFS fs.FS, table string, allowOutOfOrder bool, log logger.Interface) error {
	if db == nil {
		return fmt.Errorf("cannot run migrations on a nil database connection")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from gorm: %w", err)
	}

	// 1. Create a Postgres session-level advisory locker
	sessionLocker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("failed to create postgres session locker: %w", err)
	}

	// 2. Set the base filesystem to the owning service's embedded migrations
	goose.SetBaseFS(migrationsFS)

	// 3. Create a Goose provider with session locking enabled
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		sqlDB,
		migrationsFS,
		goose.WithSessionLocker(sessionLocker),
		goose.WithTableName(table),
		goose.WithAllowOutofOrder(allowOutOfOrder),
	)
	if err != nil {
		return fmt.Errorf("failed to create goose provider: %w", err)
	}

	// 4. Run up migrations
	log.Info("Running database migrations...")
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("database migrations failed: %w", err)
	}

	for _, res := range results {
		log.Info("Migration applied", zap.String("migration", res.Source.Path), zap.Duration("duration", res.Duration))
	}

	log.Info("Database migrations applied successfully.")
	return nil
}
