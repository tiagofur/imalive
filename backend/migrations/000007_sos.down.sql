-- (dev/CI only; see up header for the policy)
DROP INDEX IF EXISTS escalation_jobs_ladder_unique_idx;
ALTER TABLE escalation_jobs DROP CONSTRAINT IF EXISTS escalation_jobs_kind_check;
ALTER TABLE escalation_jobs
    ADD CONSTRAINT escalation_jobs_kind_check
    CHECK (kind IN ('reminder_soft', 'reminder_urgent', 'notify_contact', 'notify_circle'));
ALTER TABLE escalation_jobs
    ADD CONSTRAINT escalation_jobs_user_id_cycle_date_kind_key
    UNIQUE (user_id, cycle_date, kind);
