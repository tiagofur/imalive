package escalation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/tiagofur/imalive/backend/internal/escalation/store"
)

// Incident lifecycle audit kinds (append-only evidence).
const (
	AuditIncidentOpened         = "incident.opened"
	AuditIncidentCircleNotified = "incident.circle_notified"
	AuditIncidentClosed         = "incident.closed"
	// AuditLadderSkipped marks a rung that completed without notifying
	// because there was nobody to notify or nobody left to alert.
	AuditLadderSkipped = "ladder.skipped_no_contacts"
)

// Incident states (incidents.state CHECK, migration 000006).
const (
	IncidentContact1Notified = "contact1_notified"
	IncidentReclaimed        = "reclaimed"
	IncidentCircleNotified   = "circle_notified"
	IncidentClosedResolved   = "closed_resolved"
	IncidentClosedFalseAlarm = "closed_false_alarm"
)

// openIncidentForJob creates the incident of the rung inside the caller's
// transaction (the machine starts at Contacto1Notificado: the incident is
// born with its first notification), freezing the last known life signal.
// A race with an already-open incident reuses it: the partial unique index
// rejects the second open and the loser keeps alerting the same incident.
func openIncidentForJob(ctx context.Context, q *store.Queries, job store.EscalationJob) (store.Incident, error) {
	lastSignal, err := q.GetLastSignalAt(ctx, job.UserID)
	if err != nil {
		return store.Incident{}, fmt.Errorf("last signal: %w", err)
	}
	inc, err := q.CreateIncident(ctx, store.CreateIncidentParams{
		ID: uuid.Must(uuid.NewV7()), UserID: job.UserID,
		CycleDate: job.CycleDate, LastSignalAt: lastSignal,
	})
	if err == nil {
		return inc, nil
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return store.Incident{}, fmt.Errorf("create incident: %w", err)
	}
	open, gerr := q.GetOpenIncidentByUser(ctx, job.UserID)
	if gerr != nil {
		return store.Incident{}, fmt.Errorf("reuse open incident: %w", gerr)
	}
	return open, nil
}

