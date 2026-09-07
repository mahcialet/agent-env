// Package migrations embeds the authoritative ordered SQLite migrations.
package migrations

import "embed"

// Files contains the schema migrations distributed with the executable.
//
//go:embed *.sql
var Files embed.FS
