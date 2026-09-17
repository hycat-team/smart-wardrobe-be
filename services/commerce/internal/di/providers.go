package di

import (
	"context"
	"io/fs"

	sharedbootstrap "smart-wardrobe-be/shared/bootstrap"
)

// provideMigrationsFS exposes the Commerce-owned migration tree. Commerce has
// no migrations yet, so it returns nil and App.Run skips auto-migration;
// future Commerce migrations add an embed.go and return it here (FR-035).
func provideMigrationsFS() fs.FS {
	return nil
}

type noopValidator struct{}

func (noopValidator) Validate(context.Context) error { return nil }

// provideValidator supplies a no-op startup validator: Commerce has no
// catalog checks of its own. App.Run still nil-guards it.
func provideValidator() sharedbootstrap.StartupValidator {
	return noopValidator{}
}
