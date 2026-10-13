-- Revert integin Hardening: Revert river_job HOT updates, fillfactor and prune index

BEGIN;

DROP INDEX IF EXISTS river_job_prune_idx;
DROP INDEX IF EXISTS river_job_active_state_idx;

ALTER TABLE river_job RESET (fillfactor);

CREATE INDEX IF NOT EXISTS river_job_state_and_scheduled_at
    ON river_job (state, scheduled_at);

COMMIT;
