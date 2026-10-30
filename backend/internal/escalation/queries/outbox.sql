-- Outbox transaccional (ADR-0004): el motor escribe aquí en la MISMA
-- transacción que el cambio de estado; el dispatcher de #57 solo lee.

-- name: InsertOutboxEvent :exec
-- Dedupe at-least-once por (job_id, destination): re-ejecutar un job no
-- duplica jamás la misma notificación. Una fila por llamada (el círculo son
-- ≤ 5 contactos): el motor las inserta dentro de su transacción.
INSERT INTO outbox_event (id, user_id, channel, destination, payload, job_id, incident_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (job_id, destination) DO NOTHING;
