// White-box tests of the failure paths: they inject errors into the engine's
// resolver seam to exercise backoff and the dead-letter without real
// breakage. The black-box scenarios live in executor_test.go.
package escalation

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/tiagofur/imalive/backend/internal/escalation/store"
	"github.com/tiagofur/imalive/backend/internal/platform/db/dbtest"
)

type internalClock struct{ now time.Time }

func (c *internalClock) Now() time.Time { return c.now }

func TestBackoff(t *testing.T) {
	cases := []struct {
		attempts int16
		want     time.Duration
	}{
		{1, time.Minute},
		{2, 2 * time.Minute},
		{3, 4 * time.Minute},
		{7, time.Hour},   // capped
		{0, time.Minute}, // defensive: never zero or negative
	}
	for _, c := range cases {
		if got := backoff(c.attempts); got != c.want {
			t.Fatalf("backoff(%d) = %v, want %v", c.attempts, got, c.want)
		}
	}
}

func TestEngineRetriesAndDeadLetters(t *testing.T) {
	ctx := context.Background()
	p := dbtest.New(t)
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	at := func(y int, mo time.Month, d, h, mi int) time.Time {
		return time.Date(y, mo, d, h, mi, 0, 0, loc).UTC()
	}

	userID := p.SeedUser("+5215540000001")
	clock := &internalClock{now: at(2026, 10, 30, 21, 30)}
	engine := New(p.Pool, clock, slog.New(slog.NewTextHandler(io.Discard, nil)))
	engine.ProcessTick(ctx) // opens the cycle

	boom := errors.New("resolve exploded")
	engine.resolve = func(context.Context, store.EscalationJob) ([]store.InsertOutboxEventParams, error) {
		return nil, boom
	}

	jobAt := func() store.EscalationJob {
		rows, err := p.Pool.Query(ctx,
			`SELECT id, user_id, cycle_date, kind, run_at, state, attempts, max_attempts, locked_until, last_error, created_at, updated_at
			 FROM escalation_jobs WHERE user_id = $1 AND kind = $2`, userID, KindReminderSoft)
		if err != nil {
			t.Fatalf("query job: %v", err)
		}
		defer rows.Close()
		var row store.EscalationJob
		found := false
		for rows.Next() {
			if err := rows.Scan(&row.ID, &row.UserID, &row.CycleDate, &row.Kind, &row.RunAt,
				&row.State, &row.Attempts, &row.MaxAttempts, &row.LockedUntil, &row.LastError,
				&row.CreatedAt, &row.UpdatedAt); err != nil {
				t.Fatalf("scan job: %v", err)
			}
			found = true
		}
		if !found {
			t.Fatal("reminder_soft not found")
		}
		return row
	}

	// Attempt 1 fails: pending again with a 1-minute backoff, error recorded.
	clock.now = at(2026, 10, 30, 22, 0) // soft due
	engine.ProcessTick(ctx)
	job := jobAt()
	if job.State != "pending" || job.Attempts != 1 || job.LastError == nil {
		t.Fatalf("after 1st failure: state=%s attempts=%d last_error=%v, want pending/1/recorded", job.State, job.Attempts, job.LastError)
	}
	if got, want := job.RunAt.Time.UTC(), at(2026, 10, 30, 22, 1); !got.Equal(want) {
		t.Fatalf("retry run_at = %v, want +1min backoff (%v)", got, want)
	}

	// The backoff holds: a tick inside the window claims nothing.
	clock.now = at(2026, 10, 30, 22, 0).Add(30 * time.Second)
	engine.ProcessTick(ctx)
	if got := jobAt(); got.Attempts != 1 {
		t.Fatalf("tick inside the backoff moved attempts to %d, want 1", got.Attempts)
	}

	// Attempt 2 fails: 2-minute backoff. Attempt 3 exhausts: dead-letter.
	clock.now = at(2026, 10, 30, 22, 2)
	engine.ProcessTick(ctx)
	if got := jobAt(); got.Attempts != 2 || got.State != "pending" {
		t.Fatalf("after 2nd failure: attempts=%d state=%s, want 2/pending", got.Attempts, got.State)
	}
	clock.now = at(2026, 10, 30, 22, 5)
	engine.ProcessTick(ctx)
	job = jobAt()
	if job.State != "failed" || job.Attempts != 3 {
		t.Fatalf("after 3rd failure: state=%s attempts=%d, want failed/3", job.State, job.Attempts)
	}

	// The dead-letter is loud: exactly one job.failed audit row, no outbox.
	audit, err := store.New(p.Pool).ListAuditEventsByUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListAuditEventsByUser: %v", err)
	}
	fails := 0
	for _, ev := range audit {
		if ev.Kind == AuditJobFailed {
			fails++
		}
	}
	if fails != 1 {
		t.Fatalf("audit job.failed rows = %d, want 1", fails)
	}
	if _, err := engine.resolve(ctx, job); !errors.Is(err, boom) {
		t.Fatalf("resolve = %v, want the injected error", err)
	}
}
