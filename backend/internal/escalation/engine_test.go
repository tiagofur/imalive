package escalation_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tiagofur/imalive/backend/internal/escalation"
	"github.com/tiagofur/imalive/backend/internal/escalation/store"
	"github.com/tiagofur/imalive/backend/internal/platform/db/dbtest"
)

// fakeClock is the simulated time of the engine tests: the whole motor runs
// under it (motor-escalada.md: "el motor se prueba entero con tiempo simulado").
type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

func testEngine(p *dbtest.Pool, now time.Time) (*escalation.Engine, *fakeClock) {
	clock := &fakeClock{now: now}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return escalation.New(p.Pool, clock, log), clock
}

func TestSchedule(t *testing.T) {
	open := time.Date(2026, 10, 30, 21, 0, 0, 0, time.UTC)
	t.Run("default grace window then fixed offsets", func(t *testing.T) {
		steps := escalation.Schedule(open, time.Hour)
		want := []struct {
			kind  string
			runAt time.Time
		}{
			{escalation.KindReminderSoft, open.Add(time.Hour)},
			{escalation.KindReminderUrgent, open.Add(5 * time.Hour)},
			{escalation.KindNotifyContact, open.Add(9 * time.Hour)},
			{escalation.KindNotifyCircle, open.Add(13 * time.Hour)},
		}
		if len(steps) != len(want) {
			t.Fatalf("steps = %d, want %d", len(steps), len(want))
		}
		for i, w := range want {
			if steps[i].Kind != w.kind || !steps[i].RunAt.Equal(w.runAt) {
				t.Fatalf("step %d = %s@%v, want %s@%v", i, steps[i].Kind, steps[i].RunAt, w.kind, w.runAt)
			}
		}
	})
	t.Run("zero grace escalates from the slot itself", func(t *testing.T) {
		steps := escalation.Schedule(open, 0)
		if !steps[0].RunAt.Equal(open) {
			t.Fatalf("soft = %v, want the open instant %v", steps[0].RunAt, open)
		}
	})
}

