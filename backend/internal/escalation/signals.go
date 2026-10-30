package escalation

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Life-signal audit kinds (append-only evidence).
const (
	AuditSignalReceived = "signal.received" // a check-in arrived and cancelled the ladder
	AuditPauseActivated = "pause.activated" // the pause froze pending rungs
)

// SignalReceivedTx records the life signal inside the CALLER's transaction
// (the check-in handler): every pending job of the user is cancelled
// atomically with the signal (motor-escalada.md §Cola de trabajo — the
// cancellation must never race the ladder, nor outlive a rolled-back
// check-in). The signal itself is evidence even with nothing to cancel.
func (e *Engine) SignalReceivedTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, cycleDate pgtype.Date) error {
	q := e.store.WithTx(tx)
	n, err := q.CancelPendingJobsByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("cancel pending jobs: %w", err)
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
