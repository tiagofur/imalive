// Package migrations embeds the SQL migration files so the migration runner
// (internal/platform/db) ships them inside the binaries: cmd/migrate, CI and
// the integration tests all share this single source.
package migrations

import "embed"

// FS carries every SQL migration, embedded at build time.
//
//go:embed *.sql
var FS embed.FS
