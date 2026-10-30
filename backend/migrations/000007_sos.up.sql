-- 000007_sos.up.sql
-- SOS (R-10): un kind más en la cola, no una máquina aparte. El armado crea
-- un job sos_alert con run_at = now()+10s (countdown server-side) y la
-- cancelación reusa el path transaccional de cancelación.
-- Dos cambios de forma:
--   1. El CHECK de kinds gana 'sos_alert'.
--   2. La UNIQUE (user_id, cycle_date, kind) del ladder impediría un segundo
--      SOS el mismo día: pasa a índice único PARCIAL que exime a sos_alert
--      (un SOS puede repetirse; la escalera sigue siendo única por ciclo).
-- Política de down: herramienta de dev/CI; en producción solo hacia adelante.

ALTER TABLE escalation_jobs DROP CONSTRAINT escalation_jobs_kind_check;

ALTER TABLE escalation_jobs
    ADD CONSTRAINT escalation_jobs_kind_check
    CHECK (kind IN ('reminder_soft', 'reminder_urgent', 'notify_contact',
                    'notify_circle', 'sos_alert'));

ALTER TABLE escalation_jobs DROP CONSTRAINT escalation_jobs_user_id_cycle_date_kind_key;

CREATE UNIQUE INDEX escalation_jobs_ladder_unique_idx
    ON escalation_jobs (user_id, cycle_date, kind)
    WHERE kind <> 'sos_alert';
