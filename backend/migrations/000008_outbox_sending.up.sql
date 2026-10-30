-- 000008_outbox_sending.up.sql
-- El dispatcher (#57) reclama eventos del outbox con lease: el estado
-- 'sending' marca el evento en vuelo (dispatched_at es el instante del
-- claim; un worker que muere lo libera al expirar el lease, at-least-once).
-- Política de down: herramienta de dev/CI; en producción solo hacia adelante.

ALTER TABLE outbox_event DROP CONSTRAINT outbox_event_state_check;

ALTER TABLE outbox_event
    ADD CONSTRAINT outbox_event_state_check
    CHECK (state IN ('pending', 'sending', 'sent', 'failed'));
