package escalation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tiagofur/imalive/backend/internal/escalation/store"
)

// Worker tuning (ADR-0004: correctness over throughput at beta scale).
const (
	// claimBatch caps how many due jobs one tick claims: SKIP LOCKED keeps
	// parallel workers from double-claiming, the lease bounds their hold.
	claimBatch = 10
	// leaseDuration is how long a claimed job may stay 'running' before
	// another worker reclaims it: a crashed worker cannot lose a job.
	leaseDuration = 5 * time.Minute
	// maxBackoff caps the exponential retry delay (dead-letter still bound).
	maxBackoff = time.Hour
	// outboxChannel is the MVP delivery channel of the ladder (WhatsApp-first
	// product, R-05/R-06); the dispatcher of #57 maps it to Twilio/FCM.
	outboxChannel = "whatsapp"
)

// Audit kinds of the execution path (append-only evidence, spec §Auditoría).
const (
	AuditLadderNotified = "ladder.notified"        // a rung fired: whom, what, when
	AuditJobFailed      = "job.failed"             // dead-letter after max attempts
	AuditLadderFrozen   = "ladder.frozen_by_pause" // claimed job dropped: pause landed
)

// backoff returns the retry delay after the Nth attempt (attempts is >= 1
// once claimed): 1, 2, 4, 8... minutes, capped. Exponential per ADR-0004.
func backoff(attempts int16) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	shift := uint(attempts - 1)
	if shift > 20 {
		return maxBackoff
	}
	d := (1 << shift) * time.Minute
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}

// claimAndExecute claims the due jobs (SKIP LOCKED + lease) and executes each
// one: resolve destinations → write outbox events + complete job + audit in a
// single transaction. A failed execution retries with backoff and dead-letters
// to 'failed' after max attempts — visible, never silent (ADR-0004).
func (e *Engine) claimAndExecute(ctx context.Context) {
	now := e.clock.Now()
	jobs, err := e.store.ClaimDueJobs(ctx, store.ClaimDueJobsParams{
		LockedUntil: pgTz(now.Add(leaseDuration)),
		RunAt:       pgTz(now),
		Limit:       claimBatch,
	})
	if err != nil {
		e.log.Error("escalation: claim jobs", slog.String("error", err.Error()))
		return
	}
	for _, job := range jobs {
		e.execute(ctx, job)
	}
}

// execute runs one claimed job. The pause check is defensive (R-11): the
// pause can land between the claim and this line — nothing notifies while
// paused, whatever the race.
func (e *Engine) execute(ctx context.Context, job store.EscalationJob) {
	now := e.clock.Now()
	until, err := e.store.GetUserPauseUntil(ctx, job.UserID)
	switch {
	case err != nil:
		e.failOrRetry(ctx, job, now, fmt.Errorf("read pause: %w", err))
		return
	case until.Valid && until.Time.After(now):
		if err := e.freeze(ctx, job); err != nil {
			e.log.Error("escalation: freeze job",
				slog.String("job_id", job.ID.String()), slog.String("error", err.Error()))
		}
		return
	}

	switch job.Kind {
	case KindNotifyContact, KindNotifyCircle:
		// The incident rungs own their transaction (the incident state
		// machine changes with them).
		e.executeIncidentRung(ctx, job)
	case KindSosAlert:
		// The manual emergency: bypasses the ladder, straight to the circle.
		e.executeSosRung(ctx, job)
	default:
		e.executeReminderRung(ctx, job, now)
	}
}

// executeReminderRung runs the plain notification rungs (reminders): resolve
// destinations, then outbox + done + audit in one transaction.
func (e *Engine) executeReminderRung(ctx context.Context, job store.EscalationJob, now time.Time) {
	events, err := e.resolve(ctx, job)
	if err != nil {
		e.failOrRetry(ctx, job, now, fmt.Errorf("resolve %s: %w", job.Kind, err))
		return
	}
	if err := e.commitExecution(ctx, job, events); err != nil {
		e.failOrRetry(ctx, job, now, fmt.Errorf("commit %s: %w", job.Kind, err))
	}
}

