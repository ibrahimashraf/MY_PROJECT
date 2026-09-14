-- Down migration for 0082_device_composite_tenant_keys
BEGIN;

DROP INDEX IF EXISTS device_registry_tenant_device_uidx;
DROP INDEX IF EXISTS sync_device_state_tenant_device_uidx;
DROP INDEX IF EXISTS sync_held_transaction_tenant_tx_uidx;

COMMIT;