// incidentAlertEvents builds the actionable alert events (R-08) for the given
// destinations: the payload carries who did not confirm, since when, the last
// known signal and a signed per-contact link for "Voy yo / No puedo". The
// dispatcher of #57 composes the text from these fields.
func (e *Engine) incidentAlertEvents(ctx context.Context, q *store.Queries, job store.EscalationJob, inc store.Incident, contacts []store.ListContactsByUserRow) ([]store.InsertOutboxEventParams, error) {
	user, err := q.GetUserForEscalation(ctx, job.UserID)
	if err != nil {
		return nil, fmt.Errorf("user: %w", err)
	}
	var lastSignal any
	if inc.LastSignalAt.Valid {
		lastSignal = inc.LastSignalAt.Time.UTC().Format(time.RFC3339)
	}
	events := make([]store.InsertOutboxEventParams, 0, len(contacts))
	for _, c := range contacts {
		id, err := uuid.NewV7()
		if err != nil {
			return nil, fmt.Errorf("uuid v7: %w", err)
		}
		token := ""
		if e.signer != nil {
			if token, err = e.signer.Sign(inc.ID, c.ID); err != nil {
				return nil, fmt.Errorf("sign alert link: %w", err)
			}
		}
		payload, err := json.Marshal(map[string]any{
			"event":          "incident.alert",
			"incident_id":    inc.ID,
			"user_name":      user.DisplayName,
			"cycle_date":     inc.CycleDate.Time.Format("2006-01-02"),
			"opened_at":      inc.OpenedAt.Time.UTC().Format(time.RFC3339),
			"last_signal_at": lastSignal,
			"token":          token,
			"base_url":       e.baseURL,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
		events = append(events, store.InsertOutboxEventParams{
			ID: id, UserID: job.UserID, Channel: outboxChannel,
			Destination: c.Phone, Payload: payload,
			JobID:      pgUUID(job.ID),
			IncidentID: pgUUID(inc.ID),
		})
	}
	return events, nil
}

// executeIncidentRung runs the notify_contact / notify_circle rungs: the
// incident transitions and its notifications share one transaction (ADR-0004
// — the state changed, the alerts recorded and the audit trail land together
// or nothing does).
func (e *Engine) executeIncidentRung(ctx context.Context, job store.EscalationJob) {
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		e.failOrRetry(ctx, job, e.clock.Now(), fmt.Errorf("begin: %w", err))
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := e.store.WithTx(tx)

	var events []store.InsertOutboxEventParams
	var audit auditParams
	switch job.Kind {
	case KindNotifyContact:
		contacts, err := q.ListContactsByUser(ctx, job.UserID)
		if err != nil {
			e.failOrRetry(ctx, job, e.clock.Now(), fmt.Errorf("contacts: %w", err))
			return
		}
		if len(contacts) == 0 {
			// Nobody to alert: the incident is born with its first
			// notification, and there is none — the rung completes.
			audit = auditParams{
				UserID: job.UserID, CycleDate: job.CycleDate,
				Kind: AuditLadderSkipped, JobID: &job.ID, Data: map[string]any{"kind": job.Kind},
			}
			break
		}
		inc, err := openIncidentForJob(ctx, q, job)
		if err != nil {
			e.failOrRetry(ctx, job, e.clock.Now(), err)
			return
		}
		// "Contacto 1" = the first remaining contact (lowest position).
		events, err = e.incidentAlertEvents(ctx, q, job, inc, contacts[:1])
		if err != nil {
			e.failOrRetry(ctx, job, e.clock.Now(), err)
			return
		}
		audit = auditParams{
			UserID: job.UserID, CycleDate: job.CycleDate,
			Kind: AuditIncidentOpened, JobID: &job.ID, IncidentID: &inc.ID,
			Data: map[string]any{"kind": job.Kind, "contact": contacts[0].DisplayName},
		}

	case KindNotifyCircle:
		inc, err := q.GetOpenIncidentByUser(ctx, job.UserID)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// No live incident (already closed or never opened): the rung
			// completes without notifying.
			audit = auditParams{
				UserID: job.UserID, CycleDate: job.CycleDate,
				Kind: AuditLadderSkipped, JobID: &job.ID, Data: map[string]any{"kind": job.Kind},
			}
		case err != nil:
			e.failOrRetry(ctx, job, e.clock.Now(), fmt.Errorf("open incident: %w", err))
			return
		case inc.State == IncidentReclaimed:
			// A claimant took it: the circle already heard "X está yendo".
			audit = auditParams{
				UserID: job.UserID, CycleDate: job.CycleDate,
				Kind: AuditLadderSkipped, JobID: &job.ID, IncidentID: &inc.ID,
				Data: map[string]any{"kind": job.Kind, "reason": "reclaimed"},
			}
		default:
			n, terr := q.CircleNotifiedIncident(ctx, inc.ID)
			if terr != nil {
				e.failOrRetry(ctx, job, e.clock.Now(), fmt.Errorf("transition: %w", terr))
				return
			}
			if n == 1 {
				audit = auditParams{
					UserID: job.UserID, CycleDate: job.CycleDate,
					Kind: AuditIncidentCircleNotified, JobID: &job.ID, IncidentID: &inc.ID,
					Data: map[string]any{},
				}
			}
			contacts, cerr := q.ListContactsByUser(ctx, job.UserID)
			if cerr != nil {
				e.failOrRetry(ctx, job, e.clock.Now(), fmt.Errorf("contacts: %w", cerr))
				return
			}
			events, cerr = e.incidentAlertEvents(ctx, q, job, inc, contacts)
			if cerr != nil {
				e.failOrRetry(ctx, job, e.clock.Now(), cerr)
				return
			}
		}

	default:
		e.failOrRetry(ctx, job, e.clock.Now(), fmt.Errorf("unknown incident rung %q", job.Kind))
		return
	}

	destinations := make([]string, 0, len(events))
	for _, ev := range events {
		if err := q.InsertOutboxEvent(ctx, ev); err != nil {
			e.failOrRetry(ctx, job, e.clock.Now(), fmt.Errorf("outbox insert: %w", err))
			return
		}
		destinations = append(destinations, ev.Destination)
	}
	if len(destinations) > 0 || audit.Kind == AuditLadderSkipped {
		if audit.Data == nil {
			audit.Data = map[string]any{}
		}
		audit.Data["destinations"] = destinations
		if err := insertAudit(ctx, q, audit); err != nil {
			e.failOrRetry(ctx, job, e.clock.Now(), fmt.Errorf("audit: %w", err))
			return
		}
	}
	if err := q.CompleteJob(ctx, job.ID); err != nil {
		e.failOrRetry(ctx, job, e.clock.Now(), fmt.Errorf("complete: %w", err))
		return
	}
	if err := tx.Commit(ctx); err != nil {
		e.failOrRetry(ctx, job, e.clock.Now(), fmt.Errorf("commit: %w", err))
		return
	}
	e.log.Info("escalation: incident rung executed",
		slog.String("job_id", job.ID.String()), slog.String("kind", job.Kind),
		slog.Int("destinations", len(destinations)))
}

// closeIncidentForSignal closes the user's open incident inside the caller's
// transaction (the life signal): resolved when a contact claimed it, false
// alarm otherwise — and the circle always hears that everything is fine
// (motor-escalada.md: the late signal never leaves the circle wondering).
func (e *Engine) closeIncidentForSignal(ctx context.Context, q *store.Queries, userID uuid.UUID) error {
	inc, err := q.GetOpenIncidentByUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // nothing open: nothing to close
	}
	if err != nil {
		return fmt.Errorf("open incident: %w", err)
	}
	reason := IncidentClosedFalseAlarm
	if inc.ClaimedBy.Valid {
		reason = IncidentClosedResolved
	}
	if n, err := q.CloseIncident(ctx, store.CloseIncidentParams{ID: inc.ID, State: reason}); err != nil {
		return fmt.Errorf("close: %w", err)
	} else if n == 0 {
		return nil // a parallel signal already closed it
	}

	user, err := q.GetUserForEscalation(ctx, userID)
	if err != nil {
		return fmt.Errorf("user: %w", err)
	}
	contacts, err := q.ListContactsByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("contacts: %w", err)
	}
	for _, c := range contacts {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("uuid v7: %w", err)
		}
		payload, err := json.Marshal(map[string]any{
			"event":       "incident.all_clear",
			"incident_id": inc.ID,
			"user_name":   user.DisplayName,
			"reason":      reason,
		})
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
		if err := q.InsertOutboxEvent(ctx, store.InsertOutboxEventParams{
			ID: id, UserID: userID, Channel: outboxChannel,
			Destination: c.Phone, Payload: payload,
			IncidentID: pgUUID(inc.ID),
		}); err != nil {
			return fmt.Errorf("outbox insert: %w", err)
		}
	}
	return insertAudit(ctx, q, auditParams{
		UserID: userID, CycleDate: inc.CycleDate, Kind: AuditIncidentClosed,
		IncidentID: &inc.ID,
		Data:       map[string]any{"reason": reason, "notified": len(contacts)},
	})
}
