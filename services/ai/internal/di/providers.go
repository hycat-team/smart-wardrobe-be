package di

import (
	"context"
	"io/fs"

	sharedbootstrap "smart-wardrobe-be/shared/bootstrap"
)

// provideMigrationsFS exposes the AI-owned migration tree. AI has no
// migrations yet, so it returns nil and App.Run skips auto-migration;
// future AI migrations add an embed.go and return it here (FR-035).
func provideMigrationsFS() fs.FS {
	return nil
}

type noopValidator struct{}

func (noopValidator) Validate(context.Context) error { return nil }

// provideValidator supplies a no-op startup validator: AI has no catalog
// checks of its own. App.Run still nil-guards it.
func provideValidator() sharedbootstrap.StartupValidator {
	return noopValidator{}
}
