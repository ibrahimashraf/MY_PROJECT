CREATE TABLE IF NOT EXISTS event_log (
    id BIGSERIAL NOT NULL,
    event_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    environment TEXT NOT NULL CHECK (environment IN ('TESTING', 'LIVE')),
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    aggregate_version INTEGER NOT NULL CHECK (aggregate_version > 0),
    event_type TEXT NOT NULL,
    schema_version INTEGER NOT NULL CHECK (schema_version > 0),
    occurred_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    PRIMARY KEY (id, occurred_at),
    UNIQUE (event_id, occurred_at),
    UNIQUE (aggregate_type, aggregate_id, aggregate_version, occurred_at)
) PARTITION BY RANGE (occurred_at);

CREATE TABLE IF NOT EXISTS event_log_default PARTITION OF event_log DEFAULT;

CREATE INDEX IF NOT EXISTS event_log_aggregate_replay_idx
    ON event_log (aggregate_type, aggregate_id, aggregate_version, occurred_at);

CREATE INDEX IF NOT EXISTS event_log_tenant_time_idx
    ON event_log (tenant_id, occurred_at);

CREATE INDEX IF NOT EXISTS event_log_event_type_idx
    ON event_log (event_type, occurred_at);
