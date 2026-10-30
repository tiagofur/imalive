-- 000004_pause.up.sql
-- pause: modo pausa/viaje (R-11). PII mínima: solo el instante hasta el que
-- la escalera queda congelada (NULL = sin pausa). El motor consulta este
-- valor; la racha se congela mientras haya pausa activa.
-- Política de down: herramienta de dev/CI; en producción solo hacia adelante.

ALTER TABLE users ADD COLUMN pause_until timestamptz NULL;
