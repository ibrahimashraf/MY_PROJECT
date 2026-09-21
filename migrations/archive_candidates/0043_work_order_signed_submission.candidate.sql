-- integin Ticket 03 D7-4: work-order signed submission hardening.
-- Based on docs/architecture/integin_WORK_ORDER_FIELD_PACKAGE_CONTRACT_2026-09-01.md §6
-- and migrations/0009_work_order_persistence.sql:183-204 as base. Adds
-- payload_hash, signature, signing_key_id, replay guard (operation_id UNIQUE,
-- idempotency_key UNIQUE) and receipt JSONB hardening to work_order_operation.
-- Uses TEXT ids, organization_id (no org_id shorthand), and mirrors
-- migrations/0010_work_order_rls.sql least-privilege and RLS pattern.
-- Disposable isolated apply only; never pilot. Requires verified backup
-- before any isolated exercise, then dropdb.

BEGIN;

GRANT USAGE ON SCHEMA public TO integin_runtime;

-- Extend work_order_operation with signed-payload fields.
-- 0009 base already defines:
--   id TEXT, tenant_id TEXT, organization_id TEXT, operation_id TEXT,
--   idempotency_key TEXT, request_hash TEXT, operation_type TEXT, aggregate_id TEXT,
--   expected_revision BIGINT, resulting_revision BIGINT, status TEXT,
--   receipt JSONB CHECK (jsonb_typeof(receipt)='object'), completed_at, created_at,
--   PRIMARY KEY (tenant_id, organization_id, id),
--   UNIQUE (tenant_id, organization_id, operation_id),
--   UNIQUE INDEX work_order_operation_idempotency_idx (tenant_id, organization_id, idempotency_key)
-- D7-4 adds payload hash, detached signature and signing key for offline
-- Ed25519 verification and durable replay protection.

ALTER TABLE work_order_operation
    ADD COLUMN IF NOT EXISTS payload_hash TEXT NOT NULL CHECK (btrim(payload_hash) <> '');

ALTER TABLE work_order_operation
    ADD COLUMN IF NOT EXISTS signature TEXT NOT NULL CHECK (btrim(signature) <> '');

ALTER TABLE work_order_operation
    ADD COLUMN IF NOT EXISTS signing_key_id TEXT NOT NULL CHECK (btrim(signing_key_id) <> '');

-- Strengthen payload hash format when non-empty: hex sha256 (64 chars).
-- Separate constraint so IF NOT EXISTS column with default '' does not
-- block fresh DB where column is added as NOT NULL.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'work_order_operation_payload_hash_hex_ck'
    ) THEN
        ALTER TABLE work_order_operation
            ADD CONSTRAINT work_order_operation_payload_hash_hex_ck
            CHECK (payload_hash ~ '^[0-9a-f]{64}$' OR payload_hash = '');
    END IF;
END $$;

-- Replay guard: tenant-scoped unique operation_id and idempotency_key.
-- 0009 already creates UNIQUE (tenant_id, organization_id, operation_id)
-- and UNIQUE INDEX work_order_operation_idempotency_idx; re-assert
-- idempotently so a DB created without 0009 still hardens replay.
CREATE UNIQUE INDEX IF NOT EXISTS work_order_operation_idempotency_idx
    ON work_order_operation (tenant_id, organization_id, idempotency_key);

CREATE UNIQUE INDEX IF NOT EXISTS work_order_operation_operation_id_idx
    ON work_order_operation (tenant_id, organization_id, operation_id);

-- Receipt JSONB hardening: ensure receipt is an object (already in 0009),
-- re-assert idempotently.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'work_order_operation_receipt_jsonb_ck'
    ) THEN
        ALTER TABLE work_order_operation
            ADD CONSTRAINT work_order_operation_receipt_jsonb_ck
            CHECK (jsonb_typeof(receipt) = 'object');
    END IF;
END $$;

-- Least-privilege runtime grants mirror 0010 pattern (single-table scope).
REVOKE ALL ON TABLE work_order_operation FROM PUBLIC, integin_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE work_order_operation TO integin_runtime;

-- RLS: tenant/organization isolation matching 0010 (server-derived, never client-authoritative).
ALTER TABLE work_order_operation ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_operation FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS work_order_operation_tenant_organization_isolation ON work_order_operation;
CREATE POLICY work_order_operation_tenant_organization_isolation ON work_order_operation
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

COMMIT;
