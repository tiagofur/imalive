// Package store_test verifies the escalation store against a real Postgres
// with the production migrations: the queue claim (SKIP LOCKED + lease), the
// idempotent ladder open, the at-least-once outbox dedupe and the append-only
// audit trigger (motor-escalada.md, ADR-0004).
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

// Job states mirror the CHECK constraint of migration 000005; the engine
// package (next slice) owns the domain constants.
const (
	statePending   = "pending"
	stateRunning   = "running"
	stateDone      = "done"
	stateCancelled = "cancelled"
)

func TestEscalationStore(t *testing.T) {
	ctx := context.Background()
	p := dbtest.New(t)
	q := store.New(p.Pool)
	// Fixed base instant: Postgres stores timestamptz at microsecond
	// precision, so a wall-clock now (nanoseconds) flakily misses equality
	// on the round-trip. Deterministic time keeps the assertions exact.
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

	t.Run("opens the ladder idempotently and re-arms only cancelled jobs", func(t *testing.T) {
		userID := p.SeedUser("+5215510000001")
		cycle := pgDate(date(2026, 10, 30))

		seedJob(t, p, userID, cycle, "reminder_soft", now.Add(time.Hour))
		seedJob(t, p, userID, cycle, "reminder_urgent", now.Add(2*time.Hour))

		// Re-open with a fresh identity: UNIQUE (user_id, cycle_date, kind)
		// makes the race between workers harmless and keeps the first row.
		again := now.Add(3 * time.Hour)
		createJob(t, q, userID, cycle, "reminder_soft", again)
		soft := findJob(t, p, userID, "reminder_soft")
		if soft.RunAt.Time.UTC() != now.Add(time.Hour).UTC() {
			t.Fatalf("reminder_soft run_at moved on re-open, want the original %v", soft.RunAt.Time)
		}

		// A cancelled step re-arms with the new schedule (pause → resume);
		// the steps the sweeper does not ask for stay cancelled.
		if _, err := q.CancelPendingJobsByUser(ctx, userID); err != nil {
			t.Fatalf("CancelPendingJobsByUser: %v", err)
		}
		resumed := again.Add(30 * time.Minute)
		createJob(t, q, userID, cycle, "reminder_soft", resumed)
		soft = findJob(t, p, userID, "reminder_soft")
		if soft.State != statePending || soft.RunAt.Time.UTC() != resumed.UTC() {
			t.Fatalf("cancelled job = %s at %v, want re-armed pending at %v", soft.State, soft.RunAt.Time, resumed)
		}
		if urgent := findJob(t, p, userID, "reminder_urgent"); urgent.State != stateCancelled {
			t.Fatalf("unrequested cancelled step = %s, want still cancelled", urgent.State)
		}

		// A done step never re-notifies.
		if err := q.CompleteJob(ctx, soft.ID); err != nil {
			t.Fatalf("CompleteJob: %v", err)
		}
		createJob(t, q, userID, cycle, "reminder_soft", again)
		if soft := findJob(t, p, userID, "reminder_soft"); soft.State != stateDone {
			t.Fatalf("done job reopened to %s, want done (no re-notification)", soft.State)
		}
	})

	t.Run("claims only due jobs and honors the running lease", func(t *testing.T) {
		userID := p.SeedUser("+5215510000002")
		cycle := pgDate(date(2026, 10, 30))

		due := seedJob(t, p, userID, cycle, "reminder_soft", now.Add(-time.Minute))
		later := seedJob(t, p, userID, cycle, "reminder_urgent", now.Add(time.Hour))
		leased := seedJob(t, p, userID, cycle, "notify_contact", now.Add(-2*time.Minute))
		// A running job with a live lease belongs to another worker: untouched.
		setJobState(t, p, leased.ID, stateRunning, pgTz(now.Add(5*time.Minute)))

		claim := func() []store.EscalationJob {
			t.Helper()
			rows, err := q.ClaimDueJobs(ctx, store.ClaimDueJobsParams{
				LockedUntil: pgTz(now.Add(5 * time.Minute)),
				RunAt:       pgTz(now),
				Limit:       10,
			})
			if err != nil {
				t.Fatalf("ClaimDueJobs: %v", err)
			}
			return rows
		}

		claimed := claim()
		if len(claimed) != 1 || claimed[0].ID != due.ID {
			t.Fatalf("claimed = %v, want only the due job %s", jobIDs(claimed), due.ID)
		}
		if claimed[0].State != stateRunning || claimed[0].Attempts != 1 {
			t.Fatalf("claimed state = %s attempts = %d, want running/1", claimed[0].State, claimed[0].Attempts)
		}

		// After the lease expires the job is reclaimable (at-least-once).
		setJobState(t, p, leased.ID, stateRunning, pgTz(now.Add(-time.Minute)))
		reclaimed := claim()
		if len(reclaimed) != 1 || reclaimed[0].ID != leased.ID {
			t.Fatalf("reclaimed = %v, want the expired-lease job %s", jobIDs(reclaimed), leased.ID)
		}
		if reclaimed[0].Attempts != 1 {
			t.Fatalf("reclaim attempts = %d, want 1 (the claim counts it)", reclaimed[0].Attempts)
		}

		// A later job is never claimed ahead of time.
		if got := findJob(t, p, userID, "reminder_urgent"); got.ID != later.ID || got.State != statePending {
			t.Fatalf("future job touched: %+v", got)
		}
	})

	t.Run("SKIP LOCKED claims around a row locked by a parallel worker", func(t *testing.T) {
		userID := p.SeedUser("+5215510000003")
		cycle := pgDate(date(2026, 10, 30))
		locked := seedJob(t, p, userID, cycle, "reminder_soft", now.Add(-time.Minute))
		free := seedJob(t, p, userID, cycle, "notify_contact", now.Add(-2*time.Minute))

		// Another worker holds the row lock from an open transaction: the
		// claim must skip it, not block, and take the rest.
		tx, err := p.Pool.Begin(ctx)
		if err != nil {
			t.Fatalf("Begin: %v", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx,
			`SELECT id FROM escalation_jobs WHERE id = $1 FOR UPDATE`, locked.ID); err != nil {
			t.Fatalf("lock job: %v", err)
		}

		claimed, err := q.ClaimDueJobs(ctx, store.ClaimDueJobsParams{
			LockedUntil: pgTz(now.Add(5 * time.Minute)),
			RunAt:       pgTz(now),
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("ClaimDueJobs with a locked row: %v", err)
		}
		if len(claimed) != 1 || claimed[0].ID != free.ID {
			t.Fatalf("claimed = %v, want only the unlocked job %s", jobIDs(claimed), free.ID)
		}
	})

	t.Run("cancels only the pending jobs of the user", func(t *testing.T) {
		userID := p.SeedUser("+5215510000004")
		cycle := pgDate(date(2026, 10, 30))
		seedJob(t, p, userID, cycle, "reminder_soft", now)
		seedJob(t, p, userID, cycle, "reminder_urgent", now)
		contact := seedJob(t, p, userID, cycle, "notify_contact", now)
		if err := q.CompleteJob(ctx, contact.ID); err != nil {
			t.Fatalf("CompleteJob: %v", err)
		}

		n, err := q.CancelPendingJobsByUser(ctx, userID)
		if err != nil {
			t.Fatalf("CancelPendingJobsByUser: %v", err)
		}
		if n != 2 {
			t.Fatalf("cancelled %d jobs, want 2 (done jobs untouched)", n)
		}
		for _, kind := range []string{"reminder_soft", "reminder_urgent"} {
			if got := findJob(t, p, userID, kind); got.State != stateCancelled {
				t.Fatalf("%s = %s, want cancelled", kind, got.State)
			}
		}
		if got := findJob(t, p, userID, "notify_contact"); got.State != stateDone {
			t.Fatalf("done job moved to %s, want done", got.State)
		}
	})

	t.Run("outbox dedupes a re-executed job by job_id and destination", func(t *testing.T) {
		userID := p.SeedUser("+5215510000005")
		jobID := seedJob(t, p, userID, pgDate(date(2026, 10, 30)), "notify_contact", now.Add(-time.Minute)).ID
		jobArg := pgUUID(jobID)

		first := store.InsertOutboxEventParams{
			ID: mustUUID(t), UserID: userID, Channel: "whatsapp",
			Destination: "+5215511111111", Payload: []byte(`{"kind":"ladder.notify_contact"}`),
			JobID: jobArg,
		}
		if err := q.InsertOutboxEvent(ctx, first); err != nil {
			t.Fatalf("InsertOutboxEvent: %v", err)
		}
		// The same job re-executed (at-least-once): same job_id + destination.
		dup := first
		dup.ID = mustUUID(t)
		if err := q.InsertOutboxEvent(ctx, dup); err != nil {
			t.Fatalf("deduped insert: %v", err)
		}
		if n := countRows(t, p, `SELECT count(*) FROM outbox_event WHERE job_id = $1`, jobArg); n != 1 {
			t.Fatalf("outbox rows = %d, want 1 (at-least-once dedupe)", n)
		}
		// A different destination notifies too: same job, distinct row.
		other := first
		other.ID = mustUUID(t)
		other.Destination = "+5215522222222"
		if err := q.InsertOutboxEvent(ctx, other); err != nil {
			t.Fatalf("second destination: %v", err)
		}
		if n := countRows(t, p, `SELECT count(*) FROM outbox_event WHERE job_id = $1`, jobArg); n != 2 {
			t.Fatalf("outbox rows = %d, want 2", n)
		}
	})

	t.Run("audit_events is append-only", func(t *testing.T) {
		userID := p.SeedUser("+5215510000006")
		if err := q.InsertAuditEvent(ctx, store.InsertAuditEventParams{
			UserID: userID, CycleDate: pgDate(date(2026, 10, 30)),
			Kind: "cycle.opened", Data: []byte(`{"ladder":4}`),
		}); err != nil {
			t.Fatalf("InsertAuditEvent: %v", err)
		}
		rows, err := q.ListAuditEventsByUser(ctx, userID)
		if err != nil {
			t.Fatalf("ListAuditEventsByUser: %v", err)
		}
		if len(rows) != 1 || rows[0].Kind != "cycle.opened" {
			t.Fatalf("audit rows = %+v, want the inserted event", rows)
		}

		// UPDATE and DELETE are evidence tampering: blocked by the trigger
		// for every role, the schema owner included (R-07, spec §Auditoría).
		for _, attempt := range []struct {
			name string
			sql  string
		}{
			{"update", `UPDATE audit_events SET kind = 'tampered'`},
			{"delete", `DELETE FROM audit_events`},
		} {
			t.Run(attempt.name+" is rejected", func(t *testing.T) {
				_, err := p.Pool.Exec(ctx, attempt.sql)
				var pgErr *pgconn.PgError
				if err == nil || !errors.As(err, &pgErr) || pgErr.Code != "P0001" {
					t.Fatalf("mutation error = %v, want raise_exception P0001", err)
				}
			})
		}
	})
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func createJob(t *testing.T, q *store.Queries, userID uuid.UUID, cycle pgtype.Date, kind string, runAt time.Time) {
	t.Helper()
	if err := q.CreateLadderJob(context.Background(), store.CreateLadderJobParams{
		ID: mustUUID(t), UserID: userID, CycleDate: cycle, Kind: kind, RunAt: pgTz(runAt),
	}); err != nil {
		t.Fatalf("CreateLadderJob(%s): %v", kind, err)
	}
}

func seedJob(t *testing.T, p *dbtest.Pool, userID uuid.UUID, cycle pgtype.Date, kind string, runAt time.Time) store.EscalationJob {
	t.Helper()
	createJob(t, store.New(p.Pool), userID, cycle, kind, runAt)
	return findJob(t, p, userID, kind)
}

func findJob(t *testing.T, p *dbtest.Pool, userID uuid.UUID, kind string) store.EscalationJob {
	t.Helper()
	rows, err := p.Pool.Query(context.Background(),
		`SELECT id, user_id, cycle_date, kind, run_at, state, attempts FROM escalation_jobs
		 WHERE user_id = $1 AND kind = $2`, userID, kind)
	if err != nil {
		t.Fatalf("query job %s: %v", kind, err)
	}
	defer rows.Close()
	var row store.EscalationJob
	for rows.Next() {
		if err := rows.Scan(&row.ID, &row.UserID, &row.CycleDate, &row.Kind, &row.RunAt, &row.State, &row.Attempts); err != nil {
			t.Fatalf("scan job: %v", err)
		}
	}
	if row.Kind == "" {
		t.Fatalf("job %s not found", kind)
	}
	return row
}

// setJobState writes the raw state for fixtures the store API cannot produce
// directly (a running job holding a lease belongs to a crashed worker).
func setJobState(t *testing.T, p *dbtest.Pool, id uuid.UUID, state string, lockedUntil pgtype.Timestamptz) {
	t.Helper()
	if _, err := p.Pool.Exec(context.Background(),
		`UPDATE escalation_jobs SET state = $2, locked_until = $3 WHERE id = $1`, id, state, lockedUntil); err != nil {
		t.Fatalf("set state %s: %v", state, err)
	}
}

func countRows(t *testing.T, p *dbtest.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := p.Pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func jobIDs(rows []store.EscalationJob) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(rows))
	for _, j := range rows {
		out = append(out, j.ID)
	}
	return out
}

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("uuid v7: %v", err)
	}
	return id
}

func pgUUID(u uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: u, Valid: true} }

func pgDate(t time.Time) pgtype.Date { return pgtype.Date{Time: t, Valid: true} }

func pgTz(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
