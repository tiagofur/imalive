package escalation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tiagofur/imalive/backend/internal/escalation/store"
)

// Audit event kinds (audit_events.kind, append-only). Names are stable
// evidence: they document what the engine told whom and when.
const (
	AuditCycleOpened = "cycle.opened"
)

// Engine drives the daily escalation loop (R-07): it opens the cycle when the
// check-in time passes without a signal, escalates the ladder on silence and
// cancels pending work when a life signal arrives (the signal wiring is the
// #54 signals slice). A tick never fails the caller: errors are logged with
// enough context to alert on, and the next tick self-heals (the sweeper is
// stateless by design).
type Engine struct {
	pool  *pgxpool.Pool
	store *store.Queries
	clock Clock
	log   *slog.Logger
}

// New wires the engine over a pool and an injectable clock.
func New(pool *pgxpool.Pool, clock Clock, log *slog.Logger) *Engine {
	return &Engine{pool: pool, store: store.New(pool), clock: clock, log: log}
}

// ProcessTick runs one worker tick (cmd/worker, every 30 s): open due cycles,
// then claim and execute due jobs (the #54 claim slice).
func (e *Engine) ProcessTick(ctx context.Context) {
	e.openDueCycles(ctx)
}

// openDueCycles sweeps the users and opens the daily cycle of everyone whose
// check-in time has passed today without a signal. Stateless on purpose: a
// worker that was down catches up on the next tick, and a re-open of an
// already-open cycle is a no-op at the database level.
func (e *Engine) openDueCycles(ctx context.Context) {
	users, err := e.store.ListSweepUsers(ctx)
	if err != nil {
		e.log.Error("escalation: sweep: list users", slog.String("error", err.Error()))
		return
	}
	now := e.clock.Now()
	for _, u := range users {
		if err := e.openCycleIfDue(ctx, u, now); err != nil {
			// One corrupt user must never stall the rest of the sweep; the
			// error is loud (error level + user id) for observability.
			e.log.Error("escalation: open cycle",
				slog.String("user_id", u.ID.String()), slog.String("error", err.Error()))
		}
	}
}

// openCycleIfDue opens the daily cycle of one user when due. Due means: the
// local check-in time already passed today, no signal of life for today's
// cycle date, no live ladder (one active cycle per user), and no pause (R-11:
// the ladder freezes while paused; when the pause expires the sweeper re-opens
// from the natural point).
func (e *Engine) openCycleIfDue(ctx context.Context, u store.ListSweepUsersRow, now time.Time) error {
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		// The profile validates IANA names at write time; a corrupt row is a
		// data bug that fails loud per user without blocking the sweep.
		return fmt.Errorf("timezone %q: %w", u.Timezone, err)
	}
	local := now.In(loc)
	slot := time.Date(local.Year(), local.Month(), local.Day(),
		int(u.CheckinTime.Microseconds/1_000_000)/3600,
		(int(u.CheckinTime.Microseconds/1_000_000)/60)%60, 0, 0, loc)
	if local.Before(slot) {
		return nil // today's cycle is not due yet
	}
	if u.PauseUntil.Valid && u.PauseUntil.Time.After(now) {
		return nil // paused (R-11): the ladder freezes
	}
	cycleDate := pgtype.Date{Time: local, Valid: true}

	checkin, err := e.store.HasCheckinForCycle(ctx, store.HasCheckinForCycleParams{
		UserID: u.ID, CycleDate: cycleDate,
	})
	if err != nil {
		return fmt.Errorf("check for signal: %w", err)
	}
	if checkin {
		return nil // already confirmed: nothing to escalate
	}
	active, err := e.store.CountActiveJobsByUser(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("count active jobs: %w", err)
	}
	if active > 0 {
		return nil // a live ladder exists: one active cycle per user
	}
	return e.openCycle(ctx, u, cycleDate, slot, local)
}

// openCycle schedules the ladder of the cycle and records the audit trail in
// a single transaction. The UNIQUE (user_id, cycle_date, kind) makes a race
// between two workers a no-op, and re-arms only cancelled rows (a pause that
// expired): a done cycle never re-notifies.
func (e *Engine) openCycle(ctx context.Context, u store.ListSweepUsersRow, cycleDate pgtype.Date, slot, local time.Time) error {
	grace := time.Duration(u.EscalationWindowMinutes) * time.Minute
	steps := Schedule(slot, grace)

	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := e.store.WithTx(tx)

	ladder := make([]map[string]any, 0, len(steps))
	for _, step := range steps {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("uuid v7: %w", err)
		}
		if err := q.CreateLadderJob(ctx, store.CreateLadderJobParams{
			ID: id, UserID: u.ID, CycleDate: cycleDate, Kind: step.Kind,
			RunAt: pgtype.Timestamptz{Time: step.RunAt.UTC(), Valid: true},
		}); err != nil {
			return fmt.Errorf("create job %s: %w", step.Kind, err)
		}
		ladder = append(ladder, map[string]any{"kind": step.Kind, "run_at": step.RunAt.UTC().Format(time.RFC3339)})
	}
	if err := insertAudit(ctx, q, auditParams{
		UserID: u.ID, CycleDate: cycleDate, Kind: AuditCycleOpened,
		Data: map[string]any{
			"checkin_time":  slot.Format("15:04"),
			"grace_minutes": u.EscalationWindowMinutes,
			"ladder":        ladder,
			"local_date":    local.Format("2006-01-02"),
		},
	}); err != nil {
		return fmt.Errorf("audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	e.log.Info("escalation: cycle opened",
		slog.String("user_id", u.ID.String()),
		slog.String("cycle_date", cycleDate.Time.Format("2006-01-02")),
		slog.Int("steps", len(steps)),
	)
	return nil
}

// auditParams is the engine-facing audit event: the store takes nullable
// pgtype columns, the domain takes plain optionals.
type auditParams struct {
	UserID     uuid.UUID
	CycleDate  pgtype.Date
	Kind       string
	JobID      *uuid.UUID
	IncidentID *uuid.UUID
	Data       map[string]any
}

func insertAudit(ctx context.Context, q *store.Queries, p auditParams) error {
	data, err := json.Marshal(p.Data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}
	var jobID, incidentID pgtype.UUID
	if p.JobID != nil {
		jobID = pgtype.UUID{Bytes: *p.JobID, Valid: true}
	}
	if p.IncidentID != nil {
		incidentID = pgtype.UUID{Bytes: *p.IncidentID, Valid: true}
	}
	return q.InsertAuditEvent(ctx, store.InsertAuditEventParams{
		UserID: p.UserID, CycleDate: p.CycleDate, Kind: p.Kind,
		JobID: jobID, IncidentID: incidentID, Data: data,
	})
}
