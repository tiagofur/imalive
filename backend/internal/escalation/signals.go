package escalation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tiagofur/imalive/backend/internal/escalation/store"
)

// Life-signal audit kinds (append-only evidence).
const (
	AuditSignalReceived = "signal.received" // a check-in arrived and cancelled the ladder
	AuditPauseActivated = "pause.activated" // the pause froze pending rungs
	AuditSosArmed       = "sos.armed"       // the countdown started (R-10)
	AuditSosCancelled   = "sos.cancelled"   // cancelled inside the window
)

// SignalReceivedTx records the life signal inside the CALLER's transaction
// (the check-in handler): every pending job of the user is cancelled and any
// open incident closes — resolved when a contact claimed it, false alarm
// otherwise — with an "all clear" to the circle, all atomically with the
// signal (motor-escalada.md §Cola de trabajo + §Máquina de estados del
// incidente). The signal itself is evidence even with nothing to cancel.
func (e *Engine) SignalReceivedTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, cycleDate pgtype.Date) error {
	q := e.store.WithTx(tx)
	n, err := q.CancelPendingJobsByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("cancel pending jobs: %w", err)
	}
	if err := e.closeIncidentForSignal(ctx, q, userID); err != nil {
		return fmt.Errorf("close incident: %w", err)
	}
	return insertAudit(ctx, q, auditParams{
		UserID: userID, CycleDate: cycleDate, Kind: AuditSignalReceived,
		Data: map[string]any{"cancelled_jobs": n},
	})
}

// PauseActivatedTx freezes the ladder inside the CALLER's transaction (the
// pause handler): pending jobs are cancelled atomically with the pause write
// (R-11). When the pause expires the sweeper re-opens the cycle naturally.
// Nothing to freeze (no live ladder): no audit row, the pause column itself
// is the record.
func (e *Engine) PauseActivatedTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	q := e.store.WithTx(tx)
	n, err := q.CancelPendingJobsByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("cancel pending jobs: %w", err)
	}
	if n == 0 {
		return nil
	}
	return insertAudit(ctx, q, auditParams{
		UserID: userID, Kind: AuditPauseActivated,
		Data: map[string]any{"cancelled_jobs": n},
	})
}

// SosArmTx starts the SOS countdown (R-10) inside the CALLER's transaction:
// a sos_alert job due at runAt (now+10s). Idempotent: an already-armed SOS
// returns its existing run_at instead of stacking a second one.
func (e *Engine) SosArmTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, cycleDate pgtype.Date, runAt time.Time) (time.Time, error) {
	q := e.store.WithTx(tx)
	if existing, err := q.GetPendingSosJob(ctx, userID); err == nil {
		return existing.RunAt.Time.UTC(), nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, fmt.Errorf("pending sos: %w", err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return time.Time{}, fmt.Errorf("uuid v7: %w", err)
	}
	created, err := q.CreateSosJob(ctx, store.CreateSosJobParams{
		ID: id, UserID: userID, CycleDate: cycleDate, RunAt: pgTz(runAt),
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("create sos job: %w", err)
	}
	if err := insertAudit(ctx, q, auditParams{
		UserID: userID, CycleDate: cycleDate, Kind: AuditSosArmed,
		JobID: &id, Data: map[string]any{"run_at": runAt.UTC().Format(time.RFC3339)},
	}); err != nil {
		return time.Time{}, fmt.Errorf("audit: %w", err)
	}
	return created.RunAt.Time.UTC(), nil
}

// SosCancelTx cancels the armed SOS inside the window, in the CALLER's
// transaction: idempotent (0 pending = nothing to do) and audited — the
// triage requires the evidence either way.
func (e *Engine) SosCancelTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (int64, error) {
	q := e.store.WithTx(tx)
	n, err := q.CancelPendingSosJobs(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("cancel sos jobs: %w", err)
	}
	if n == 0 {
		return 0, nil
	}
	if err := insertAudit(ctx, q, auditParams{
		UserID: userID, Kind: AuditSosCancelled,
		Data: map[string]any{"cancelled_jobs": n},
	}); err != nil {
		return 0, fmt.Errorf("audit: %w", err)
	}
	return n, nil
}
