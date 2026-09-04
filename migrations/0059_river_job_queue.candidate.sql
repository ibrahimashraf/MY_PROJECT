-- INTEGIN Background Queue: River durable job queue schema.
-- CANDIDATE ONLY: do not apply without a verified backup and isolated SQL review.
-- Builds on 0058_custody_site_handover.

BEGIN;

-- Core river_job table for PostgreSQL transactional job processing
CREATE TABLE IF NOT EXISTS river_job (
    id BIGSERIAL PRIMARY KEY,
    args JSONB NOT NULL,
    attempt SMALLINT NOT NULL DEFAULT 0,
    attempted_at TIMESTAMPTZ,
    attempted_by TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    errors JSONB[],
    finalized_at TIMESTAMPTZ,
    kind TEXT NOT NULL,
    max_attempts SMALLINT NOT NULL DEFAULT 25,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    priority SMALLINT NOT NULL DEFAULT 1,
    queue TEXT NOT NULL DEFAULT 'default',
    state TEXT NOT NULL DEFAULT 'available' CHECK (state IN ('available', 'cancelled', 'completed', 'discarded', 'pending', 'retryable', 'running', 'scheduled')),
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    tags VARCHAR(255)[] NOT NULL DEFAULT '{}'::varchar(255)[],
    unique_key BYTEA
);

CREATE INDEX IF NOT EXISTS river_job_kind ON river_job (kind);
CREATE INDEX IF NOT EXISTS river_job_state_and_scheduled_at ON river_job (state, scheduled_at);
CREATE INDEX IF NOT EXISTS river_job_args_index ON river_job USING GIN (args);

-- river_leader for distributed multi-pod coordination
CREATE TABLE IF NOT EXISTS river_leader (
    elected_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    leader_id TEXT NOT NULL PRIMARY KEY,
    name TEXT NOT NULL DEFAULT 'default' UNIQUE
);

COMMIT;
