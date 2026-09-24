BEGIN;

CREATE TABLE assurance_rule_bundles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid, -- NULL implies a global system standard (e.g. ISO4309)
    bundle_id varchar(100) NOT NULL,
    version varchar(50) NOT NULL,
    rules jsonb NOT NULL,
    rule_hash varchar(100) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Ensure a specific version of a bundle cannot be duplicated per tenant
ALTER TABLE assurance_rule_bundles 
ADD CONSTRAINT uq_rule_bundle_version 
UNIQUE (tenant_id, bundle_id, version);

-- RLS Policies
ALTER TABLE assurance_rule_bundles ENABLE ROW LEVEL SECURITY;

CREATE POLICY rule_bundles_tenant_isolation ON assurance_rule_bundles
    FOR SELECT
    USING (
        tenant_id = current_setting('app.current_tenant', true)::uuid
        OR tenant_id IS NULL -- Global rules are visible to all tenants
    );

COMMIT;
