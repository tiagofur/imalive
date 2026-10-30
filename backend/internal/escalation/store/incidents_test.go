// Store tests of the incident state machine (R-08 + R-09): guard-of-state
// transitions, the single-claimant rule and the one-open-incident-per-user
// constraint, against the real migrations.
package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tiagofur/imalive/backend/internal/escalation/store"
	"github.com/tiagofur/imalive/backend/internal/platform/db/dbtest"
)

func TestIncidentsStore(t *testing.T) {
	ctx := context.Background()
	p := dbtest.New(t)
	q := store.New(p.Pool)
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	cycle := pgDate(date(2026, 10, 30))

	seedContact := func(userID uuid.UUID, phone string, position int) uuid.UUID {
		t.Helper()
		id := mustUUID(t)
		if _, err := p.Pool.Exec(ctx,
			`INSERT INTO contacts (id, user_id, display_name, phone, position)
			 VALUES ($1, $2, 'María', $3, $4)`, id, userID, phone, position); err != nil {
			t.Fatalf("seed contact: %v", err)
		}
		return id
	}
	seedCheckin := func(userID uuid.UUID, cycleDay time.Time, at time.Time) {
		t.Helper()
		if _, err := p.Pool.Exec(ctx,
			`INSERT INTO checkins (id, user_id, cycle_date, checked_in_at)
			 VALUES ($1, $2, $3, $4)`, mustUUID(t), userID, pgDate(cycleDay), pgTz(at)); err != nil {
			t.Fatalf("seed checkin: %v", err)
		}
	}
	openIncident := func(userID uuid.UUID, lastSignal *time.Time) store.Incident {
		t.Helper()
		sig := pgtype.Timestamptz{}
		if lastSignal != nil {
			sig = pgTz(*lastSignal)
		}
		inc, err := q.CreateIncident(ctx, store.CreateIncidentParams{
			ID: mustUUID(t), UserID: userID, CycleDate: cycle, LastSignalAt: sig,
		})
		if err != nil {
			t.Fatalf("CreateIncident: %v", err)
		}
		return inc
	}

	t.Run("opens with the frozen last known signal", func(t *testing.T) {
		userID := p.SeedUser("+5215560000001")
		signal := now.Add(-14 * time.Hour)
		seedCheckin(userID, date(2026, 10, 29), signal)

		inc := openIncident(userID, &signal)
		if inc.State != "contact1_notified" || inc.ClosedAt.Valid {
			t.Fatalf("new incident = %+v, want contact1_notified and open", inc)
		}
		got, err := q.GetIncident(ctx, inc.ID)
		if err != nil {
			t.Fatalf("GetIncident: %v", err)
		}
		if got.LastSignalAt.Time.UTC() != signal.UTC() {
			t.Fatalf("last_signal_at = %v, want the frozen %v", got.LastSignalAt.Time, signal)
		}
	})

	t.Run("one open incident per user", func(t *testing.T) {
		userID := p.SeedUser("+5215560000002")
		openIncident(userID, nil)

		_, err := q.CreateIncident(ctx, store.CreateIncidentParams{
			ID: mustUUID(t), UserID: userID, CycleDate: cycle,
		})
		assertPgCode(t, err, "23505", "one open incident per user")

		// Closing frees the slot: the next cycle can open a new incident.
		if n, err := q.CloseIncident(ctx, store.CloseIncidentParams{
			ID: mustIncidentOf(t, p, userID), State: "closed_false_alarm",
		}); err != nil || n != 1 {
			t.Fatalf("CloseIncident = %d rows, err %v; want 1/nil", n, err)
		}
		openIncident(userID, nil)
	})

	t.Run("only the first claimant wins", func(t *testing.T) {
		userID := p.SeedUser("+5215560000003")
		contactA := seedContact(userID, "+5215560000031", 1)
		contactB := seedContact(userID, "+5215560000032", 2)
		inc := openIncident(userID, nil)

		n, err := q.ClaimIncident(ctx, store.ClaimIncidentParams{ID: inc.ID, ClaimedBy: pgUUID(contactA)})
		if err != nil || n != 1 {
			t.Fatalf("first claim = %d rows, err %v; want 1/nil", n, err)
		}
		got, _ := q.GetIncident(ctx, inc.ID)
		if got.State != "reclaimed" || got.ClaimedBy.Bytes != contactA {
			t.Fatalf("claimed = %s by %v, want reclaimed by contactA", got.State, got.ClaimedBy)
		}

		// The double tap and the late rival both change nothing.
		n, err = q.ClaimIncident(ctx, store.ClaimIncidentParams{ID: inc.ID, ClaimedBy: pgUUID(contactB)})
		if err != nil || n != 0 {
			t.Fatalf("second claim = %d rows, err %v; want 0/nil (first claimant stands)", n, err)
		}
	})

	t.Run("circle notification escalates from contact1 only", func(t *testing.T) {
		userID := p.SeedUser("+5215560000004")
		inc := openIncident(userID, nil)

		if n, err := q.CircleNotifiedIncident(ctx, inc.ID); err != nil || n != 1 {
			t.Fatalf("circle transition = %d rows, err %v; want 1/nil", n, err)
		}
		// From 'circle_notified' the transition does not repeat.
		if n, err := q.CircleNotifiedIncident(ctx, inc.ID); err != nil || n != 0 {
			t.Fatalf("repeat circle transition = %d rows, err %v; want 0/nil", n, err)
		}
	})

	t.Run("the signal closes exactly once, resolved after a claim", func(t *testing.T) {
		userID := p.SeedUser("+5215560000005")
		contact := seedContact(userID, "+5215560000051", 1)
		inc := openIncident(userID, nil)

		if _, err := q.ClaimIncident(ctx, store.ClaimIncidentParams{ID: inc.ID, ClaimedBy: pgUUID(contact)}); err != nil {
			t.Fatalf("claim: %v", err)
		}
		n, err := q.CloseIncident(ctx, store.CloseIncidentParams{ID: inc.ID, State: "closed_resolved"})
		if err != nil || n != 1 {
			t.Fatalf("close = %d rows, err %v; want 1/nil", n, err)
		}
		got, _ := q.GetIncident(ctx, inc.ID)
		if got.State != "closed_resolved" || !got.ClosedAt.Valid {
			t.Fatalf("closed = %s at %v, want closed_resolved with closed_at", got.State, got.ClosedAt)
		}
		if n, err := q.CloseIncident(ctx, store.CloseIncidentParams{ID: inc.ID, State: "closed_false_alarm"}); err != nil || n != 0 {
			t.Fatalf("second close = %d rows, err %v; want 0/nil", n, err)
		}
	})

	t.Run("the last known signal is the newest checkin or NULL", func(t *testing.T) {
		userID := p.SeedUser("+5215560000006")
		if none := mustLastSignal(t, q, userID); none.Valid {
			t.Fatalf("no checkins = %v, want NULL", none)
		}

		seedCheckin(userID, date(2026, 10, 29), now.Add(-2*time.Hour))
		seedCheckin(userID, date(2026, 10, 30), now.Add(-time.Hour)) // the newest wins
		last := mustLastSignal(t, q, userID)
		if !last.Valid || last.Time.UTC() != now.Add(-time.Hour).UTC() {
			t.Fatalf("last signal = %v, want the newest checkin", last)
		}
	})
}

func mustLastSignal(t *testing.T, q *store.Queries, userID uuid.UUID) pgtype.Timestamptz {
	t.Helper()
	sig, err := q.GetLastSignalAt(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetLastSignalAt: %v", err)
	}
	return sig
}

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

// mustIncidentOf returns the only incident id of the user.
func mustIncidentOf(t *testing.T, p *dbtest.Pool, userID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := p.Pool.QueryRow(context.Background(),
		`SELECT id FROM incidents WHERE user_id = $1 LIMIT 1`, userID).Scan(&id); err != nil {
		t.Fatalf("incident of user: %v", err)
	}
	return id
}
