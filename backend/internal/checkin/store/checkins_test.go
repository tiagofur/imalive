package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tiagofur/imalive/backend/internal/checkin/store"
	"github.com/tiagofur/imalive/backend/internal/platform/db/dbtest"
)

func TestCheckinsStore(t *testing.T) {
	ctx := context.Background()
	p := dbtest.New(t)
	q := store.New(p.Pool)
	now := time.Now().UTC()

	t.Run("creates and reads back the daily signal", func(t *testing.T) {
		userID := p.SeedUser("+5215512340001")
		cycleDate := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)

		created, err := q.CreateCheckin(ctx, store.CreateCheckinParams{
			ID:             mustUUID(t),
			UserID:         userID,
			CycleDate:      pgDate(cycleDate),
			CheckedInAt:    pgTz(now),
			IdempotencyKey: strPtr("idem-1"),
		})
		if err != nil {
			t.Fatalf("CreateCheckin: %v", err)
		}

		got, err := q.GetCheckinByDate(ctx, store.GetCheckinByDateParams{UserID: userID, CycleDate: pgDate(cycleDate)})
		if err != nil {
			t.Fatalf("GetCheckinByDate: %v", err)
		}
		if got.ID != created.ID {
			t.Fatalf("GetCheckinByDate = %+v, want the created row", got)
		}
	})

	t.Run("replays the same idempotency key to the same row", func(t *testing.T) {
		userID := p.SeedUser("+5215512340002")
		cycleDate := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
		key := "idem-replay"

		first, err := q.CreateCheckin(ctx, store.CreateCheckinParams{
			ID: mustUUID(t), UserID: userID, CycleDate: pgDate(cycleDate),
			CheckedInAt: pgTz(now), IdempotencyKey: &key,
		})
		if err != nil {
			t.Fatalf("first CreateCheckin: %v", err)
		}

		// A second insert with the same key violates the partial unique index:
		// the handler reads the original row instead (replay), it never writes.
		dup := first
		dup.ID = mustUUID(t)
		_, err = q.CreateCheckin(ctx, store.CreateCheckinParams{
			ID: dup.ID, UserID: userID, CycleDate: pgDate(cycleDate),
			CheckedInAt: pgTz(now), IdempotencyKey: &key,
		})
		assertPgCode(t, err, "23505", "idempotency replay")

		replayed, err := q.GetCheckinByIdempotencyKey(ctx, store.GetCheckinByIdempotencyKeyParams{UserID: userID, IdempotencyKey: &key})
		if err != nil {
			t.Fatalf("GetCheckinByIdempotencyKey: %v", err)
		}
		if replayed.ID != first.ID || replayed.CreatedAt.Time.UTC() != first.CreatedAt.Time.UTC() {
			t.Fatalf("replay = %+v, want the original row %+v", replayed, first)
		}
	})

	t.Run("rejects two check-ins on the same cycle date", func(t *testing.T) {
		userID := p.SeedUser("+5215512340003")
		cycleDate := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

		if _, err := q.CreateCheckin(ctx, store.CreateCheckinParams{
			ID: mustUUID(t), UserID: userID, CycleDate: pgDate(cycleDate), CheckedInAt: pgTz(now),
		}); err != nil {
			t.Fatalf("first CreateCheckin: %v", err)
		}
		_, err := q.CreateCheckin(ctx, store.CreateCheckinParams{
			ID: mustUUID(t), UserID: userID, CycleDate: pgDate(cycleDate), CheckedInAt: pgTz(now),
		})
		assertPgCode(t, err, "23505", "UNIQUE(user_id, cycle_date)")
	})

	t.Run("keys are unique per user, not globally", func(t *testing.T) {
		userA := p.SeedUser("+5215512340004")
		userB := p.SeedUser("+5215512340005")
		cycleDate := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
		key := "shared-key"

		if _, err := q.CreateCheckin(ctx, store.CreateCheckinParams{
			ID: mustUUID(t), UserID: userA, CycleDate: pgDate(cycleDate), CheckedInAt: pgTz(now), IdempotencyKey: &key,
		}); err != nil {
			t.Fatalf("userA CreateCheckin: %v", err)
		}
		if _, err := q.CreateCheckin(ctx, store.CreateCheckinParams{
			ID: mustUUID(t), UserID: userB, CycleDate: pgDate(cycleDate), CheckedInAt: pgTz(now), IdempotencyKey: &key,
		}); err != nil {
			t.Fatalf("userB CreateCheckin with the same key: %v", err)
		}
	})
}

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("uuid v7: %v", err)
	}
	return id
}

func strPtr(s string) *string { return &s }

func pgDate(t time.Time) pgtype.Date { return pgtype.Date{Time: t, Valid: true} }

func pgTz(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func assertPgCode(t *testing.T, err error, code, what string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected a constraint violation (%s: %s), got nil", code, what)
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != code {
		t.Fatalf("error = %v, want pg code %s (%s)", err, code, what)
	}
}
