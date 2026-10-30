-- 000003_checkins.up.sql
-- checkins: la señal diaria (R-04). Sin historial de ubicación ni notas:
--   solo el hecho del check-in. PII mínima: ninguna adicional al user_id.
-- cycle_date es la fecha del ciclo en la zona del usuario (no UTC):
--   en LATAM (UTC-3..UTC-8) una fecha UTC cambia de día a media tarde y
--   rompería UNIQUE por día alrededor del check-in de 21:00. El timestamp
--   sí es timestamptz UTC. Desvío documentado en el plan del issue #38.
-- UNIQUE (user_id, cycle_date): una señal por usuario y día.
-- Índice único parcial sobre idempotency_key: replay del header
--   Idempotency-Key de POST /v1/checkins (contrato v0.1.0).
-- Política de down: herramienta de dev/CI; en producción solo hacia adelante.

CREATE TABLE checkins (
    id              uuid PRIMARY KEY,
    user_id         uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    cycle_date      date        NOT NULL,
    checked_in_at   timestamptz NOT NULL,
    idempotency_key text        NULL CHECK (char_length(idempotency_key) <= 64),
    created_at      timestamptz NOT NULL DEFAULT now(),

    UNIQUE (user_id, cycle_date)
);

CREATE UNIQUE INDEX checkins_user_idempotency_key_idx
    ON checkins (user_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
