-- integin Stage A evidence-retention foundation.
-- CANDIDATE ONLY: requires isolated SQL review, verified pre-apply backup, and controlled pilot execution evidence.

BEGIN;

CREATE TABLE IF NOT EXISTS evidence_metadata (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    inspection_id TEXT NOT NULL,
    object_key TEXT NOT NULL,
    content_type TEXT NOT NULL,
    ciphertext_bytes BIGINT NOT NULL CHECK (ciphertext_bytes > 0),
    plaintext_sha256 TEXT NOT NULL,
    ciphertext_sha256 TEXT NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL,
    device_id TEXT NOT NULL,
    authority_id TEXT NOT NULL,
    authority_epoch BIGINT NOT NULL CHECK (authority_epoch > 0),
    transaction_id TEXT NOT NULL,
    receipt_id TEXT NOT NULL,
    signature_algorithm TEXT NOT NULL,
    key_id TEXT NOT NULL,
    classification TEXT NOT NULL,
    retention_reference TEXT NOT NULL,
    hold_state TEXT NOT NULL,
    redaction_policy_reference TEXT NOT NULL,
    registered_by TEXT NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, object_key),
    FOREIGN KEY (tenant_id, organization_id, inspection_id)
        REFERENCES inspection_record (tenant_id, organization_id, id),
    CHECK (object_key = tenant_id || '/' || organization_id || '/evidence/' || id),
    CHECK (plaintext_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (ciphertext_sha256 ~ '^[0-9a-f]{64}$')
);

CREATE INDEX evidence_metadata_inspection_lookup
    ON evidence_metadata (tenant_id, organization_id, inspection_id, captured_at, id);

ALTER TABLE evidence_metadata ENABLE ROW LEVEL SECURITY;
ALTER TABLE evidence_metadata FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS evidence_metadata_tenant_isolation ON evidence_metadata;
CREATE POLICY evidence_metadata_tenant_isolation ON evidence_metadata
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

COMMIT;
