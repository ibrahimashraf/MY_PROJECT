-- Migration 0079: Signed Audit Checkpoints (Workstream 6).
-- Append-only registry of signed audit checkpoint metadata. The full signed
-- checkpoint objects (entries + Ed25519 signature) live in object storage;
-- this table holds the RLS-isolated link row. Rows are immutable: UPDATE and
-- DELETE are blocked by the audit_checkpoint_no_update trigger.

BEGIN;

CREATE TABLE IF NOT EXISTS audit_checkpoint_registry (
    id                   TEXT        PRIMARY KEY,
    tenant_id            TEXT        NOT NULL,
    environment          TEXT        NOT NULL,
    sequence_start       BIGINT      NOT NULL,
    sequence_end         BIGINT      NOT NULL,
    previous_root_sha256 TEXT        NOT NULL,
    root_sha256          TEXT        NOT NULL,
    signature            TEXT        NOT NULL,
    key_id               TEXT        NOT NULL,
    object_key           TEXT        NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, root_sha256),
    CHECK (sequence_end >= sequence_start),
    CHECK (
        btrim(tenant_id) <> '' AND
        btrim(environment) <> '' AND
        btrim(root_sha256) <> ''
    )
);

CREATE INDEX IF NOT EXISTS audit_checkpoint_registry_tenant_env_sequence_idx
    ON audit_checkpoint_registry (tenant_id, environment, sequence_end DESC);

REVOKE ALL ON TABLE audit_checkpoint_registry FROM PUBLIC, integin_test_runtime;
GRANT SELECT, INSERT ON TABLE audit_checkpoint_registry TO integin_test_runtime;

ALTER TABLE audit_checkpoint_registry ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_checkpoint_registry FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS audit_checkpoint_registry_tenant_isolation ON audit_checkpoint_registry;
CREATE POLICY audit_checkpoint_registry_tenant_isolation ON audit_checkpoint_registry
    USING (tenant_id = NULLIF(current_setting('integin.tenant_id', true), '') AND tenant_id IS NOT NULL)
    WITH CHECK (tenant_id = NULLIF(current_setting('integin.tenant_id', true), '') AND tenant_id IS NOT NULL);

CREATE OR REPLACE FUNCTION audit_checkpoint_no_update() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit_checkpoint_registry is append-only: UPDATE/DELETE not allowed';
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS audit_checkpoint_immutable ON audit_checkpoint_registry;
CREATE TRIGGER audit_checkpoint_immutable
    BEFORE UPDATE OR DELETE ON audit_checkpoint_registry
    FOR EACH ROW EXECUTE FUNCTION audit_checkpoint_no_update();

COMMIT;