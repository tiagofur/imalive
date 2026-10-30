package escalation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tiagofur/imalive/backend/internal/escalation/store"
	"github.com/tiagofur/imalive/backend/internal/platform/msg"
	"github.com/tiagofur/imalive/backend/internal/platform/push"
)

// Dispatcher tuning: the outbox SLO (age < 5 min) against the 30 s worker
// tick leaves room for a few retries before the dead-letter.
const (
	dispatchBatch    = 10
	dispatchLease    = 5 * time.Minute
	dispatchMaxTries = 3
)

// outboxPayload is the minimum envelope every outbox event carries; the
// message copy is composed from these fields (never from raw user input).
type outboxPayload struct {
	Event       string `json:"event"`
	Kind        string `json:"kind"`
	UserName    string `json:"user_name"`
	ClaimerName string `json:"claimer_name"`
	OpenedAt    string `json:"opened_at"`
	LastSignal  any    `json:"last_signal_at"`
	Token       string `json:"token"`
	BaseURL     string `json:"base_url"`
}

// Dispatcher drains the transactional outbox (ADR-0004): claim with the same
// queue pattern as the jobs (SKIP LOCKED + lease), deliver OUTSIDE the
// transaction, mark sent — a crash between send and mark is the accepted
// at-least-once re-send, absorbed by event-id dedupe at the provider.
type Dispatcher struct {
	pool     *pgxpool.Pool
	store    *store.Queries
	clock    Clock
	log      *slog.Logger
	messages msg.Notifier
	push     push.Sender
}

// NewDispatcher wires the dispatcher over a pool with the two channel
// senders (WhatsApp/SMS text via msg.Notifier, device push via push.Sender).
func NewDispatcher(pool *pgxpool.Pool, clock Clock, log *slog.Logger, messages msg.Notifier, pushSender push.Sender) *Dispatcher {
	return &Dispatcher{
		pool: pool, store: store.New(pool), clock: clock, log: log,
		messages: messages, push: pushSender,
	}
}

// DispatchTick claims and delivers one batch of due outbox events. A tick
// never fails the caller: errors are logged and retried next tick.
func (d *Dispatcher) DispatchTick(ctx context.Context) {
	now := d.clock.Now()
	events, err := d.store.ClaimDueOutboxEvents(ctx, store.ClaimDueOutboxEventsParams{
		CreatedAt:    pgTz(now),
		DispatchedAt: pgTz(now.Add(-dispatchLease)),
		Limit:        dispatchBatch,
	})
	if err != nil {
		d.log.Error("dispatch: claim", slog.String("error", err.Error()))
		return
	}
	for _, ev := range events {
		d.deliver(ctx, ev)
	}
}

// deliver sends one event through its channel and records the outcome. The
// send is deliberately NOT in a transaction with the mark: the network call
// cannot roll back, and the claim state ('sending' + lease) already prevents
// parallel double delivery.
func (d *Dispatcher) deliver(ctx context.Context, ev store.OutboxEvent) {
	var payload outboxPayload
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		d.fail(ctx, ev, fmt.Errorf("payload: %w", err))
		return
	}

	var sendErr error
	switch ev.Channel {
	case "whatsapp", "sms":
		text, err := textFor(payload)
		if err != nil {
			sendErr = err
			break
		}
		sendErr = d.messages.Send(ctx, ev.Destination, text)
	case "push":
		title, body := pushCopy(payload)
		sendErr = d.push.Push(ctx, ev.Destination, title, body, ev.ID.String())
	default:
		sendErr = fmt.Errorf("unknown channel %q", ev.Channel)
	}
	if sendErr != nil {
		d.fail(ctx, ev, sendErr)
		return
	}
	if err := d.store.MarkOutboxSent(ctx, ev.ID); err != nil {
		d.log.Error("dispatch: mark sent", slog.String("event_id", ev.ID.String()), slog.String("error", err.Error()))
		return
	}
	d.log.Info("dispatch: delivered",
		slog.String("event_id", ev.ID.String()), slog.String("channel", ev.Channel))
}

// fail retries on the next tick or dead-letters once attempts run out —
// visible, never silent (the outbox SLO alerts on failed with age).
func (d *Dispatcher) fail(ctx context.Context, ev store.OutboxEvent, cause error) {
	lastErr := cause.Error()
	if int(ev.Attempts) < dispatchMaxTries {
		if err := d.store.RetryOutboxEvent(ctx, store.RetryOutboxEventParams{ID: ev.ID, LastError: &lastErr}); err != nil {
			d.log.Error("dispatch: retry", slog.String("event_id", ev.ID.String()), slog.String("error", err.Error()))
			return
		}
		d.log.Warn("dispatch: scheduled for retry",
			slog.String("event_id", ev.ID.String()), slog.Int("attempt", int(ev.Attempts)),
			slog.String("error", cause.Error()))
		return
	}
	if err := d.store.FailOutboxEvent(ctx, store.FailOutboxEventParams{ID: ev.ID, LastError: &lastErr}); err != nil {
		d.log.Error("dispatch: dead-letter", slog.String("event_id", ev.ID.String()), slog.String("error", err.Error()))
		return
	}
	d.log.Error("dispatch: event failed permanently",
		slog.String("event_id", ev.ID.String()), slog.String("channel", ev.Channel),
		slog.String("error", cause.Error()))
}

// textFor composes the WhatsApp/SMS text of an event (the R-08 alert copy
// lives here: who did not confirm, since when, last known signal, link).
// Unknown events are an error so a payload change fails loud, not silent.
func textFor(p outboxPayload) (string, error) {
	switch p.Event {
	case "escalation.notify":
		if p.Kind == KindReminderUrgent {
			return fmt.Sprintf("⚠️ %s: aún no confirmas hoy. Tus contactos serán alertados pronto.", p.UserName), nil
		}
		return fmt.Sprintf("👋 Hola %s, ¿todo bien? Confirma que estás OK.", p.UserName), nil
	case "incident.alert":
		link := ""
		if p.Token != "" && p.BaseURL != "" {
			link = fmt.Sprintf(" Responde aquí: %s/alert/%s", p.BaseURL, p.Token)
		}
		last := "sin señal registrada"
		if s, ok := p.LastSignal.(string); ok && s != "" {
			last = s
		}
		return fmt.Sprintf("⚠️ %s no ha confirmado desde %s. Última señal conocida: %s. ¿Puedes comprobarlo?%s",
			p.UserName, p.OpenedAt, last, link), nil
	case "incident.claimed":
		return fmt.Sprintf("✅ %s está yendo a comprobar cómo está %s.", p.ClaimerName, p.UserName), nil
	case "incident.all_clear":
		return fmt.Sprintf("✅ Todo en orden: %s confirmó que está bien.", p.UserName), nil
	default:
		return "", fmt.Errorf("unknown event %q", p.Event)
	}
}

// pushCopy is the title/body pair for push notifications.
func pushCopy(p outboxPayload) (string, string) {
	if p.Event == "escalation.notify" {
		if p.Kind == KindReminderUrgent {
			return "Tranqui · recordatorio urgente", "Confirma que estás bien: tus contactos serán alertados pronto."
		}
		return "Tranqui · hora del check-in", "¿Todo bien? Confirma que estás OK."
	}
	// Incident notices reach the circle as WhatsApp/SMS with the link; the
	// user's own device gets the state through the app (GET /v1/incidents).
	return "Tranqui · alerta familiar", "Abre la app para ver el estado del incidente."
}
