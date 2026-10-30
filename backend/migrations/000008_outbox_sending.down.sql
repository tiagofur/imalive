-- (dev/CI only; see up header for the policy)
ALTER TABLE outbox_event DROP CONSTRAINT IF EXISTS outbox_event_state_check;
ALTER TABLE outbox_event
    ADD CONSTRAINT outbox_event_state_check
    CHECK (state IN ('pending', 'sent', 'failed'));
