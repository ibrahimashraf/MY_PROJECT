-- INTEGIN certificate artifact metadata foundation.
-- CANDIDATE ONLY. Do not apply without fresh backup and disposable up/down review.
BEGIN;

CREATE TABLE IF NOT EXISTS certificate_artifact (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    certificate_id TEXT NOT NULL,
    artifact_type TEXT NOT NULL CHECK (artifact_type IN ('CERTIFICATE_PDF')),
    object_key TEXT NOT NULL,
    content_type TEXT NOT NULL CHECK (content_type = 'application/pdf'),
    byte_size BIGINT NOT NULL CHECK (byte_size > 0),
    artifact_sha256 BYTEA NOT NULL CHECK (octet_length(artifact_sha256) = 32),
    snapshot_sha256 BYTEA NOT NULL CHECK (octet_length(snapshot_sha256) = 32),
    renderer_version TEXT NOT NULL,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, certificate_id, artifact_type),
    UNIQUE (tenant_id, organization_id, object_key),
    FOREIGN KEY (certificate_id) REFERENCES certificate_record(id) ON DELETE RESTRICT,
    CHECK (btrim(object_key) <> '' AND object_key !~ '(^/|\\.\\.)'),
    CHECK (btrim(renderer_version) <> '')
);

CREATE INDEX certificate_artifact_lookup_idx
    ON certificate_artifact (tenant_id, organization_id, certificate_id, artifact_type);

ALTER TABLE certificate_artifact ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_artifact FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS certificate_artifact_tenant_isolation ON certificate_artifact;
CREATE POLICY certificate_artifact_tenant_isolation ON certificate_artifact
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;
