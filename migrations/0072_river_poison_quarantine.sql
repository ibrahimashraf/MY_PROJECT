-- DLQ quarantine for River poison-pill jobs: preserves forensic evidence of
-- fatal unhandled payload panics and binds quarantine rows to the tenant that
-- owns the job so they never stall the shared queue or leak across tenants.

BEGIN;

CREATE TABLE IF NOT EXISTS river_poison_quarantine (
    job_id         BIGINT      PRIMARY KEY,
    kind           TEXT        NOT NULL,
    tenant_id      TEXT,
    organization_id TEXT       ,
    args           JSONB       NOT NULL,
    error_text     TEXT        NOT NULL,
    stack_trace    TEXT        NOT NULL DEFAULT '',
    attempt        INTEGER     NOT NULL,
    original_state TEXT        NOT NULL,
    quarantined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (btrim(kind) <> ''),
    CHECK (attempt >= 1)
);

CREATE INDEX IF NOT EXISTS river_poison_quarantine_kind_idx
    ON river_poison_quarantine (kind, quarantined_at DESC);

CREATE INDEX IF NOT EXISTS river_poison_quarantine_tenant_idx
    ON river_poison_quarantine (tenant_id, organization_id, quarantined_at DESC);

ALTER TABLE river_poison_quarantine ENABLE ROW LEVEL SECURITY;
ALTER TABLE river_poison_quarantine FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS river_poison_quarantine_tenant_isolation ON river_poison_quarantine;
CREATE POLICY river_poison_quarantine_tenant_isolation ON river_poison_quarantine
    USING (
        tenant_id IS NULL
        OR (
            tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
            AND (
                organization_id IS NULL
                OR organization_id = NULLIF(current_setting('integin.organization_id', true), '')
            )
        )
    )
    WITH CHECK (
        tenant_id IS NULL
        OR (
            tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
            AND (
                organization_id IS NULL
                OR organization_id = NULLIF(current_setting('integin.organization_id', true), '')
            )
        )
    );

GRANT SELECT, INSERT, UPDATE, DELETE ON river_poison_quarantine TO integin_runtime;

COMMIT;