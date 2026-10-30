-- 000004_pause.down.sql (dev/CI only; see up header for the policy)

ALTER TABLE users DROP COLUMN pause_until;
