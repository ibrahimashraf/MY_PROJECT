-- Immutable audit log with hash chain for tamper evidence.
-- CANDIDATE ONLY. Do not apply without a fresh backup and disposable review.
BEGIN;

CREATE TABLE IF NOT EXISTS audit_log (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    actor_name TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    old_value JSONB,
    new_value JSONB,
    metadata JSONB,
    ip_address TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    previous_hash TEXT NOT NULL,
    entry_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, id)
);

CREATE INDEX IF NOT EXISTS audit_log_tenant_org_idx ON audit_log (tenant_id, organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_log_entity_idx ON audit_log (tenant_id, organization_id, entity_type, entity_id);
CREATE INDEX IF NOT EXISTS audit_log_actor_idx ON audit_log (tenant_id, organization_id, actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_log_event_idx ON audit_log (tenant_id, organization_id, event_type, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_log_hash_chain_idx ON audit_log (tenant_id, organization_id, previous_hash, entry_hash);

REVOKE ALL ON TABLE audit_log FROM PUBLIC, integin_test_runtime;
GRANT SELECT,INSERT ON TABLE audit_log TO integin_test_runtime;
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS audit_log_tenant_organization_isolation ON audit_log;
CREATE POLICY audit_log_tenant_organization_isolation ON audit_log
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE OR REPLACE FUNCTION audit_log_no_update() RETURNS trigger AS $$
BEGIN RAISE EXCEPTION 'audit_log is immutable: UPDATE/DELETE not allowed'; RETURN NULL; END; $$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS audit_log_immutable ON audit_log;
CREATE TRIGGER audit_log_immutable BEFORE UPDATE OR DELETE ON audit_log FOR EACH ROW EXECUTE FUNCTION audit_log_no_update();

COMMIT;

