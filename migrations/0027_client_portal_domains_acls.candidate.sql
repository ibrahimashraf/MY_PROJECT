-- 0027_client_portal_domains_acls.candidate.sql
-- Client portal custom domains + granular ACLs + QuickLink + client audit trail

BEGIN;

CREATE TABLE client_portal_domain (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    org_id UUID NOT NULL REFERENCES organizations(id),
    domain TEXT NOT NULL,
    cert_arn TEXT,
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'VERIFIED', 'ACTIVE', 'REVOKED')),
    verified_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, org_id, domain)
);

ALTER TABLE client_portal_domain ENABLE ROW LEVEL SECURITY;
ALTER TABLE client_portal_domain FORCE ROW LEVEL SECURITY;

CREATE POLICY client_portal_domain_tenant_isolation ON client_portal_domain
    USING (tenant_id = current_setting('app.current_tenant')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

CREATE INDEX client_portal_domain_status_idx
    ON client_portal_domain (tenant_id, org_id, status);

CREATE TABLE client_portal_acl (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    org_id UUID NOT NULL REFERENCES organizations(id),
    client_id UUID NOT NULL,
    user_email TEXT NOT NULL,
    permissions JSONB NOT NULL DEFAULT '{"can_view_equip": false, "can_download_certs": false, "can_view_history": false, "can_download_registers": false}'::jsonb,
    granted_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, org_id, client_id, user_email)
);

ALTER TABLE client_portal_acl ENABLE ROW LEVEL SECURITY;
ALTER TABLE client_portal_acl FORCE ROW LEVEL SECURITY;

CREATE POLICY client_portal_acl_tenant_isolation ON client_portal_acl
    USING (tenant_id = current_setting('app.current_tenant')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

CREATE INDEX client_portal_acl_client_lookup_idx
    ON client_portal_acl (tenant_id, org_id, client_id);

CREATE TABLE quick_link (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    org_id UUID NOT NULL REFERENCES organizations(id),
    token CHAR(32) NOT NULL,
    target_type TEXT NOT NULL CHECK (target_type IN ('INSPECTION', 'CERTIFICATE', 'ASSET', 'REGISTER')),
    target_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    access_count BIGINT NOT NULL DEFAULT 0 CHECK (access_count >= 0),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (token)
);

ALTER TABLE quick_link ENABLE ROW LEVEL SECURITY;
ALTER TABLE quick_link FORCE ROW LEVEL SECURITY;

CREATE POLICY quick_link_tenant_isolation ON quick_link
    USING (tenant_id = current_setting('app.current_tenant')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

CREATE INDEX quick_link_token_idx
    ON quick_link (token);

CREATE INDEX quick_link_expiry_idx
    ON quick_link (expires_at)
    WHERE expires_at > now();

CREATE TABLE client_audit_entry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    org_id UUID NOT NULL REFERENCES organizations(id),
    client_id UUID NOT NULL,
    actor_email TEXT NOT NULL,
    action TEXT NOT NULL,
    target_type TEXT,
    target_id UUID,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE client_audit_entry ENABLE ROW LEVEL SECURITY;
ALTER TABLE client_audit_entry FORCE ROW LEVEL SECURITY;

CREATE POLICY client_audit_entry_tenant_isolation ON client_audit_entry
    USING (tenant_id = current_setting('app.current_tenant')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

CREATE INDEX client_audit_entry_client_time_idx
    ON client_audit_entry (tenant_id, org_id, client_id, occurred_at DESC);

COMMIT;