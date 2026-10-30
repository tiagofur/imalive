// Package db centralizes PostgreSQL access for the backend: the connection
// pool and the embedded migration runner shared by cmd/migrate, CI and the
// integration tests (one migration path, zero external binaries).
package db

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registers the pgx5:// driver used by the runner
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/tiagofur/imalive/backend/migrations"
)

// Up applies all pending migrations. DownOne rolls back a single step and
// Version reports the current schema version (dirty migrations surface as
// errors). The runner never logs data, only versions and outcomes. The
// migrations are embedded in the migrations package: one shared source for
// cmd/migrate, CI and the integration tests (docs/stack/backend-go.md).
func Up(databaseURL string) error {
	m, err := newMigrator(databaseURL)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		_ = closeMigrate(m)
		return fmt.Errorf("db: migrate up: %w", err)
	}
	return closeMigrate(m)
}

// DownOne rolls back exactly one migration step (dev/CI only: production is
// forward-only per the policy in every migration header).
func DownOne(databaseURL string) error {
	m, err := newMigrator(databaseURL)
	if err != nil {
		return err
	}
	if err := m.Steps(-1); err != nil {
		_ = closeMigrate(m)
		return fmt.Errorf("db: migrate down: %w", err)
	}
	return closeMigrate(m)
}

// Version reports the current schema version and whether it is dirty.
func Version(databaseURL string) (uint, bool, error) {
	m, err := newMigrator(databaseURL)
	if err != nil {
		return 0, false, err
	}
	v, dirty, err := m.Version()
	if err != nil {
		_ = closeMigrate(m)
		return 0, false, fmt.Errorf("db: migrate version: %w", err)
	}
	return v, dirty, closeMigrate(m)
}

// closeMigrate discards the source driver Close returns alongside the error.
func closeMigrate(m *migrate.Migrate) error {
	_, err := m.Close()
	return err
}

func newMigrator(databaseURL string) (*migrate.Migrate, error) {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("db: migration source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, "pgx5://"+trimScheme(databaseURL))
	if err != nil {
		return nil, fmt.Errorf("db: migrate init: %w", err)
	}
	return m, nil
}

// trimScheme rewrites postgres[ql]:// to the scheme golang-migrate's pgx/v5
// driver expects (pgx5://), keeping the rest of the DSN intact.
func trimScheme(databaseURL string) string {
	for _, prefix := range []string{"postgres://", "postgresql://"} {
		if len(databaseURL) >= len(prefix) && databaseURL[:len(prefix)] == prefix {
			return databaseURL[len(prefix):]
		}
	}
	return databaseURL
}
