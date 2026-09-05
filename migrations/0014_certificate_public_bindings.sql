-- INTEGIN certificate public-binding foundation.
-- CANDIDATE ONLY. Do not apply without a fresh backup, disposable up/down review,
-- cross-organization isolation proof, and issuance snapshot integration proof.
BEGIN;

ALTER TABLE certificate_policy
    ADD CONSTRAINT certificate_policy_tenant_org_id_unique
    UNIQUE (tenant_id, organization_id, id);

CREATE TABLE IF NOT EXISTS asset_registry (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    asset_type TEXT NOT NULL,
    serial_number TEXT NOT NULL,
    description TEXT NOT NULL,
    lifecycle_state TEXT NOT NULL CHECK (lifecycle_state IN ('ACTIVE', 'RETIRED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by TEXT NOT NULL,
    updated_by TEXT NOT NULL,
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, asset_id),
    CHECK (btrim(asset_id) <> ''),
    CHECK (btrim(asset_type) <> ''),
    CHECK (btrim(serial_number) <> ''),
    CHECK (btrim(description) <> '')
);

CREATE TABLE IF NOT EXISTS inspection_public_scope (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    inspection_id TEXT NOT NULL,
    inspection_revision BIGINT NOT NULL CHECK (inspection_revision > 0),
    inspection_type TEXT NOT NULL,
    taxonomy_version BIGINT NOT NULL CHECK (taxonomy_version > 0),
    result_state TEXT NOT NULL CHECK (result_state IN ('PASS', 'FAIL', 'CONDITIONAL')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by TEXT NOT NULL,
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, inspection_id, inspection_revision),
    FOREIGN KEY (tenant_id, organization_id, inspection_id)
        REFERENCES inspection_record (tenant_id, organization_id, id),
    CHECK (btrim(inspection_type) <> '')
);

CREATE TABLE IF NOT EXISTS inspection_public_scope_item (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    scope_id TEXT NOT NULL,
    scope_code TEXT NOT NULL,
    display_label TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('PASS', 'FAIL', 'NOT_APPLICABLE')),
    display_order BIGINT NOT NULL CHECK (display_order > 0),
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, scope_id, scope_code),
    UNIQUE (tenant_id, organization_id, scope_id, display_order),
    FOREIGN KEY (tenant_id, organization_id, scope_id)
        REFERENCES inspection_public_scope (tenant_id, organization_id, id),
    CHECK (btrim(scope_code) <> ''),
    CHECK (btrim(display_label) <> '')
);

CREATE TABLE IF NOT EXISTS certificate_policy_public_binding (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    policy_id TEXT NOT NULL,
    binding_key TEXT NOT NULL CHECK (binding_key IN ('asset.id', 'asset.serial_number', 'asset.description', 'asset.type', 'inspection.type', 'inspection.public_scope')),
    required BOOLEAN NOT NULL DEFAULT FALSE,
    display_order BIGINT NOT NULL CHECK (display_order > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by TEXT NOT NULL,
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, policy_id, binding_key),
    UNIQUE (tenant_id, organization_id, policy_id, display_order),
    FOREIGN KEY (tenant_id, organization_id, policy_id)
        REFERENCES certificate_policy (tenant_id, organization_id, id)
);

ALTER TABLE certificate_snapshot
    ADD COLUMN public_binding_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX asset_registry_lookup_idx
    ON asset_registry (tenant_id, organization_id, asset_id, lifecycle_state);
CREATE INDEX inspection_public_scope_lookup_idx
    ON inspection_public_scope (tenant_id, organization_id, inspection_id, inspection_revision);
CREATE INDEX certificate_policy_public_binding_lookup_idx
    ON certificate_policy_public_binding (tenant_id, organization_id, policy_id, display_order);

ALTER TABLE asset_registry ENABLE ROW LEVEL SECURITY;
ALTER TABLE asset_registry FORCE ROW LEVEL SECURITY;
ALTER TABLE inspection_public_scope ENABLE ROW LEVEL SECURITY;
ALTER TABLE inspection_public_scope FORCE ROW LEVEL SECURITY;
ALTER TABLE inspection_public_scope_item ENABLE ROW LEVEL SECURITY;
ALTER TABLE inspection_public_scope_item FORCE ROW LEVEL SECURITY;
ALTER TABLE certificate_policy_public_binding ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_policy_public_binding FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS asset_registry_tenant_isolation ON asset_registry;
CREATE POLICY asset_registry_tenant_isolation ON asset_registry
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));
DROP POLICY IF EXISTS inspection_public_scope_tenant_isolation ON inspection_public_scope;
CREATE POLICY inspection_public_scope_tenant_isolation ON inspection_public_scope
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));
DROP POLICY IF EXISTS inspection_public_scope_item_tenant_isolation ON inspection_public_scope_item;
CREATE POLICY inspection_public_scope_item_tenant_isolation ON inspection_public_scope_item
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));
DROP POLICY IF EXISTS certificate_policy_public_binding_tenant_isolation ON certificate_policy_public_binding;
CREATE POLICY certificate_policy_public_binding_tenant_isolation ON certificate_policy_public_binding
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;
