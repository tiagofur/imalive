package escalation_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tiagofur/imalive/backend/internal/escalation"
	"github.com/tiagofur/imalive/backend/internal/platform/db/dbtest"
)

// The execution tests drive the whole motor under simulated time: a full day
// of silence walks the ladder to the circle; a pause landing mid-flight
// cancels instead of notifying (motor-escalada.md, escenarios obligatorios
// 2, 6 y 9; los 3/4/5 son del incidente #55).
func TestEngineExecutesLadder(t *testing.T) {
	ctx := context.Background()
	p := dbtest.New(t)
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	local := func(y int, mo time.Month, d, h, mi int) time.Time {
		return time.Date(y, mo, d, h, mi, 0, 0, loc).UTC()
	}

	t.Run("a full day of silence walks the ladder to the circle", func(t *testing.T) {
		userID := p.SeedUser("+5215530000001")
		contactPhone := "+5215530000099"
		seedContact(t, p, userID, "María", contactPhone, 1)
		engine, clock := testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx) // opens the cycle at the slot

		type rung struct {
			at    time.Time
			kind  string
			phone string
		}
		rungs := []rung{
			{local(2026, 10, 30, 22, 0), escalation.KindReminderSoft, "+5215530000001"},
			{local(2026, 10, 31, 2, 0), escalation.KindReminderUrgent, "+5215530000001"},
			{local(2026, 10, 31, 6, 0), escalation.KindNotifyContact, contactPhone},
			{local(2026, 10, 31, 10, 0), escalation.KindNotifyCircle, contactPhone},
		}
		for i, r := range rungs {
			clock.now = r.at
			engine.ProcessTick(ctx)
			if n := outboxCount(t, p, userID); n != i+1 {
				t.Fatalf("after rung %d: outbox = %d events, want %d", i+1, n, i+1)
			}
			if got := lastEventDestination(t, p, userID); got != r.phone {
				t.Fatalf("rung %s notified %s, want %s", r.kind, got, r.phone)
			}
			job := jobOf(t, p, userID, r.kind)
			if job.State != "done" {
				t.Fatalf("rung %s state = %s, want done", r.kind, job.State)
			}
			if n := auditKindCount(t, p, userID, escalation.AuditLadderNotified); n != i+1 {
				t.Fatalf("rung %s: audit notified = %d rows, want %d", r.kind, n, i+1)
			}
		}

		// Silence, not signal: every state change left evidence (spec §Auditoría).
		if n := auditKindCount(t, p, userID, escalation.AuditCycleOpened); n != 1 {
			t.Fatalf("audit cycle.opened = %d rows, want 1", n)
		}
		// Re-running the tick at the same instant cannot duplicate anything.
		engine.ProcessTick(ctx)
		if n := outboxCount(t, p, userID); n != 4 {
			t.Fatalf("outbox after re-tick = %d events, want 4 (no duplicates)", n)
		}
	})

	t.Run("no contacts: the contact rungs complete without notifications", func(t *testing.T) {
		userID := p.SeedUser("+5215530000002")
		engine, clock := testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx)
		clock.now = local(2026, 10, 31, 6, 0)
		engine.ProcessTick(ctx)
		// The two reminder rungs fired to the user's own phone; the contact
		// rung had nobody to notify and completed without crashing.
		if n := outboxCount(t, p, userID); n != 2 {
			t.Fatalf("outbox with no contacts = %d events, want 2 (only the user's reminders)", n)
		}
		if job := jobOf(t, p, userID, escalation.KindNotifyContact); job.State != "done" {
			t.Fatalf("notify_contact state = %s, want done (never crashes on an empty circle)", job.State)
		}
	})

	t.Run("a pause landing mid-flight cancels the rung instead of notifying", func(t *testing.T) {
		userID := p.SeedUser("+5215530000003")
		engine, clock := testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx) // cycle open, soft not due yet

		// The user pauses before the soft reminder is due (R-11).
		if _, err := p.Pool.Exec(ctx,
			`UPDATE users SET pause_until = $2 WHERE id = $1`,
			userID, pgTz(local(2026, 10, 31, 12, 0))); err != nil {
			t.Fatalf("seed pause: %v", err)
		}
		clock.now = local(2026, 10, 30, 22, 0)
		engine.ProcessTick(ctx)

		if n := outboxCount(t, p, userID); n != 0 {
			t.Fatalf("outbox while paused = %d events, want 0 (nothing notifies during a pause)", n)
		}
		job := jobOf(t, p, userID, escalation.KindReminderSoft)
		if job.State != "cancelled" {
			t.Fatalf("soft state = %s, want cancelled", job.State)
		}
		if n := auditKindCount(t, p, userID, escalation.AuditLadderFrozen); n != 1 {
			t.Fatalf("audit frozen = %d rows, want 1 (the why is on the record)", n)
		}
	})
}

func seedContact(t *testing.T, p *dbtest.Pool, userID uuid.UUID, name, phone string, position int) {
	t.Helper()
	if _, err := p.Pool.Exec(context.Background(),
		`INSERT INTO contacts (id, user_id, display_name, phone, position)
		 VALUES ($1, $2, $3, $4, $5)`,
		mustUUID(t), userID, name, phone, position); err != nil {
		t.Fatalf("seed contact: %v", err)
	}
}

func outboxCount(t *testing.T, p *dbtest.Pool, userID uuid.UUID) int {
	t.Helper()
	var n int
	if err := p.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM outbox_event WHERE user_id = $1`, userID).Scan(&n); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	return n
}

func lastEventDestination(t *testing.T, p *dbtest.Pool, userID uuid.UUID) string {
	t.Helper()
	var dest string
	if err := p.Pool.QueryRow(context.Background(),
		`SELECT destination FROM outbox_event WHERE user_id = $1
		 ORDER BY created_at DESC, id DESC LIMIT 1`, userID).Scan(&dest); err != nil {
		t.Fatalf("query outbox: %v", err)
	}
	return dest
}

func auditKindCount(t *testing.T, p *dbtest.Pool, userID uuid.UUID, kind string) int {
	t.Helper()
	var n int
	if err := p.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_events WHERE user_id = $1 AND kind = $2`, userID, kind).Scan(&n); err != nil {
		t.Fatalf("count audit %s: %v", kind, err)
	}
	return n
}
