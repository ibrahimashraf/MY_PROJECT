-- Migration 0027: Client Portal Domains + ACLs + QuickLink + Audit Trail
-- Compatible with existing schema (TEXT IDs, integin.* settings, FORCE RLS)
-- CANDIDATE ONLY: do not apply without verified pre-apply backup and isolated up/down review.

BEGIN;

CREATE TABLE IF NOT EXISTS client_portal_domain (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    domain TEXT NOT NULL CHECK (char_length(domain) BETWEEN 1 AND 255),
    cert_arn TEXT,
    status TEXT NOT NULL CHECK (status IN ('PENDING','VERIFIED','ACTIVE','REVOKED')) DEFAULT 'PENDING',
    verified_at TIMESTAMPTZ,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, domain)
);

ALTER TABLE client_portal_domain ENABLE ROW LEVEL SECURITY;
ALTER TABLE client_portal_domain FORCE ROW LEVEL SECURITY;

CREATE POLICY client_portal_domain_tenant_isolation ON client_portal_domain
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE INDEX IF NOT EXISTS client_portal_domain_status_idx ON client_portal_domain (tenant_id, organization_id, status);

CREATE TABLE IF NOT EXISTS client_portal_acl (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    client_id TEXT NOT NULL,
    user_email TEXT NOT NULL CHECK (char_length(user_email) BETWEEN 5 AND 255),
    permissions JSONB NOT NULL DEFAULT '{"can_view_equip":false,"can_download_certs":false,"can_view_history":false,"can_download_registers":false}'::jsonb,
    granted_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, client_id, user_email)
);

ALTER TABLE client_portal_acl ENABLE ROW LEVEL SECURITY;
ALTER TABLE client_portal_acl FORCE ROW LEVEL SECURITY;

CREATE POLICY client_portal_acl_tenant_isolation ON client_portal_acl
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE INDEX IF NOT EXISTS client_portal_acl_client_lookup_idx ON client_portal_acl (tenant_id, organization_id, client_id);

CREATE TABLE IF NOT EXISTS quick_link (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    token CHAR(32) NOT NULL CHECK (char_length(token) = 32),
    target_type TEXT NOT NULL CHECK (target_type IN ('INSPECTION','CERTIFICATE','ASSET','REGISTER')),
    target_id TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    access_count BIGINT NOT NULL DEFAULT 0 CHECK (access_count >= 0),
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (token)
);

ALTER TABLE quick_link ENABLE ROW LEVEL SECURITY;
ALTER TABLE quick_link FORCE ROW LEVEL SECURITY;

CREATE POLICY quick_link_tenant_isolation ON quick_link
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE INDEX IF NOT EXISTS quick_link_token_idx ON quick_link (token);
CREATE INDEX IF NOT EXISTS quick_link_expiry_idx ON quick_link (tenant_id, organization_id, expires_at);

CREATE TABLE IF NOT EXISTS client_audit_entry (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    client_id TEXT NOT NULL,
    actor_email TEXT NOT NULL CHECK (char_length(actor_email) BETWEEN 5 AND 255),
    action TEXT NOT NULL,
    target_type TEXT,
    target_id TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE client_audit_entry ENABLE ROW LEVEL SECURITY;
ALTER TABLE client_audit_entry FORCE ROW LEVEL SECURITY;

CREATE POLICY client_audit_entry_tenant_isolation ON client_audit_entry
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE INDEX IF NOT EXISTS client_audit_entry_client_time_idx ON client_audit_entry (tenant_id, organization_id, client_id, occurred_at DESC);

COMMIT;