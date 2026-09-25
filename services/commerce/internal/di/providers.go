package di

import (
	"context"
	"io/fs"

	sharedbootstrap "smart-wardrobe-be/shared/bootstrap"

	commercemigrations "smart-wardrobe-be/services/commerce/database/migrations"
)

// provideMigrationsFS exposes the Commerce-owned migration tree as an fs.FS so
// the runtime can run migrations on start when enabled (FR-035).
func provideMigrationsFS() fs.FS {
	return commercemigrations.EmbedFS
}

type noopValidator struct{}

func (noopValidator) Validate(context.Context) error { return nil }

// provideValidator supplies a no-op startup validator: Commerce has no
// catalog checks of its own. App.Run still nil-guards it.
func provideValidator() sharedbootstrap.StartupValidator {
	return noopValidator{}
}
