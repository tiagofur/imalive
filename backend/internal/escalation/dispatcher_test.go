// Dispatcher tests against a real Postgres: claim → deliver → mark with the
// queue semantics (SKIP LOCKED, lease) and the at-least-once idempotency the
// spec demands (motor-escalada.md §Outbox transaccional).
package escalation_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tiagofur/imalive/backend/internal/escalation"
	"github.com/tiagofur/imalive/backend/internal/platform/db/dbtest"
	"github.com/tiagofur/imalive/backend/internal/platform/push"
)

// recorderNotifier records texts per phone (a sandbox stand-in for Twilio).
type recorderNotifier struct {
	sent map[string][]string
}

func newRecorderNotifier() *recorderNotifier {
	return &recorderNotifier{sent: map[string][]string{}}
}

func (r *recorderNotifier) Send(_ context.Context, phone, text string) error {
	r.sent[phone] = append(r.sent[phone], text)
	return nil
}

// failingNotifier fails every send (the retry/dead-letter path).
type failingNotifier struct{ calls int }

func (f *failingNotifier) Send(context.Context, string, string) error {
	f.calls++
	return errDispatchBoom
}

var errDispatchBoom = &boomError{}

type boomError struct{}

func (*boomError) Error() string { return "dispatch exploded" }

func seedOutbox(t *testing.T, p *dbtest.Pool, at time.Time, userID uuid.UUID, channel, destination, payload string) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	// created_at must be claimable by the simulated clock (the dispatcher
	// compares it against the injected Clock, not the DB now()).
	if _, err := p.Pool.Exec(context.Background(),
		`INSERT INTO outbox_event (id, user_id, channel, destination, payload, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`, id, userID, channel, destination, []byte(payload), pgTz(at.Add(-time.Minute))); err != nil {
		t.Fatalf("seed outbox: %v", err)
	}
	return id
}

func TestDispatcher(t *testing.T) {
	ctx := context.Background()
	p := dbtest.New(t)
	out := io.Writer(io.Discard)
	if os.Getenv("ESCALATION_TEST_LOG") != "" {
		out = os.Stderr
	}
	log := slog.New(slog.NewTextHandler(out, nil))
	userID := p.SeedUser("+5215598000001")
	clock := &fakeClock{now: time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)}

	t.Run("delivers whatsapp events and marks them sent", func(t *testing.T) {
		notifier := newRecorderNotifier()
		d := escalation.NewDispatcher(p.Pool, clock, log, notifier, push.NewSandboxSender())
		id := seedOutbox(t, p, clock.now, userID, "whatsapp", "+5215598000001",
			`{"event":"escalation.notify","kind":"reminder_soft","user_name":"Ana"}`)

		d.DispatchTick(ctx)

		if got := notifier.sent["+5215598000001"]; len(got) != 1 || got[0] != "👋 Hola Ana, ¿todo bien? Confirma que estás OK." {
			t.Fatalf("delivered = %q, want the soft reminder copy", got)
		}
		var state string
		if err := p.Pool.QueryRow(ctx, `SELECT state FROM outbox_event WHERE id = $1`, id).Scan(&state); err != nil || state != "sent" {
			t.Fatalf("event state = %s (err %v), want sent", state, err)
		}
	})

	t.Run("composes the R-08 alert copy with the claimable link", func(t *testing.T) {
		notifier := newRecorderNotifier()
		d := escalation.NewDispatcher(p.Pool, clock, log, notifier, push.NewSandboxSender())
		seedOutbox(t, p, clock.now, userID, "whatsapp", "+5215598000099",
			`{"event":"incident.alert","user_name":"Ana","opened_at":"2026-10-31T04:12:00Z","last_signal_at":"2026-10-30T03:05:00Z","token":"tok123","base_url":"https://tranqui.test"}`)

		d.DispatchTick(ctx)

		got := notifier.sent["+5215598000099"]
		if len(got) != 1 {
			t.Fatalf("deliveries = %d, want 1", len(got))
		}
		for _, want := range []string{"Ana no ha confirmado", "Última señal conocida", "https://tranqui.test/alert/tok123"} {
			if !strings.Contains(got[0], want) {
				t.Fatalf("alert copy %q missing %q", got[0], want)
			}
		}
	})

	t.Run("push events reach the push sender with event-id dedupe", func(t *testing.T) {
		d := escalation.NewDispatcher(p.Pool, clock, log, newRecorderNotifier(), push.NewSandboxSender())
		pushSandbox := push.NewSandboxSender()
		d2 := escalation.NewDispatcher(p.Pool, clock, log, newRecorderNotifier(), pushSandbox)
		id := seedOutbox(t, p, clock.now, userID, "push", "fcm-token-1",
			`{"event":"escalation.notify","kind":"reminder_soft","user_name":"Ana"}`)

		d.DispatchTick(ctx)
		// Simulate the at-least-once re-send: the event goes back to pending
		// (a crash between send and mark) and the dispatcher runs again.
		if _, err := p.Pool.Exec(ctx,
			`UPDATE outbox_event SET state = 'pending', attempts = 0 WHERE id = $1`, id); err != nil {
			t.Fatalf("reset event: %v", err)
		}
		d2.DispatchTick(ctx)

		deliveries := pushSandbox.Deliveries()
		if len(deliveries) != 1 {
			t.Fatalf("push deliveries = %d, want 1 (the provider dedupes by event_id)", len(deliveries))
		}
		if deliveries[0].Token != "fcm-token-1" || deliveries[0].EventID != id.String() {
			t.Fatalf("delivery = %+v, want the token and the event id", deliveries[0])
		}
	})

	t.Run("failures retry and then dead-letter visibly", func(t *testing.T) {
		notifier := &failingNotifier{}
		d := escalation.NewDispatcher(p.Pool, clock, log, notifier, push.NewSandboxSender())
		id := seedOutbox(t, p, clock.now, userID, "whatsapp", "+5215598000002", `{"event":"escalation.notify"}`)

		for i := 0; i < 3; i++ {
			d.DispatchTick(ctx)
		}
		// The claim counts as the attempt: 3 ticks = 3 sends, then dead-letter.
		if notifier.calls != 3 {
			t.Fatalf("send attempts = %d, want 3 (one per claim, capped)", notifier.calls)
		}
		d.DispatchTick(ctx) // nothing left to claim
		if notifier.calls != 3 {
			t.Fatalf("attempts after dead-letter = %d, want 3 (failed does not retry)", notifier.calls)
		}
		var state string
		if err := p.Pool.QueryRow(ctx, `SELECT state FROM outbox_event WHERE id = $1`, id).Scan(&state); err != nil || state != "failed" {
			t.Fatalf("event state = %s (err %v), want failed", state, err)
		}
	})

	t.Run("unknown payloads dead-letter instead of crashing", func(t *testing.T) {
		d := escalation.NewDispatcher(p.Pool, clock, log, newRecorderNotifier(), push.NewSandboxSender())
		seedOutbox(t, p, clock.now, userID, "whatsapp", "+5215598000003", `{"event":"mystery.event"}`)
		d.DispatchTick(ctx)
		var failed int
		if err := p.Pool.QueryRow(ctx,
			`SELECT count(*) FROM outbox_event WHERE user_id = $1 AND state = 'failed'`, userID).Scan(&failed); err != nil || failed != 1 {
			t.Fatalf("failed events = %d (err %v), want 1", failed, err)
		}
	})
}
