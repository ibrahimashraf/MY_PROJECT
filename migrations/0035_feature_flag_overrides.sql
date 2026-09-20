-- Migration 0035: Feature Flag Override Persistence
-- TEXT IDs, combined RLS, FORCE RLS

BEGIN;

CREATE TABLE IF NOT EXISTS feature_flag_override (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    flag_key TEXT NOT NULL CHECK (char_length(flag_key) BETWEEN 1 AND 128),
    scope TEXT NOT NULL CHECK (scope IN ('ORGANIZATION','USER','CLIENT','PROJECT','DEVICE')),
    scope_id TEXT NOT NULL CHECK (char_length(scope_id) BETWEEN 1 AND 255),
    state TEXT NOT NULL CHECK (state IN ('INHERITED','ENABLED','DISABLED','EXPIRED')),
    reason TEXT,
    expires_at TIMESTAMPTZ,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, flag_key, scope, scope_id)
);

CREATE INDEX IF NOT EXISTS idx_flag_override_tenant_org ON feature_flag_override (tenant_id, organization_id);
CREATE INDEX IF NOT EXISTS idx_flag_override_flag ON feature_flag_override (tenant_id, organization_id, flag_key);
CREATE INDEX IF NOT EXISTS idx_flag_override_scope ON feature_flag_override (tenant_id, organization_id, scope, scope_id);

ALTER TABLE feature_flag_override ENABLE ROW LEVEL SECURITY;
ALTER TABLE feature_flag_override FORCE ROW LEVEL SECURITY;

REVOKE ALL ON TABLE feature_flag_override FROM PUBLIC, integin_test_runtime;
GRANT SELECT,INSERT,UPDATE,DELETE ON TABLE feature_flag_override TO integin_test_runtime;
DROP POLICY IF EXISTS feature_flag_override_isolation ON feature_flag_override;
CREATE POLICY feature_flag_override_isolation ON feature_flag_override
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;

