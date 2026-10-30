-- Auditoría append-only (R-07): cada transición del motor deja evidencia aquí.
-- El trigger de la migración 000005 bloquea UPDATE/DELETE.

-- name: InsertAuditEvent :exec
INSERT INTO audit_events (user_id, cycle_date, kind, job_id, incident_id, data)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListAuditEventsByUser :many
SELECT id, occurred_at, user_id, cycle_date, kind, job_id, incident_id, data
FROM audit_events
WHERE user_id = $1
ORDER BY occurred_at, id;
