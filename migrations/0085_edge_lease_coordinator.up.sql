-- Migration: 0085_edge_lease_coordinator
-- Sovereign Edge Appliance lease lifecycle: exclusive sub-assembly leases,
-- overrun drafts, tombstone suppression, epoch log, revocations, Merkle log.
-- Invariants: composite tenant keys, FORCE RLS with NULLIF tenant isolation,
-- least-privilege grants to integin_test_runtime.

-- 1. Sub-Assembly Leases: exclusive mutex with overrun lifecycle
CREATE TABLE IF NOT EXISTS sub_assembly_leases (
    tenant_id           VARCHAR(64)  NOT NULL,
    organization_id     VARCHAR(64)  NOT NULL DEFAULT 'default',
    sub_assembly_id     VARCHAR(128) NOT NULL,
    parent_asset_did    VARCHAR(256) NOT NULL,
    assigned_device_id  VARCHAR(128) NOT NULL,
    inspector_id        VARCHAR(64)  NOT NULL,
    required_skill      VARCHAR(64)  NOT NULL,
    criticality_class   VARCHAR(32)  NOT NULL,
    lease_epoch         BIGINT       NOT NULL,
    lease_start         TIMESTAMPTZ  NOT NULL,
    valid_until         TIMESTAMPTZ  NOT NULL,
    state               VARCHAR(32)  NOT NULL DEFAULT 'ACTIVE',
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, sub_assembly_id)
);

ALTER TABLE sub_assembly_leases ENABLE ROW LEVEL SECURITY;
ALTER TABLE sub_assembly_leases FORCE ROW LEVEL SECURITY;

CREATE POLICY sub_assembly_leases_tenant_isolation ON sub_assembly_leases
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('integin.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('integin.tenant_id', true), ''));

CREATE INDEX IF NOT EXISTS idx_leases_sweep
ON sub_assembly_leases (tenant_id, state, valid_until)
WHERE state = 'ACTIVE';

-- 2. Draft Work Orders: overrun landing envelopes (idempotent per epoch)
CREATE TABLE IF NOT EXISTS draft_work_orders (
    tenant_id             VARCHAR(64)  NOT NULL,
    organization_id       VARCHAR(64)  NOT NULL DEFAULT 'default',
    draft_id              UUID         NOT NULL,
    sub_assembly_id       VARCHAR(128) NOT NULL,
    parent_asset_did      VARCHAR(256) NOT NULL,
    original_inspector_id VARCHAR(64)  NOT NULL,
    lease_epoch           BIGINT       NOT NULL,
    state                 VARCHAR(32)  NOT NULL,
    override_id           VARCHAR(64),
    created_at            TIMESTAMPTZ  NOT NULL,
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, draft_id),
    CONSTRAINT uq_subassembly_epoch UNIQUE (tenant_id, sub_assembly_id, lease_epoch)
);

ALTER TABLE draft_work_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE draft_work_orders FORCE ROW LEVEL SECURITY;

