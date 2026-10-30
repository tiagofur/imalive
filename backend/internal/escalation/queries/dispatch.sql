-- El dispatcher de salida (ADR-0004): reclama eventos pendientes del outbox
-- con el mismo patrón de la cola de jobs (SKIP LOCKED + lease), envía fuera
-- de la transacción (red) y marca sent / pending-retry / failed. El lease se
-- anota en dispatched_at: un worker que muere con eventos 'sending' los
-- libera al expirar (at-least-once; el dedupe por event_id lo absorben los
-- proveedores). $1 = ahora, $2 = horizonte del lease, $3 = lote.

-- name: ClaimDueOutboxEvents :many
UPDATE outbox_event SET
    state = 'sending',
    dispatched_at = now(),
    attempts = attempts + 1
WHERE id IN (
    SELECT e.id FROM outbox_event e
    WHERE (e.state = 'pending' AND e.created_at <= $1)
       OR (e.state = 'sending' AND e.dispatched_at < $2)
    ORDER BY e.created_at
    FOR UPDATE SKIP LOCKED
    LIMIT $3
)
RETURNING *;

-- name: MarkOutboxSent :exec
UPDATE outbox_event SET
    state = 'sent',
    dispatched_at = now(),
    last_error = NULL
WHERE id = $1;

-- name: RetryOutboxEvent :exec
-- Sin backoff dedicado: el reintento es el siguiente tick (30 s); el SLO de
-- edad del outbox (< 5 min) prefiere reintentos rápidos.
UPDATE outbox_event SET
    state = 'pending',
    last_error = $2
WHERE id = $1;

-- name: FailOutboxEvent :exec
-- Dead-letter visible: agotados los intentos, la observabilidad alerta.
UPDATE outbox_event SET
    state = 'failed',
    last_error = $2
WHERE id = $1;
