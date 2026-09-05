-- INTEGIN certificate authority lifecycle foundation.
-- CANDIDATE ONLY: do not apply without a verified pre-apply backup and isolated up/down review.
BEGIN;

CREATE TABLE IF NOT EXISTS certificate_policy (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    template_code TEXT NOT NULL,
    template_version BIGINT NOT NULL CHECK (template_version > 0),
    policy_version BIGINT NOT NULL CHECK (policy_version > 0),
    validity_days BIGINT NOT NULL CHECK (validity_days > 0 AND validity_days <= 3650),
    self_issue_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL CHECK (status IN ('DRAFT', 'APPROVED', 'RETIRED')),
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_by TEXT,
    approved_at TIMESTAMPTZ,
    UNIQUE (tenant_id, organization_id, template_code, template_version, policy_version),
    FOREIGN KEY (tenant_id, organization_id, template_code, template_version) REFERENCES certificate_template (tenant_id, organization_id, template_code, version)
 );

CREATE TABLE IF NOT EXISTS certificate_number_sequence (
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    next_value BIGINT NOT NULL CHECK (next_value > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id)
 );

CREATE TABLE IF NOT EXISTS certificate_record (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    certificate_number TEXT,
    inspection_id TEXT NOT NULL REFERENCES inspection_record(id),
    inspection_revision BIGINT NOT NULL CHECK (inspection_revision > 0),
    asset_id TEXT NOT NULL,
    template_code TEXT NOT NULL,
    template_version BIGINT NOT NULL CHECK (template_version > 0),
    policy_id TEXT NOT NULL REFERENCES certificate_policy(id),
    policy_version BIGINT NOT NULL CHECK (policy_version > 0),
    profile TEXT NOT NULL CHECK (profile IN ('INDEPENDENT_REVIEW', 'SENIOR_SELF_ISSUE')),
    status TEXT NOT NULL CHECK (status IN ('DRAFT', 'PENDING_REVIEW', 'APPROVED', 'SIGNED', 'ISSUED', 'EXPIRED', 'REVOKED', 'SUPERSEDED')),
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_by TEXT,
    reviewed_at TIMESTAMPTZ,
    signed_by TEXT,
    signed_at TIMESTAMPTZ,
    issued_by TEXT,
    issued_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    self_issue_reason TEXT,
    self_issue_policy_evidence TEXT,
    public_token_digest BYTEA UNIQUE,
    revoked_by TEXT,
    revoked_at TIMESTAMPTZ,
    revocation_reason TEXT,
    supersedes_id TEXT REFERENCES certificate_record(id),
    superseded_by_id TEXT REFERENCES certificate_record(id),
    CHECK ((profile = 'INDEPENDENT_REVIEW' AND self_issue_reason IS NULL AND self_issue_policy_evidence IS NULL) OR (profile = 'SENIOR_SELF_ISSUE' AND self_issue_reason IS NOT NULL AND self_issue_policy_evidence IS NOT NULL)),
    CHECK ((status IN ('ISSUED', 'EXPIRED', 'REVOKED', 'SUPERSEDED') AND issued_by IS NOT NULL AND issued_at IS NOT NULL AND expires_at IS NOT NULL AND certificate_number IS NOT NULL) OR status NOT IN ('ISSUED', 'EXPIRED', 'REVOKED', 'SUPERSEDED')),
    UNIQUE (tenant_id, organization_id, certificate_number)
 );
CREATE UNIQUE INDEX certificate_record_one_active_revision ON certificate_record (tenant_id, organization_id, inspection_id, inspection_revision) WHERE status IN ('DRAFT', 'PENDING_REVIEW', 'APPROVED', 'SIGNED', 'ISSUED', 'EXPIRED');

CREATE TABLE IF NOT EXISTS certificate_snapshot (
    certificate_id TEXT PRIMARY KEY REFERENCES certificate_record(id) ON DELETE RESTRICT,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    template_snapshot JSONB NOT NULL,
    cell_snapshot JSONB NOT NULL,
    snapshot_sha256 BYTEA NOT NULL CHECK (octet_length(snapshot_sha256) = 32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
 );

CREATE TABLE IF NOT EXISTS certificate_audit_event (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    certificate_id TEXT NOT NULL REFERENCES certificate_record(id) ON DELETE RESTRICT,
    action TEXT NOT NULL CHECK (action IN ('DRAFT_CREATED', 'REVIEWED', 'SIGNED', 'ISSUED', 'EXPIRED', 'REVOKED', 'SUPERSEDED')),
    actor_id TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    policy_evidence JSONB NOT NULL DEFAULT '{}'::jsonb
 );

CREATE INDEX certificate_record_scope_status_idx ON certificate_record (tenant_id, organization_id, status, created_at DESC);
CREATE INDEX certificate_audit_event_certificate_idx ON certificate_audit_event (tenant_id, organization_id, certificate_id, occurred_at);

ALTER TABLE certificate_policy ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_policy FORCE ROW LEVEL SECURITY;
ALTER TABLE certificate_number_sequence ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_number_sequence FORCE ROW LEVEL SECURITY;
ALTER TABLE certificate_record ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_record FORCE ROW LEVEL SECURITY;
ALTER TABLE certificate_snapshot ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_snapshot FORCE ROW LEVEL SECURITY;
ALTER TABLE certificate_audit_event ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_audit_event FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS certificate_policy_tenant_isolation ON certificate_policy;
CREATE POLICY certificate_policy_tenant_isolation ON certificate_policy USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));
DROP POLICY IF EXISTS certificate_number_sequence_tenant_isolation ON certificate_number_sequence;
CREATE POLICY certificate_number_sequence_tenant_isolation ON certificate_number_sequence USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));
DROP POLICY IF EXISTS certificate_record_tenant_isolation ON certificate_record;
CREATE POLICY certificate_record_tenant_isolation ON certificate_record USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));
DROP POLICY IF EXISTS certificate_snapshot_tenant_isolation ON certificate_snapshot;
CREATE POLICY certificate_snapshot_tenant_isolation ON certificate_snapshot USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));
DROP POLICY IF EXISTS certificate_audit_event_tenant_isolation ON certificate_audit_event;
CREATE POLICY certificate_audit_event_tenant_isolation ON certificate_audit_event USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;