// freeze cancels a claimed job because the user's pause went live mid-flight:
// the notification never goes out, and the audit trail records why.
func (e *Engine) freeze(ctx context.Context, job store.EscalationJob) error {
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := e.store.WithTx(tx)
	if err := q.CancelJob(ctx, job.ID); err != nil {
		return fmt.Errorf("cancel: %w", err)
	}
	if err := insertAudit(ctx, q, auditParams{
		UserID: job.UserID, CycleDate: job.CycleDate, Kind: AuditLadderFrozen,
		JobID: &job.ID,
		Data:  map[string]any{"kind": job.Kind},
	}); err != nil {
		return fmt.Errorf("audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	e.log.Info("escalation: job frozen by pause",
		slog.String("job_id", job.ID.String()), slog.String("kind", job.Kind))
	return nil
}

// commitExecution makes the job's effects atomic (the outbox invariant,
// ADR-0004): the state changed to done, the notifications recorded in the
// outbox and the audit row land together — a rollback loses all of them, a
// commit guarantees all of them.
func (e *Engine) commitExecution(ctx context.Context, job store.EscalationJob, events []store.InsertOutboxEventParams) error {
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := e.store.WithTx(tx)
	destinations := make([]string, 0, len(events))
	for _, ev := range events {
		if err := q.InsertOutboxEvent(ctx, ev); err != nil {
			return fmt.Errorf("outbox insert: %w", err)
		}
		destinations = append(destinations, ev.Destination)
	}
	if err := q.CompleteJob(ctx, job.ID); err != nil {
		return fmt.Errorf("complete: %w", err)
	}
	if err := insertAudit(ctx, q, auditParams{
		UserID: job.UserID, CycleDate: job.CycleDate, Kind: AuditLadderNotified,
		JobID: &job.ID,
		Data: map[string]any{
			"kind":         job.Kind,
			"destinations": destinations,
			"attempts":     job.Attempts,
		},
	}); err != nil {
		return fmt.Errorf("audit: %w", err)
	}
	return tx.Commit(ctx)
}

// failOrRetry moves a failed job to a backoff retry or, once attempts are
// exhausted, to the 'failed' dead-letter with an audit row: the SLO alerts on
// it, silence is never an option (ADR-0004).
func (e *Engine) failOrRetry(ctx context.Context, job store.EscalationJob, now time.Time, cause error) {
	if job.Attempts < job.MaxAttempts {
		next := now.Add(backoff(job.Attempts))
		lastErr := cause.Error()
		if err := e.store.RetryJob(ctx, store.RetryJobParams{
			ID: job.ID, RunAt: pgTz(next), LastError: &lastErr,
		}); err != nil {
			e.log.Error("escalation: schedule retry",
				slog.String("job_id", job.ID.String()), slog.String("error", err.Error()))
			return
		}
		e.log.Warn("escalation: job scheduled for retry",
			slog.String("job_id", job.ID.String()), slog.String("kind", job.Kind),
			slog.Int("attempt", int(job.Attempts)), slog.String("error", cause.Error()))
		return
	}

	lastErr := cause.Error()
	if err := e.store.FailJob(ctx, store.FailJobParams{ID: job.ID, LastError: &lastErr}); err != nil {
		e.log.Error("escalation: dead-letter job",
			slog.String("job_id", job.ID.String()), slog.String("error", err.Error()))
		return
	}
	if err := insertAudit(ctx, e.store, auditParams{
		UserID: job.UserID, CycleDate: job.CycleDate, Kind: AuditJobFailed,
		JobID: &job.ID,
		Data: map[string]any{
			"kind":     job.Kind,
			"error":    cause.Error(),
			"attempts": job.Attempts,
		},
	}); err != nil {
		e.log.Error("escalation: audit failed job",
			slog.String("job_id", job.ID.String()), slog.String("error", err.Error()))
	}
	e.log.Error("escalation: job failed permanently",
		slog.String("job_id", job.ID.String()), slog.String("kind", job.Kind),
		slog.Int("attempts", int(job.Attempts)), slog.String("error", cause.Error()))
}

// pgUUID converts a domain uuid to the nullable store type.
func pgUUID(u uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: u, Valid: true} }

// pgTz converts a domain instant to the nullable store timestamp.
func pgTz(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

// resolveNotifications is the production resolver: it maps a job kind to its
// outbox events (destinations + payload). The dispatcher of #57 reads the
// outbox; nothing here sends anything. Reminders reach BOTH channels: the
// WhatsApp text and the device push when a token is registered (R-05).
func (e *Engine) resolveNotifications(ctx context.Context, job store.EscalationJob) ([]store.InsertOutboxEventParams, error) {
	user, err := e.store.GetUserForEscalation(ctx, job.UserID)
	if err != nil {
		return nil, fmt.Errorf("user: %w", err)
	}
	cycle := job.CycleDate.Time.Format("2006-01-02")
	var dests []string
	var pushToken string
	switch job.Kind {
	case KindReminderSoft, KindReminderUrgent:
		dests = []string{user.Phone}
		if user.PushToken != nil {
			pushToken = *user.PushToken
		}
	case KindNotifyContact:
		contacts, err := e.store.ListContactsByUser(ctx, job.UserID)
		if err != nil {
			return nil, fmt.Errorf("contacts: %w", err)
		}
		// "Contacto 1" = the first remaining contact (lowest position): the
		// circle CRUD does not renumber on delete, so position 1 may be gone.
		if len(contacts) > 0 {
			dests = []string{contacts[0].Phone}
		}
	case KindNotifyCircle:
		contacts, err := e.store.ListContactsByUser(ctx, job.UserID)
		if err != nil {
			return nil, fmt.Errorf("contacts: %w", err)
		}
		dests = make([]string, 0, len(contacts))
		for _, c := range contacts {
			dests = append(dests, c.Phone)
		}
	default:
		return nil, fmt.Errorf("unknown job kind %q", job.Kind)
	}

	events := make([]store.InsertOutboxEventParams, 0, len(dests)+1)
	for _, dest := range dests {
		id, err := uuid.NewV7()
		if err != nil {
			return nil, fmt.Errorf("uuid v7: %w", err)
		}
		payload, err := json.Marshal(map[string]any{
			"event":      "escalation.notify",
			"kind":       job.Kind,
			"user_name":  user.DisplayName,
			"cycle_date": cycle,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
		events = append(events, store.InsertOutboxEventParams{
			ID: id, UserID: job.UserID, Channel: outboxChannel,
			Destination: dest, Payload: payload,
			JobID: pgUUID(job.ID),
		})
	}
	if pushToken != "" {
		id, err := uuid.NewV7()
		if err != nil {
			return nil, fmt.Errorf("uuid v7: %w", err)
		}
		payload, err := json.Marshal(map[string]any{
			"event":      "escalation.notify",
			"kind":       job.Kind,
			"user_name":  user.DisplayName,
			"cycle_date": cycle,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
		events = append(events, store.InsertOutboxEventParams{
			ID: id, UserID: job.UserID, Channel: "push",
			Destination: pushToken, Payload: payload,
			JobID: pgUUID(job.ID),
		})
	}
	return events, nil
}
