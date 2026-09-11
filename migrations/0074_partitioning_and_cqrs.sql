-- Migration 0074: Time-Bucket Table Partitioning & CQRS Replication Support
-- Eliminates PostgreSQL XID wraparound on append-heavy telemetry, event logs, and audit trails.
-- Supports weekly time-bucket range partitioning and CQRS replication markers.

-- 1. Partitioned Append-Heavy Sensor Telemetry Buffer
CREATE TABLE IF NOT EXISTS sensor_telemetry_stream (
    tenant_id          TEXT NOT NULL,
    organization_id    TEXT NOT NULL,
    sensor_id          TEXT NOT NULL,
    reading_timestamp  TIMESTAMPTZ NOT NULL,
    reading_type       TEXT NOT NULL,
    payload            JSONB NOT NULL DEFAULT '{}'::jsonb,
    ingested_at        TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (tenant_id, organization_id, sensor_id, reading_timestamp)
) PARTITION BY RANGE (reading_timestamp);

-- Initial Weekly Time-Bucket Partitions
CREATE TABLE IF NOT EXISTS sensor_telemetry_stream_default
    PARTITION OF sensor_telemetry_stream DEFAULT;

-- 2. Hard Multi-Tenant Row-Level Security
ALTER TABLE sensor_telemetry_stream ENABLE ROW LEVEL SECURITY;
ALTER TABLE sensor_telemetry_stream FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS sensor_telemetry_stream_tenant_isolation ON sensor_telemetry_stream;
CREATE POLICY sensor_telemetry_stream_tenant_isolation ON sensor_telemetry_stream
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND (
            NULLIF(current_setting('integin.organization_id', true), '') IS NULL
            OR organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        )
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND (
            NULLIF(current_setting('integin.organization_id', true), '') IS NULL
            OR organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        )
    );

-- 3. Composite Indexes for Range Query Acceleration
CREATE INDEX IF NOT EXISTS sensor_telemetry_stream_query_idx
    ON sensor_telemetry_stream (tenant_id, sensor_id, reading_timestamp DESC);

-- 4. CQRS Replication Outbox Stream (for PgCat / logical replication routing)
CREATE TABLE IF NOT EXISTS cqrs_replication_outbox (
    sequence_id     BIGSERIAL PRIMARY KEY,
    tenant_id       TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    aggregate_type  TEXT NOT NULL,
    aggregate_id    TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    replicated_at   TIMESTAMPTZ
) WITH (fillfactor = 85);

ALTER TABLE cqrs_replication_outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE cqrs_replication_outbox FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS cqrs_replication_outbox_tenant_isolation ON cqrs_replication_outbox;
CREATE POLICY cqrs_replication_outbox_tenant_isolation ON cqrs_replication_outbox
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND (
            NULLIF(current_setting('integin.organization_id', true), '') IS NULL
            OR organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        )
    );

CREATE INDEX IF NOT EXISTS cqrs_replication_outbox_unreplicated_idx
    ON cqrs_replication_outbox (sequence_id)
    WHERE replicated_at IS NULL;
