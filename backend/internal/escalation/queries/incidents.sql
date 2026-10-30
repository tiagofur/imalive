-- La máquina de estados del incidente (R-08 + R-09). Toda transición es una
-- UPDATE con guard de estado: los estados vivos sólo cambian desde los
-- estados válidos, y el nº de filas afectadas decide la idempotencia (el
-- perdedor de una carrera ve 0 filas, no un error). El motor de escalada es
-- el dueño: las transiciones corren DENTRO de su transacción (outbox + audit
-- + estado, todo o nada, ADR-0004).

-- name: CreateIncident :one
INSERT INTO incidents (id, user_id, cycle_date, last_signal_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetOpenIncidentByUser :one
SELECT * FROM incidents
WHERE user_id = $1
  AND state IN ('contact1_notified', 'reclaimed', 'circle_notified');

-- name: GetIncident :one
SELECT * FROM incidents
WHERE id = $1;

-- name: GetCurrentIncidentForUser :one
-- El más reciente del usuario (contrato GET /v1/incidents/current): abierto
-- o el último cerrado.
SELECT * FROM incidents
WHERE user_id = $1
ORDER BY opened_at DESC
LIMIT 1;

-- name: ClaimIncident :execrows
-- "Voy yo" (R-08): el PRIMER reclamante gana; desde 'reclaimed' o estados
-- cerrados la UPDATE afecta 0 filas (idempotencia del doble tap).
UPDATE incidents SET
    state = 'reclaimed',
    claimed_by = $2,
    claimed_at = now(),
    updated_at = now()
WHERE id = $1
  AND state IN ('contact1_notified', 'circle_notified');

-- name: CircleNotifiedIncident :execrows
-- "No puedo" o el rung del círculo: la alerta escala a todo el círculo.
UPDATE incidents SET
    state = 'circle_notified',
    updated_at = now()
WHERE id = $1
  AND state = 'contact1_notified';

-- name: CloseIncident :execrows
-- La señal de vida del usuario SIEMPRE cierra ($2 = closed_resolved cuando
-- hubo reclamante, closed_false_alarm si no); 0 filas si ya estaba cerrado.
UPDATE incidents SET
    state = $2,
    closed_at = now(),
    updated_at = now()
WHERE id = $1
  AND state IN ('contact1_notified', 'reclaimed', 'circle_notified');

-- name: GetLastSignalAt :one
-- La "última señal de vida conocida" de la alerta (R-08): NULL si el usuario
-- nunca confirmó.
SELECT max(checked_in_at)::timestamptz AS last_signal FROM checkins
WHERE user_id = $1;

-- name: GetContactByID :one
-- El nombre del reclamante para la alerta "X está yendo" (R-09).
SELECT id, display_name FROM contacts
WHERE id = $1;

-- name: CancelPendingJobsByUserAndKind :execrows
-- "No puedo": el rung del círculo ya no hace falta (la alerta sale ya) —
-- cancelar SOLO ese kind, sin tocar el resto de la escalera.
UPDATE escalation_jobs SET
    state = 'cancelled',
    locked_until = NULL,
    updated_at = now()
WHERE user_id = $1 AND kind = $2 AND state = 'pending';
