-- integin Hardening: Asynchronous Soft-Delete & Anti-Cascade Avalanche Table
-- Prevents max_locks_per_transaction overflow and long exclusive lock outages during enterprise tenant purges.

BEGIN;

CREATE TABLE IF NOT EXISTS tenant_purge_tombstone (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL,
    organization_id TEXT,
    requested_by TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED')),
    batch_size INT NOT NULL DEFAULT 5000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS tenant_purge_tombstone_status_idx
    ON tenant_purge_tombstone (status, created_at)
    WHERE status IN ('PENDING', 'PROCESSING');

ALTER TABLE tenant_purge_tombstone ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_purge_tombstone FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_purge_tombstone_isolation ON tenant_purge_tombstone;
CREATE POLICY tenant_purge_tombstone_isolation ON tenant_purge_tombstone
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND tenant_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND tenant_id IS NOT NULL
    );

COMMIT;
