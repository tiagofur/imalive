-- (dev/CI only; see up header for the policy)
DROP TRIGGER IF EXISTS audit_events_append_only ON audit_events;
DROP FUNCTION IF EXISTS audit_events_block_mutation();
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS outbox_event;
DROP TABLE IF EXISTS escalation_jobs;
