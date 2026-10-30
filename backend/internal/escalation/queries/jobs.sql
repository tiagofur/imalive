-- Lógica de planificación del motor (R-07): las queries de scheduling reciben
-- `now` como parámetro del reloj inyectable (Clock); now() de BD solo en
-- timestamps de metadata. Inserts de fila única (la escalera son 4 kinds fijos
-- y el círculo son ≤ 5 contactos): sin unnest, el store queda tipado simple.

-- name: CreateLadderJob :exec
-- Apertura idempotente del ciclo: la UNIQUE (user_id, cycle_date, kind) hace
-- la carrera entre workers inofensiva. Re-armar SOLO filas 'cancelled' (una
-- pausa canceló la escalera y al expirar se re-abre desde el punto natural);
-- un ciclo 'done' jamás se re-notifica.
INSERT INTO escalation_jobs (id, user_id, cycle_date, kind, run_at)
VALUES ($1, $2, $3, $4, $5)
-- El predicado debe casar con el índice parcial (000007): el ladder es único
-- por ciclo; sos_alert queda exento (un SOS puede repetirse el mismo día).
ON CONFLICT (user_id, cycle_date, kind) WHERE kind <> 'sos_alert' DO UPDATE
SET state = 'pending',
    run_at = EXCLUDED.run_at,
    attempts = 0,
    max_attempts = 3,
    locked_until = NULL,
    last_error = NULL,
    updated_at = now()
WHERE escalation_jobs.state = 'cancelled';

-- name: ListSweepUsers :many
-- Escaneo del sweeper (apertura de ciclos). Correcto a escala beta (10
-- familias): revisitar con un índice de "usuarios sin señal hoy" si crece.
SELECT id, phone, display_name, timezone, checkin_time, escalation_window_minutes, pause_until
FROM users;

-- name: HasCheckinForCycle :one
SELECT EXISTS (
    SELECT 1 FROM checkins
    WHERE user_id = $1 AND cycle_date = $2
) AS has_checkin;

-- name: CountActiveJobsByUser :one
-- Un solo ciclo activo por usuario: no abrir uno nuevo mientras la escalera
-- anterior siga viva (pendiente o ejecutándose).
SELECT count(*) AS active FROM escalation_jobs
WHERE user_id = $1 AND state IN ('pending', 'running');

-- name: ClaimDueJobs :many
-- Corazón de la cola (ADR-0004): SKIP LOCKED serializa workers paralelos sin
-- duplicar; el lease (locked_until) reclama jobs 'running' de un worker muerto
-- (at-least-once). $1 = lease hasta, $2 = ahora, $3 = lote.
UPDATE escalation_jobs SET
    state = 'running',
    locked_until = $1,
    attempts = attempts + 1,
    updated_at = now()
WHERE id IN (
    SELECT j.id FROM escalation_jobs AS j
    WHERE (j.state = 'pending' AND j.run_at <= $2)
       OR (j.state = 'running' AND j.locked_until < $2)
    ORDER BY j.run_at
    FOR UPDATE SKIP LOCKED
    LIMIT $3
)
RETURNING *;

-- name: CompleteJob :exec
UPDATE escalation_jobs SET
    state = 'done',
    locked_until = NULL,
    last_error = NULL,
    updated_at = now()
WHERE id = $1;

-- name: RetryJob :exec
-- Reintento con backoff: $2 = run_at calculado por el dominio.
UPDATE escalation_jobs SET
    state = 'pending',
    run_at = $2,
    locked_until = NULL,
    last_error = $3,
    updated_at = now()
WHERE id = $1;

-- name: FailJob :exec
-- Dead-letter visible: agotados los reintentos, el job queda 'failed' y la
-- observabilidad alerta (nunca silencio, ADR-0004).
UPDATE escalation_jobs SET
    state = 'failed',
    locked_until = NULL,
    last_error = $2,
    updated_at = now()
WHERE id = $1;

-- name: CancelJob :exec
UPDATE escalation_jobs SET
    state = 'cancelled',
    locked_until = NULL,
    updated_at = now()
WHERE id = $1;

-- name: CancelPendingJobsByUser :execrows
-- Señal de vida o pausa: los jobs pendientes del usuario se cancelan EN LA
-- MISMA transacción que registra la señal (motor-escalada.md, cancelación).
UPDATE escalation_jobs SET
    state = 'cancelled',
    locked_until = NULL,
    updated_at = now()
WHERE user_id = $1 AND state = 'pending';

-- name: GetUserPauseUntil :one
-- Lectura de la pausa (R-11) para el chequeo defensivo del executor.
SELECT pause_until FROM users
WHERE id = $1;

-- name: ListContactsByUser :many
-- Destinatarios de la escalada (R-03): contacto 1 = position 1, círculo = todos.
SELECT id, display_name, phone, position
FROM contacts
WHERE user_id = $1
ORDER BY position;

-- name: GetUserForEscalation :one
-- Datos mínimos del destinatario de los recordatorios y del payload (nombre
-- de la persona que no confirmó). El teléfono es el destino WhatsApp
-- (R-05/R-06) y push_token el destino push cuando existe.
SELECT id, phone, display_name, push_token FROM users
WHERE id = $1;

-- name: GetPendingSosJob :one
-- El armado es idempotente: un SOS ya en cuenta regresiva devuelve su run_at.
SELECT * FROM escalation_jobs
WHERE user_id = $1 AND kind = 'sos_alert' AND state = 'pending'
ORDER BY run_at
LIMIT 1;

-- name: CreateSosJob :one
INSERT INTO escalation_jobs (id, user_id, cycle_date, kind, run_at)
VALUES ($1, $2, $3, 'sos_alert', $4)
RETURNING *;

-- name: CancelPendingSosJobs :execrows
-- La cancelación dentro de la ventana: idempotente, auditada por el llamador.
UPDATE escalation_jobs SET
    state = 'cancelled',
    locked_until = NULL,
    updated_at = now()
WHERE user_id = $1 AND kind = 'sos_alert' AND state = 'pending';

-- name: SetUserPushToken :exec
-- Registro del token FCM (POST /v1/devices): re-registrar reemplaza.
UPDATE users SET push_token = $2, updated_at = now()
WHERE id = $1;

-- name: ClearUserPushToken :exec
-- Revocación (DELETE /v1/devices): idempotente.
UPDATE users SET push_token = NULL, updated_at = now()
WHERE id = $1;
