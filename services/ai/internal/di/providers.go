package di

import (
	"context"
	"io/fs"

	aimigrations "smart-wardrobe-be/services/ai/database/migrations"
	sharedbootstrap "smart-wardrobe-be/shared/bootstrap"
)

// provideMigrationsFS exposes the AI-owned migration tree as an fs.FS so
// the runtime can run migrations on start when enabled (FR-035).
func provideMigrationsFS() fs.FS {
	return aimigrations.EmbedFS
}

type noopValidator struct{}

func (noopValidator) Validate(context.Context) error { return nil }

// provideValidator supplies a no-op startup validator: AI has no catalog
// checks of its own. App.Run still nil-guards it.
func provideValidator() sharedbootstrap.StartupValidator {
	return noopValidator{}
}
