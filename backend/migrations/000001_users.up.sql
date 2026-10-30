-- 000001_users.up.sql
-- users: la identidad del check-in (R-01/R-02). PII mínima (producto.md §H):
--   phone (identidad del producto, único global), display_name (campo `name`
--   del contrato; mapeo documentado en el store), timezone + checkin_time +
--   escalation_window_minutes (configuración del loop diario).
-- Cero datos de ubicación o historial. Cero credenciales: los secretos de
-- sesión viven en Redis (issue #39), no en esta tabla.
-- Política de down: herramienta de dev/CI; en producción las migraciones son
-- solo hacia adelante.

CREATE TABLE users (
    id                        uuid PRIMARY KEY,
    phone                     text        NOT NULL UNIQUE
        CHECK (phone ~ '^\+[1-9][0-9]{7,14}$'),
    display_name              text        NOT NULL
        CHECK (char_length(display_name) BETWEEN 1 AND 80),
    timezone                  text        NOT NULL,
    checkin_time              time        NOT NULL DEFAULT '21:00',
    escalation_window_minutes smallint    NOT NULL DEFAULT 60
        CHECK (escalation_window_minutes BETWEEN 0 AND 1440),
    created_at                timestamptz NOT NULL DEFAULT now(),
    updated_at                timestamptz NOT NULL DEFAULT now()
);
