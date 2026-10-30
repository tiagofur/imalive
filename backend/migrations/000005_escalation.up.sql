-- 000005_escalation.up.sql
-- Motor de escalada (R-07, ADR-0004): cola de trabajo, outbox transaccional y
-- auditoría append-only. PII mínima: solo datos ya viventes en la BD (nombre y
-- teléfono del destino, clave de mensaje); cero ubicación, cero PII nueva.
--
-- escalation_jobs: la cola. UNIQUE (user_id, cycle_date, kind) hace idempotente
--   la apertura del ciclo ante carreras entre workers (el sweeper re-abre con
--   ON CONFLICT re-armando solo filas 'cancelled' — pausa → resume).
--   locked_until es el lease: un worker que muere con jobs 'running' los libera
--   al expirar (at-least-once con reintentos).
-- outbox_event: la salida notificable. Se inserta en la MISMA transacción que
--   el cambio de estado del motor (ADR-0004: jamás notificar un estado que no
--   quedó grabado, ni perder notificaciones de estados grabados).
--   UNIQUE parcial (job_id, destination): re-ejecutar un job no duplica jamás
--   la misma notificación (dedupe at-least-once por job). El dispatcher (#57)
--   dedupe además por event_id hacia los proveedores.
-- audit_events: la evidencia de la promesa "best effort" (spec §8). SIN FK en
--   user_id: la evidencia sobrevive al borrado de cuenta (F2). El trigger
--   bloquea UPDATE/DELETE a cualquier rol, incluido el dueño del esquema; el
--   rol restringido de producción llega con ADR-0006 (hosting).
-- incident_id: correlación de los incidentes de #55 (la máquina de estados del
--   incidente llega después; la columna espera, sin FK).
-- Las queries de planificación del motor reciben `now` como parámetro (reloj
-- inyectable): now() de BD solo en timestamps de metadata, nunca en lógica.
-- Política de down: herramienta de dev/CI; en producción solo hacia adelante.

CREATE TABLE escalation_jobs (
    id           uuid PRIMARY KEY,
    user_id      uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    cycle_date   date        NOT NULL,
    kind         text        NOT NULL
        CHECK (kind IN ('reminder_soft', 'reminder_urgent', 'notify_contact', 'notify_circle')),
    run_at       timestamptz NOT NULL,
    state        text        NOT NULL DEFAULT 'pending'
        CHECK (state IN ('pending', 'running', 'done', 'failed', 'cancelled')),
    attempts     smallint    NOT NULL DEFAULT 0,
    max_attempts smallint    NOT NULL DEFAULT 3,
    locked_until timestamptz NULL,
    last_error   text        NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),

    UNIQUE (user_id, cycle_date, kind)
);

-- El único acceso caliente del worker: los jobs vencidos, en orden de vencimiento.
CREATE INDEX escalation_jobs_due_idx
    ON escalation_jobs (run_at)
    WHERE state = 'pending';

CREATE TABLE outbox_event (
    id            uuid PRIMARY KEY,
    user_id       uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    channel       text        NOT NULL CHECK (channel IN ('whatsapp', 'sms', 'push')),
    destination   text        NOT NULL,
    payload       jsonb       NOT NULL,
    job_id        uuid        NULL REFERENCES escalation_jobs (id) ON DELETE SET NULL,
    incident_id   uuid        NULL,
    state         text        NOT NULL DEFAULT 'pending'
        CHECK (state IN ('pending', 'sent', 'failed')),
    attempts      smallint    NOT NULL DEFAULT 0,
    last_error    text        NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    dispatched_at timestamptz NULL,

    UNIQUE (job_id, destination)
);

-- El único acceso caliente del dispatcher (#57): la cola pendiente, en orden de edad.
CREATE INDEX outbox_event_pending_idx
    ON outbox_event (created_at)
    WHERE state = 'pending';

CREATE TABLE audit_events (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    user_id     uuid        NOT NULL,
    cycle_date  date        NULL,
    kind        text        NOT NULL,
    job_id      uuid        NULL,
    incident_id uuid        NULL,
    data        jsonb       NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX audit_events_user_occurred_idx
    ON audit_events (user_id, occurred_at);

-- Append-only real: ni UPDATE ni DELETE, para cualquier rol (el dueño del
-- esquema incluido). Nada de lógica de producto borra esta tabla.
CREATE FUNCTION audit_events_block_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_events is append-only (evidence table, R-07)';
END;
$$;

CREATE TRIGGER audit_events_append_only
    BEFORE UPDATE OR DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION audit_events_block_mutation();
