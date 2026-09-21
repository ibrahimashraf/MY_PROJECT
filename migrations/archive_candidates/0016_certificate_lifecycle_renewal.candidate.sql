-- integin certificate lifecycle renewal foundation.
-- CANDIDATE ONLY: do not apply without a verified pre-apply backup and isolated up/down review.
-- Migration fingerprint: 0016_certificate_lifecycle_renewal (sha256 to be recorded in review).
BEGIN;

-- Renewal tracking table: records each renewal attempt with authority derivation and snapshot
CREATE TABLE IF NOT EXISTS certificate_renewal_attempt (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    certificate_id TEXT NOT NULL REFERENCES certificate_record(id) ON DELETE RESTRICT,
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    authority_tenant_id TEXT NOT NULL,
    authority_organization_id TEXT NOT NULL,
    previous_status TEXT NOT NULL CHECK (previous_status IN ('DRAFT', 'PENDING_REVIEW', 'APPROVED', 'SIGNED', 'ISSUED', 'EXPIRED', 'REVOKED', 'SUPERSEDED')),
    new_status TEXT NOT NULL CHECK (new_status IN ('DRAFT', 'PENDING_REVIEW', 'APPROVED', 'SIGNED', 'ISSUED', 'EXPIRED', 'REVOKED', 'SUPERSEDED')),
    renewal_reason TEXT NOT NULL CHECK (length(renewal_reason) > 0),
    authority_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    result_state TEXT CHECK (result_state IN ('SUCCESS', 'FAILED_AUTHORITY', 'FAILED_INSPECTION', 'FAILED_POLICY', 'DUPLICATE')),
    error_detail TEXT,
    completed_at TIMESTAMPTZ,
    UNIQUE (tenant_id, organization_id, certificate_id, attempted_at),
    CHECK (btrim(renewal_reason) <> '')
);

-- Index for renewal attempt lookup by certificate and time range
CREATE INDEX IF NOT EXISTS certificate_renewal_attempt_cert_idx
    ON certificate_renewal_attempt (tenant_id, organization_id, certificate_id, attempted_at DESC);

-- Index for authority scope lookup
CREATE INDEX IF NOT EXISTS certificate_renewal_attempt_authority_idx
    ON certificate_renewal_attempt (authority_tenant_id, authority_organization_id, attempted_at);

-- Renewal policy: tenant/organization isolation
ALTER TABLE certificate_renewal_attempt ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_renewal_attempt FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS certificate_renewal_attempt_tenant_isolation ON certificate_renewal_attempt;
CREATE POLICY certificate_renewal_attempt_tenant_isolation ON certificate_renewal_attempt USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

-- Enhanced certificate_record: add renewal tracking fields
ALTER TABLE certificate_record ADD COLUMN IF NOT EXISTS renewal_count BIGINT NOT NULL DEFAULT 0 CHECK (renewal_count >= 0);

ALTER TABLE certificate_record ADD COLUMN IF NOT EXISTS last_renewal_attempt TIMESTAMPTZ;

ALTER TABLE certificate_record ADD COLUMN IF NOT EXISTS renewal_authority_token BYTEA;

-- Index for renewal-optimized queries
CREATE INDEX certificate_record_renewal_idx
    ON certificate_record (tenant_id, organization_id, renewal_count DESC, last_renewal_attempt DESC);

COMMIT;