CREATE POLICY draft_work_orders_tenant_isolation ON draft_work_orders
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('integin.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('integin.tenant_id', true), ''));

-- 3. Tombstone Registry: suppression anchors with monotonic sequencing
CREATE TABLE IF NOT EXISTS lease_tombstones (
    tenant_id             VARCHAR(64)  NOT NULL,
    organization_id       VARCHAR(64)  NOT NULL DEFAULT 'default',
    sub_assembly_id       VARCHAR(128) NOT NULL,
    lease_epoch           BIGINT       NOT NULL,
    tombstone_sequence    BIGINT       NOT NULL,
    revoked_device_id     VARCHAR(128) NOT NULL,
    revocation_timestamp  TIMESTAMPTZ  NOT NULL,
    authorizing_did       VARCHAR(256) NOT NULL,
    reason                VARCHAR(64)  NOT NULL,
    PRIMARY KEY (tenant_id, sub_assembly_id)
);

ALTER TABLE lease_tombstones ENABLE ROW LEVEL SECURITY;
ALTER TABLE lease_tombstones FORCE ROW LEVEL SECURITY;

CREATE POLICY lease_tombstones_tenant_isolation ON lease_tombstones
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('integin.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('integin.tenant_id', true), ''));

-- 4. Authority Epoch Log & Device Revocations (offline bundle ingest target)
CREATE TABLE IF NOT EXISTS authority_epoch_log (
    tenant_id           VARCHAR(64)  NOT NULL,
    organization_id     VARCHAR(64)  NOT NULL DEFAULT 'default',
    epoch               BIGINT       NOT NULL,
    authority_root_did  VARCHAR(256) NOT NULL,
    applied_at          TIMESTAMPTZ  NOT NULL,
    bundle_digest       BYTEA        NOT NULL,
    PRIMARY KEY (tenant_id, epoch)
);

ALTER TABLE authority_epoch_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE authority_epoch_log FORCE ROW LEVEL SECURITY;

CREATE POLICY authority_epoch_log_tenant_isolation ON authority_epoch_log
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('integin.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('integin.tenant_id', true), ''));

CREATE TABLE IF NOT EXISTS device_revocation_registry (
    tenant_id           VARCHAR(64)  NOT NULL,
    organization_id     VARCHAR(64)  NOT NULL DEFAULT 'default',
    device_id           VARCHAR(128) NOT NULL,
    revoked_at_epoch    BIGINT       NOT NULL,
    reason              TEXT         NOT NULL,
    recorded_at         TIMESTAMPTZ  NOT NULL,
    PRIMARY KEY (tenant_id, device_id)
);

ALTER TABLE device_revocation_registry ENABLE ROW LEVEL SECURITY;
ALTER TABLE device_revocation_registry FORCE ROW LEVEL SECURITY;

CREATE POLICY device_revocation_registry_tenant_isolation ON device_revocation_registry
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('integin.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('integin.tenant_id', true), ''));

-- 5. Append-Only Merkle Audit Log (no DELETE grant: immutability by privilege)
CREATE TABLE IF NOT EXISTS merkle_audit_log (
    tenant_id           VARCHAR(64)  NOT NULL,
    organization_id     VARCHAR(64)  NOT NULL DEFAULT 'default',
    sequence_id         BIGSERIAL    NOT NULL,
    sub_assembly_id     VARCHAR(128) NOT NULL,
    lease_epoch         BIGINT       NOT NULL,
    authority_epoch     BIGINT       NOT NULL,
    event_type          VARCHAR(64)  NOT NULL,
    payload_hash        BYTEA        NOT NULL,
    prev_event_hash     BYTEA        NOT NULL,
    event_hash          BYTEA        NOT NULL,
    permanent_flag      BOOLEAN      NOT NULL DEFAULT TRUE,
    committed_at        TIMESTAMPTZ  NOT NULL,
    PRIMARY KEY (tenant_id, sequence_id)
);

ALTER TABLE merkle_audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE merkle_audit_log FORCE ROW LEVEL SECURITY;

CREATE POLICY merkle_audit_log_tenant_isolation ON merkle_audit_log
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('integin.tenant_id', true), ''))
    WITH CHECK (tenant_id = NULLIF(current_setting('integin.tenant_id', true), ''));

CREATE INDEX IF NOT EXISTS idx_merkle_tenant_seq
ON merkle_audit_log (tenant_id, sequence_id DESC);

-- Least-privilege grants (matches 0073/0075 patterns; merkle log is append-only)
GRANT SELECT, INSERT, UPDATE ON sub_assembly_leases TO integin_test_runtime;
GRANT SELECT, INSERT, UPDATE ON draft_work_orders TO integin_test_runtime;
GRANT SELECT, INSERT, UPDATE ON lease_tombstones TO integin_test_runtime;
GRANT SELECT, INSERT ON authority_epoch_log TO integin_test_runtime;
GRANT SELECT, INSERT, UPDATE ON device_revocation_registry TO integin_test_runtime;
GRANT SELECT, INSERT ON merkle_audit_log TO integin_test_runtime;
GRANT USAGE, SELECT ON SEQUENCE merkle_audit_log_sequence_id_seq TO integin_test_runtime;
