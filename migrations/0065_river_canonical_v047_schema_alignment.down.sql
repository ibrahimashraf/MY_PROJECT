-- Revert INTEGIN Hardening: Revert River canonical v0.47 schema alignment

BEGIN;

DROP TABLE IF EXISTS river_notification;
DROP TABLE IF EXISTS river_queue;
DROP INDEX IF EXISTS river_job_unique_idx;
DROP FUNCTION IF EXISTS river_job_state_in_bitmask(BIT(8), river_job_state);

ALTER TABLE river_job DROP COLUMN IF EXISTS unique_states;

ALTER TABLE river_job ALTER COLUMN state DROP DEFAULT;
ALTER TABLE river_job 
    ALTER COLUMN state TYPE text USING state::text;
ALTER TABLE river_job 
    ALTER COLUMN state SET DEFAULT 'available'::text;

ALTER TABLE river_job 
    ADD CONSTRAINT river_job_state_check 
    CHECK (state IN ('available', 'cancelled', 'completed', 'discarded', 'pending', 'retryable', 'running', 'scheduled'));

CREATE UNIQUE INDEX IF NOT EXISTS river_job_unique_active_idx 
    ON river_job (unique_key) 
    WHERE state IN ('available', 'running', 'retryable');

DROP TABLE IF EXISTS river_migration;
DROP TYPE IF EXISTS river_job_state;

COMMIT;
