package migrations

import "embed"

// EmbedFS embeds all Commerce-owned SQL migrations so they are compiled into the
// service binary and can run on start when enabled (constitution FR-035).
//
//go:embed *.sql
var EmbedFS embed.FS
