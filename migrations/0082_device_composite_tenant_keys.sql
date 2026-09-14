-- Migration 0082: Formalize composite tenant boundary on device_registry and sync_device_state
-- Ensures primary keys / unique indexes strictly bind tenant_id to prevent multi-tenant collision.
BEGIN;

-- 1. Ensure composite unique indexes exist for strict multi-tenant boundaries
CREATE UNIQUE INDEX IF NOT EXISTS device_registry_tenant_device_uidx
    ON device_registry (tenant_id, device_id);

CREATE UNIQUE INDEX IF NOT EXISTS sync_device_state_tenant_device_uidx
    ON sync_device_state (tenant_id, device_id);

CREATE UNIQUE INDEX IF NOT EXISTS sync_held_transaction_tenant_tx_uidx
    ON sync_held_transaction (tenant_id, transaction_id);

COMMIT;
