package di

import (
	"io/fs"

	accountmigrations "smart-wardrobe-be/services/account/database/migrations"
)

// provideMigrationsFS exposes the Account-owned migration tree as an fs.FS so
// the runtime can run migrations on start when enabled (FR-035).
func provideMigrationsFS() fs.FS {
	return accountmigrations.EmbedFS
}
