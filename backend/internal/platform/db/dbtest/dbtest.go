// Package dbtest provides the shared Postgres test fixture: a throwaway
// container from the same image as production, migrated with the exact
// embedded runner that ships to prod (docs/stack/backend-go.md:
// "Postgres y Redis reales, migraciones incluidas").
package dbtest

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/tiagofur/imalive/backend/internal/platform/db"
)

// Pool bundles the migrated pool with the helpers the store tests share.
type Pool struct {
	t    *testing.T
	Pool *pgxpool.Pool
	dsn  string
}

// New runs postgres:16, applies all embedded migrations and returns the
// migrated pool plus cleanup registered on the test. Tests skip (with a
// clear message) when no container runtime is available: CI is the gate.
func New(t *testing.T) *Pool {
	t.Helper()
	ctx := context.Background()
	pgc, err := postgres.Run(ctx, "postgres:16",
		postgres.WithDatabase("imalive"),
		postgres.WithUsername("imalive"),
		postgres.WithPassword("imalive"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Skipf("dbtest: container runtime unavailable, skipping integration test: %v", err)
	}
	t.Cleanup(func() { _ = pgc.Terminate(ctx) })

	dsn, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("dbtest: connection string: %v", err)
	}
	if err := db.Up(dsn); err != nil {
		t.Fatalf("dbtest: migrate up: %v", err)
	}
	pool, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("dbtest: open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return &Pool{t: t, Pool: pool, dsn: dsn}
}

// URL exposes the container DSN for tests that exercise the migration
// runner itself rather than the pool.
func (p *Pool) URL() string { return p.dsn }

// SeedUser inserts a user directly (the users store ships with #39 auth;
// these tests only need the FK target). UUIDs are app-generated (v7):
// Postgres 16 has no native uuidv7 (issue #38 plan).
func (p *Pool) SeedUser(phone string) uuid.UUID {
	p.t.Helper()
	const q = `INSERT INTO users (id, phone, display_name, timezone)
	           VALUES ($1, $2, 'Ana', 'America/Mexico_City')`
	id, err := uuid.NewV7()
	if err != nil {
		p.t.Fatalf("dbtest: uuid v7: %v", err)
	}
	if _, err := p.Pool.Exec(context.Background(), q, id, phone); err != nil {
		p.t.Fatalf("dbtest: seed user: %v", err)
	}
	return id
}
