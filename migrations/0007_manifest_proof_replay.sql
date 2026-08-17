-- Durable, tenant-scoped replay protection for signed work-package manifest proofs.
-- Source-only until separately reviewed and applied to the isolated pilot.

CREATE TABLE IF NOT EXISTS manifest_proof_replay (
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    purpose TEXT NOT NULL,
    request_id TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, device_id, purpose, request_id),
    CHECK (btrim(tenant_id) <> ''),
    CHECK (btrim(organization_id) <> ''),
    CHECK (btrim(device_id) <> ''),
    CHECK (btrim(purpose) <> ''),
    CHECK (btrim(request_id) <> '')
);

CREATE INDEX IF NOT EXISTS manifest_proof_replay_expiry_idx
    ON manifest_proof_replay (expires_at);

ALTER TABLE manifest_proof_replay ENABLE ROW LEVEL SECURITY;

CREATE POLICY manifest_proof_replay_tenant_isolation
    ON manifest_proof_replay
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

-- Retention is intentionally bounded. The store also removes an expired matching
-- key atomically before insert; operations may periodically run:
-- DELETE FROM manifest_proof_replay WHERE expires_at < now();
