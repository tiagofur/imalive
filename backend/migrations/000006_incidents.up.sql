-- 000006_incidents.up.sql
-- incidents: la máquina de estados del incidente (R-08 + R-09), parte del
-- motor (motor-escalada.md §Máquina de estados del incidente). PII mínima:
-- referencias a filas existentes (user, contacto reclamante) y el hecho del
-- incidente; cero ubicación, cero contenido libre.
-- Un solo incidente abierto por usuario: índice único parcial sobre los
-- estados vivos (la alerta es una por ciclo; el cierre lo da la señal del
-- usuario o la escalada al círculo).
-- claimed_by es el ÚNICO reclamante ("Voy yo"): el primero gana y la
-- máquina de estados hace idempotente el resto (R-09: sin duplicar).
-- last_signal_at congela la "última señal de vida conocida" que la alerta
-- muestra (R-08) — se llena al abrir, nunca se actualiza después.
-- Política de down: herramienta de dev/CI; en producción solo hacia adelante.

CREATE TABLE incidents (
    id             uuid PRIMARY KEY,
    user_id        uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    cycle_date     date        NOT NULL,
    state          text        NOT NULL DEFAULT 'contact1_notified'
        CHECK (state IN ('contact1_notified', 'reclaimed', 'circle_notified',
                         'closed_resolved', 'closed_false_alarm')),
    claimed_by     uuid        NULL REFERENCES contacts (id) ON DELETE SET NULL,
    claimed_at     timestamptz NULL,
    last_signal_at timestamptz NULL,
    opened_at      timestamptz NOT NULL DEFAULT now(),
    closed_at      timestamptz NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

-- Un solo incidente abierto por usuario (estados vivos).
CREATE UNIQUE INDEX incidents_one_open_idx
    ON incidents (user_id)
    WHERE state IN ('contact1_notified', 'reclaimed', 'circle_notified');

-- La consulta del usuario (GET /v1/incidents/current, contrato v0.3.0).
CREATE INDEX incidents_user_recent_idx
    ON incidents (user_id, opened_at DESC);
