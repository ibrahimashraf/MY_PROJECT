-- Migration 0077: Retention, Legal Hold, and Export Access Policy Engine
-- (P1.3). Four tenant-scoped registries driving disposition, legal hold, and
-- export approval; deletion_evidence_receipt is append-only with an immutable
-- trigger mirroring the audit_log pattern (0038).

BEGIN;

CREATE TABLE IF NOT EXISTS tenant_retention_policy (
    tenant_id      TEXT        NOT NULL,
    entity_type    TEXT        NOT NULL,
    retention_days INT         NOT NULL DEFAULT 365,
    action         TEXT        NOT NULL DEFAULT 'PURGE',
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by     TEXT        NOT NULL,
    PRIMARY KEY (tenant_id, entity_type),
    CHECK (btrim(tenant_id) <> ''),
    CHECK (btrim(entity_type) <> ''),
    CHECK (retention_days >= 0),
    CHECK (action IN ('ARCHIVE', 'PURGE'))
);

CREATE TABLE IF NOT EXISTS legal_hold_registry (
    id          TEXT        NOT NULL,
    tenant_id   TEXT        NOT NULL,
    entity_type TEXT        NOT NULL,
    entity_id   TEXT        NOT NULL,
    reason      TEXT        NOT NULL,
    placed_by   TEXT        NOT NULL,
    placed_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    status      TEXT        NOT NULL DEFAULT 'ACTIVE',
    released_by TEXT,
    released_at TIMESTAMPTZ,
    PRIMARY KEY (id),
    UNIQUE (tenant_id, entity_type, entity_id, id),
    CHECK (btrim(id) <> ''),
    CHECK (btrim(tenant_id) <> ''),
    CHECK (btrim(entity_type) <> ''),
    CHECK (btrim(entity_id) <> ''),
    CHECK (btrim(reason) <> ''),
    CHECK (btrim(placed_by) <> ''),
    CHECK (status IN ('ACTIVE', 'RELEASED')),
    CHECK (status = 'RELEASED' OR released_by IS NULL),
    CHECK (status = 'RELEASED' OR released_at IS NULL),
    CHECK (released_at IS NULL OR released_at >= placed_at)
);

CREATE TABLE IF NOT EXISTS export_approval_registry (
    id               TEXT        NOT NULL,
    tenant_id        TEXT        NOT NULL,
    export_id        TEXT        NOT NULL,
    requested_by     TEXT        NOT NULL,
    approved_by      TEXT,
    status           TEXT        NOT NULL DEFAULT 'PENDING',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    decided_at       TIMESTAMPTZ,
    rejection_reason TEXT,
    PRIMARY KEY (id),
    UNIQUE (tenant_id, export_id),
    CHECK (btrim(id) <> ''),
    CHECK (btrim(tenant_id) <> ''),
    CHECK (btrim(export_id) <> ''),
    CHECK (btrim(requested_by) <> ''),
    CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED')),
    CHECK (decided_at IS NULL OR decided_at >= created_at),
    CHECK (status = 'PENDING' OR decided_at IS NOT NULL),
    CHECK (status <> 'REJECTED' OR coalesce(btrim(rejection_reason), '') <> '')
);

CREATE TABLE IF NOT EXISTS deletion_evidence_receipt (
    id             TEXT        NOT NULL,
    tenant_id      TEXT        NOT NULL,
    entity_type    TEXT        NOT NULL,
    entity_id      TEXT        NOT NULL,
    actor_id       TEXT        NOT NULL,
    deleted_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    tombstone_hash TEXT        NOT NULL,
    metadata       JSONB,
    PRIMARY KEY (id),
    UNIQUE (tenant_id, id),
    CHECK (btrim(id) <> ''),
    CHECK (btrim(tenant_id) <> ''),
    CHECK (btrim(entity_type) <> ''),
    CHECK (btrim(entity_id) <> ''),
    CHECK (btrim(actor_id) <> ''),
    CHECK (btrim(tombstone_hash) <> '')
);

CREATE INDEX IF NOT EXISTS legal_hold_registry_active_idx
    ON legal_hold_registry (tenant_id, entity_type, entity_id, status);

CREATE INDEX IF NOT EXISTS export_approval_registry_status_idx
    ON export_approval_registry (tenant_id, status);

CREATE INDEX IF NOT EXISTS deletion_evidence_receipt_entity_idx
    ON deletion_evidence_receipt (tenant_id, entity_type, entity_id, deleted_at DESC);

ALTER TABLE tenant_retention_policy ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_retention_policy FORCE ROW LEVEL SECURITY;
ALTER TABLE legal_hold_registry ENABLE ROW LEVEL SECURITY;
ALTER TABLE legal_hold_registry FORCE ROW LEVEL SECURITY;
ALTER TABLE export_approval_registry ENABLE ROW LEVEL SECURITY;
ALTER TABLE export_approval_registry FORCE ROW LEVEL SECURITY;
ALTER TABLE deletion_evidence_receipt ENABLE ROW LEVEL SECURITY;
ALTER TABLE deletion_evidence_receipt FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_retention_policy_tenant_isolation ON tenant_retention_policy;
CREATE POLICY tenant_retention_policy_tenant_isolation ON tenant_retention_policy
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
    );

DROP POLICY IF EXISTS legal_hold_registry_tenant_isolation ON legal_hold_registry;
CREATE POLICY legal_hold_registry_tenant_isolation ON legal_hold_registry
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
    );

DROP POLICY IF EXISTS export_approval_registry_tenant_isolation ON export_approval_registry;
CREATE POLICY export_approval_registry_tenant_isolation ON export_approval_registry
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
    );

DROP POLICY IF EXISTS deletion_evidence_receipt_tenant_isolation ON deletion_evidence_receipt;
CREATE POLICY deletion_evidence_receipt_tenant_isolation ON deletion_evidence_receipt
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
    );

GRANT SELECT, INSERT, UPDATE, DELETE ON tenant_retention_policy TO integin_test_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON legal_hold_registry TO integin_test_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON export_approval_registry TO integin_test_runtime;
GRANT SELECT, INSERT ON deletion_evidence_receipt TO integin_test_runtime;

CREATE OR REPLACE FUNCTION deletion_evidence_receipt_immutable() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'deletion_evidence_receipt is append-only: UPDATE/DELETE not allowed';
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS deletion_evidence_receipt_immutable_trigger ON deletion_evidence_receipt;
CREATE TRIGGER deletion_evidence_receipt_immutable_trigger
    BEFORE UPDATE OR DELETE ON deletion_evidence_receipt
    FOR EACH ROW EXECUTE FUNCTION deletion_evidence_receipt_immutable();

COMMIT;
