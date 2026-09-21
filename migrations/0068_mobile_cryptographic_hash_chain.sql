-- integin Hardening: Field Cryptographic Merkle Hash Chain Ledger
-- Stores tamper-evident cryptographic hash links for offline mobile event batches.

BEGIN;

CREATE TABLE IF NOT EXISTS mobile_sync_hash_chain (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    sequence_number BIGINT NOT NULL,
    transaction_id TEXT NOT NULL,
    previous_hash TEXT NOT NULL,
    payload_hash TEXT NOT NULL,
    chain_hash TEXT NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT mobile_sync_hash_chain_device_seq UNIQUE (tenant_id, organization_id, device_id, sequence_number)
);

CREATE INDEX IF NOT EXISTS mobile_sync_hash_chain_lookup_idx
    ON mobile_sync_hash_chain (tenant_id, organization_id, device_id, sequence_number DESC);

ALTER TABLE mobile_sync_hash_chain ENABLE ROW LEVEL SECURITY;
ALTER TABLE mobile_sync_hash_chain FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS mobile_sync_hash_chain_isolation ON mobile_sync_hash_chain;
CREATE POLICY mobile_sync_hash_chain_isolation ON mobile_sync_hash_chain
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );

COMMIT;