func TestEngineOpensCycles(t *testing.T) {
	ctx := context.Background()
	p := dbtest.New(t)
	// America/Mexico_City (UTC-6 on 2026-10-30): the seeded default check-in
	// slot is 21:00 local, the default grace window 60 minutes.
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	local := func(y int, mo time.Month, d, h, mi int) time.Time {
		return time.Date(y, mo, d, h, mi, 0, 0, loc).UTC()
	}

	t.Run("opens the ladder once the slot passes, with simulated time", func(t *testing.T) {
		userID := p.SeedUser("+5215520000001")
		engine, _ := testEngine(p, local(2026, 10, 30, 20, 59))
		engine.ProcessTick(ctx)
		if n := jobCount(t, p, userID); n != 0 {
			t.Fatalf("jobs before the slot = %d, want 0", n)
		}

		engine, _ = testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx)

		if n := jobCount(t, p, userID); n != 4 {
			t.Fatalf("jobs after the slot = %d, want 4 (the whole ladder)", n)
		}
		soft := jobOf(t, p, userID, escalation.KindReminderSoft)
		// The ladder anchors to the slot (21:00) + grace, not to the tick:
		// soft = local 22:00, then +4h per rung.
		if got := soft.RunAt.Time.UTC(); !got.Equal(local(2026, 10, 30, 22, 0)) {
			t.Fatalf("soft run_at = %v, want local 22:00 (%v)", got, local(2026, 10, 30, 22, 0))
		}
		circle := jobOf(t, p, userID, escalation.KindNotifyCircle)
		if got := circle.RunAt.Time.UTC(); !got.Equal(local(2026, 10, 31, 10, 0)) {
			t.Fatalf("circle run_at = %v, want local 10:00 next day", got)
		}
		if got := soft.CycleDate.Time.Format("2006-01-02"); got != "2026-10-30" {
			t.Fatalf("cycle_date = %s, want the user-local date 2026-10-30", got)
		}

		// The sweep is stateless: a second tick is a no-op, not a re-open.
		engine.ProcessTick(ctx)
		if n := jobCount(t, p, userID); n != 4 {
			t.Fatalf("jobs after a second tick = %d, want 4 (idempotent)", n)
		}

		audit := auditEvents(t, p, userID)
		if len(audit) != 1 || audit[0].Kind != escalation.AuditCycleOpened {
			t.Fatalf("audit = %+v, want exactly one cycle.opened", audit)
		}
	})

	t.Run("does not open when the signal already exists", func(t *testing.T) {
		userID := p.SeedUser("+5215520000002")
		cycle := pgDate(time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC))
		if _, err := p.Pool.Exec(ctx,
			`INSERT INTO checkins (id, user_id, cycle_date, checked_in_at)
			 VALUES ($1, $2, $3, $4)`, mustUUID(t), userID, cycle, pgTz(local(2026, 10, 30, 21, 5))); err != nil {
			t.Fatalf("seed checkin: %v", err)
		}
		engine, _ := testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx)
		if n := jobCount(t, p, userID); n != 0 {
			t.Fatalf("jobs with an existing signal = %d, want 0", n)
		}
	})

	t.Run("does not open while a live ladder exists", func(t *testing.T) {
		userID := p.SeedUser("+5215520000003")
		// Yesterday's ladder is still escalating: one active cycle per user.
		engine, _ := testEngine(p, local(2026, 10, 29, 21, 30))
		engine.ProcessTick(ctx)
		if n := jobCount(t, p, userID); n != 4 {
			t.Fatalf("seed ladder = %d jobs, want 4", n)
		}
		// Next day at slot time: the old ladder still lives, no new cycle.
		engine, _ = testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx)
		if n := jobCount(t, p, userID); n != 4 {
			t.Fatalf("jobs with a live ladder = %d, want 4", n)
		}
	})

	t.Run("freezes while paused and opens when the pause expires", func(t *testing.T) {
		userID := p.SeedUser("+5215520000004")
		if _, err := p.Pool.Exec(ctx,
			`UPDATE users SET pause_until = $2 WHERE id = $1`,
			userID, pgTz(local(2026, 10, 30, 22, 0))); err != nil {
			t.Fatalf("seed pause: %v", err)
		}
		engine, _ := testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx)
		if n := jobCount(t, p, userID); n != 0 {
			t.Fatalf("jobs while paused = %d, want 0 (R-11: the ladder freezes)", n)
		}

		engine, _ = testEngine(p, local(2026, 10, 30, 22, 30))
		engine.ProcessTick(ctx)
		if n := jobCount(t, p, userID); n != 4 {
			t.Fatalf("jobs after the pause expired = %d, want 4 (re-opened)", n)
		}
		soft := jobOf(t, p, userID, escalation.KindReminderSoft)
		if got := soft.RunAt.Time.UTC(); !got.Equal(local(2026, 10, 30, 22, 0)) {
			t.Fatalf("soft after resume = %v, want local 22:00 (anchored to the slot; overdue, so it fires on the next tick)", got)
		}
	})

	t.Run("one corrupt user never stalls the sweep", func(t *testing.T) {
		good := p.SeedUser("+5215520000005")
		bad := p.SeedUser("+5215520000006")
		if _, err := p.Pool.Exec(ctx,
			`UPDATE users SET timezone = 'Mars/Olympus' WHERE id = $1`, bad); err != nil {
			t.Fatalf("corrupt timezone: %v", err)
		}
		engine, _ := testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx) // must not panic and must not skip the good user
		if n := jobCount(t, p, good); n != 4 {
			t.Fatalf("good user jobs = %d, want 4 (sweep continued past the corrupt row)", n)
		}
		if n := jobCount(t, p, bad); n != 0 {
			t.Fatalf("corrupt user jobs = %d, want 0", n)
		}
	})
}

func jobCount(t *testing.T, p *dbtest.Pool, userID uuid.UUID) int {
	t.Helper()
	var n int
	if err := p.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM escalation_jobs WHERE user_id = $1`, userID).Scan(&n); err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	return n
}

func jobOf(t *testing.T, p *dbtest.Pool, userID uuid.UUID, kind string) store.EscalationJob {
	t.Helper()
	rows, err := p.Pool.Query(context.Background(),
		`SELECT id, user_id, cycle_date, kind, run_at, state, attempts
		 FROM escalation_jobs WHERE user_id = $1 AND kind = $2`, userID, kind)
	if err != nil {
		t.Fatalf("query job %s: %v", kind, err)
	}
	defer rows.Close()
	var row store.EscalationJob
	found := false
	for rows.Next() {
		if err := rows.Scan(&row.ID, &row.UserID, &row.CycleDate, &row.Kind, &row.RunAt, &row.State, &row.Attempts); err != nil {
			t.Fatalf("scan job: %v", err)
		}
		found = true
	}
	if !found {
		t.Fatalf("job %s not found", kind)
	}
	return row
}

func auditEvents(t *testing.T, p *dbtest.Pool, userID uuid.UUID) []store.AuditEvent {
	t.Helper()
	rows, err := store.New(p.Pool).ListAuditEventsByUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListAuditEventsByUser: %v", err)
	}
	return rows
}

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("uuid v7: %v", err)
	}
	return id
}

func pgDate(d time.Time) pgtype.Date { return pgtype.Date{Time: d, Valid: true} }

func pgTz(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
