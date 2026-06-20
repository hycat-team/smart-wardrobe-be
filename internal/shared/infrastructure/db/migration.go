package db

import (
	"context"
	"fmt"
	"smart-wardrobe-be/migrations"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"gorm.io/gorm"
)

// RunMigrations executes embedded Goose SQL migrations using a Postgres session-level advisory lock
// to prevent concurrent migration execution in multi-replica deployments.
func RunMigrations(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from gorm: %w", err)
	}

	// 1. Create a Postgres session-level advisory locker
	sessionLocker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("failed to create postgres session locker: %w", err)
	}

	// 2. Set the base filesystem to our embedded migrations
	goose.SetBaseFS(migrations.EmbedFS)

	// 3. Create a Goose provider with session locking enabled
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		sqlDB,
		migrations.EmbedFS,
		goose.WithSessionLocker(sessionLocker),
	)
	if err != nil {
		return fmt.Errorf("failed to create goose provider: %w", err)
	}

	// 4. Run up migrations
	fmt.Println("Running database migrations...")
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("database migrations failed: %w", err)
	}

	for _, res := range results {
		fmt.Printf("Migrated: %s (%s)\n", res.Source.Path, res.Duration)
	}

	fmt.Println("Database migrations applied successfully.")
	return nil
}
