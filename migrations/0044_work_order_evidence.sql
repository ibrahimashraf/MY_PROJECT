-- INTEGIN Ticket 03 D7-6: work-order evidence reference (URL, not blob).
-- CANDIDATE ONLY: do not apply without verified pre-apply backup and isolated up/down review.
-- Design: docs/architecture/INTEGIN_WORK_ORDER_FIELD_PACKAGE_CONTRACT_2026-09-01.md:78
-- Stores external object reference only; content_hash is SHA-256 hex, reference_url is external store URL.
-- Follows 0009 TEXT ids + composite FK and 0010 combined tenant/organization RLS patterns.
-- Candidate style: disposable isolated apply only on integin_repo_test@15432 (createdb -> psql -f -> dropdb), never pilot.
-- Do not mount routes, change package enforcement, OIDC/OpenBao, or access private material via this migration.

BEGIN;

CREATE TABLE IF NOT EXISTS work_order_evidence (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    content_hash TEXT NOT NULL CONSTRAINT work_order_evidence_content_hash_ck
        CHECK (content_hash ~ '^[a-fA-F0-9]{64}$'),
    reference_url TEXT NOT NULL CONSTRAINT work_order_evidence_reference_url_ck
        CHECK (reference_url ~ '^https?://' AND octet_length(reference_url) <= 2048 AND btrim(reference_url) <> ''),
    created_by TEXT NOT NULL CHECK (btrim(created_by) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, id),
    CONSTRAINT work_order_evidence_work_order_fk
        FOREIGN KEY (tenant_id, organization_id, work_order_id)
        REFERENCES work_order (tenant_id, organization_id, id)
        ON DELETE CASCADE,
    CHECK (btrim(tenant_id) <> ''),
    CHECK (btrim(organization_id) <> ''),
    CHECK (btrim(id) <> ''),
    CHECK (btrim(work_order_id) <> '')
);

CREATE INDEX IF NOT EXISTS work_order_evidence_work_order_idx
    ON work_order_evidence (tenant_id, organization_id, work_order_id, created_at);

CREATE INDEX IF NOT EXISTS work_order_evidence_content_hash_idx
    ON work_order_evidence (tenant_id, organization_id, content_hash);

-- Least-privilege runtime grants (mirrors 0010:5-15, no BYPASSRLS).
REVOKE ALL ON TABLE work_order_evidence FROM PUBLIC, integin_test_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE work_order_evidence TO integin_test_runtime;

ALTER TABLE work_order_evidence ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_evidence FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS work_order_evidence_tenant_organization_isolation ON work_order_evidence;
CREATE POLICY work_order_evidence_tenant_organization_isolation ON work_order_evidence
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

COMMIT;

