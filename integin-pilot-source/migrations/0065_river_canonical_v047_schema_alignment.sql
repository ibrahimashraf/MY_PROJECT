-- integin Hardening: Canonical River v0.47 Schema Alignment
-- Aligns river_job with official River driver schema (ENUM river_job_state, unique_states bit(8),
-- river_job_state_in_bitmask, river_queue, river_notification, and river_migration table).

BEGIN;

-- 1. Create canonical river_job_state enum if not exists
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'river_job_state') THEN
        CREATE TYPE river_job_state AS ENUM (
            'available',
            'cancelled',
            'completed',
            'discarded',
            'pending',
            'retryable',
            'running',
            'scheduled'
        );
    END IF;
END$$;

-- 2. Drop existing partial indexes that reference state as text
DROP INDEX IF EXISTS river_job_available_idx;
DROP INDEX IF EXISTS river_job_active_state_idx;
DROP INDEX IF EXISTS river_job_prune_idx;
DROP INDEX IF EXISTS river_job_queue_priority_idx;
DROP INDEX IF EXISTS river_job_unique_active_idx;

-- 3. Migrate river_job.state column to canonical enum type
ALTER TABLE river_job ALTER COLUMN state DROP DEFAULT;
ALTER TABLE river_job DROP CONSTRAINT IF EXISTS river_job_state_check;

ALTER TABLE river_job 
    ALTER COLUMN state TYPE river_job_state USING state::river_job_state;

ALTER TABLE river_job 
    ALTER COLUMN state SET DEFAULT 'available'::river_job_state;

-- 4. Recreate high-performance partial indexes using river_job_state enum
CREATE INDEX IF NOT EXISTS river_job_available_idx 
    ON river_job (scheduled_at, priority) 
    WHERE state = 'available';

CREATE INDEX IF NOT EXISTS river_job_active_state_idx
    ON river_job (state, scheduled_at)
    WHERE state IN ('available', 'running', 'retryable');

CREATE INDEX IF NOT EXISTS river_job_queue_priority_idx 
    ON river_job (queue, priority, scheduled_at) 
    WHERE state = 'available';

CREATE INDEX IF NOT EXISTS river_job_prune_idx
    ON river_job (finalized_at)
    WHERE state IN ('completed', 'cancelled', 'discarded');

-- 5. Add unique_states bit(8)
ALTER TABLE river_job ADD COLUMN IF NOT EXISTS unique_states BIT(8);

-- 4. Create immutable bitmask evaluation function required by River for unique jobs
CREATE OR REPLACE FUNCTION river_job_state_in_bitmask(bitmask BIT(8), state river_job_state)
RETURNS boolean
LANGUAGE SQL
IMMUTABLE
AS $$
    SELECT CASE state
        WHEN 'available' THEN get_bit(bitmask, 7)
        WHEN 'cancelled' THEN get_bit(bitmask, 6)
        WHEN 'completed' THEN get_bit(bitmask, 5)
        WHEN 'discarded' THEN get_bit(bitmask, 4)
        WHEN 'pending'   THEN get_bit(bitmask, 3)
        WHEN 'retryable' THEN get_bit(bitmask, 2)
        WHEN 'running'   THEN get_bit(bitmask, 1)
        WHEN 'scheduled' THEN get_bit(bitmask, 0)
        ELSE 0
    END = 1;
$$;

-- 5. Canonical unique index for River v0.47
DROP INDEX IF EXISTS river_job_unique_active_idx;

CREATE UNIQUE INDEX IF NOT EXISTS river_job_unique_idx ON river_job (unique_key)
    WHERE unique_key IS NOT NULL
      AND unique_states IS NOT NULL
      AND river_job_state_in_bitmask(unique_states, state);

-- 6. Create river_queue table
CREATE TABLE IF NOT EXISTS river_queue (
    name text PRIMARY KEY NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    paused_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- 7. Create river_notification outbox table
CREATE TABLE IF NOT EXISTS river_notification (
    id bigserial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    payload text NOT NULL,
    topic text NOT NULL,
    CONSTRAINT topic_length CHECK (length(topic) > 0 AND length(topic) < 128)
);

CREATE INDEX IF NOT EXISTS river_notification_created_at_idx ON river_notification (created_at);
CREATE INDEX IF NOT EXISTS river_notification_topic_id_idx ON river_notification (topic, id);

-- 8. Create canonical river_migration table and register all 7 versions
DROP TABLE IF EXISTS river_migration CASCADE;

CREATE TABLE river_migration (
    line TEXT NOT NULL,
    version bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT line_length CHECK (char_length(line) > 0 AND char_length(line) < 128),
    CONSTRAINT version_gte_1 CHECK (version >= 1),
    PRIMARY KEY (line, version)
);

INSERT INTO river_migration (line, version)
VALUES 
    ('main', 1),
    ('main', 2),
    ('main', 3),
    ('main', 4),
    ('main', 5),
    ('main', 6),
    ('main', 7)
ON CONFLICT (line, version) DO NOTHING;

COMMIT;
