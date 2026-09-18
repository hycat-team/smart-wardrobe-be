package migrations

import "embed"

// EmbedFS embeds all SQL migrations so they are compiled directly into the app binary.
//
//go:embed *.sql
var EmbedFS embed.FS
