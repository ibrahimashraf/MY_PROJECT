-- Drop dependent foreign keys
ALTER TABLE authority_package DROP CONSTRAINT authority_package_device_id_fkey;
ALTER TABLE sync_device_state DROP CONSTRAINT sync_device_state_device_id_fkey;
ALTER TABLE sync_receipt DROP CONSTRAINT sync_receipt_device_id_fkey;
ALTER TABLE sync_held_transaction DROP CONSTRAINT sync_held_transaction_device_id_fkey;

-- Drop primary keys that need to be redefined
ALTER TABLE device_registry DROP CONSTRAINT device_registry_pkey;
ALTER TABLE sync_device_state DROP CONSTRAINT sync_device_state_pkey;

-- Drop unique constraint on sync_receipt that needs tenant_id
ALTER TABLE sync_receipt DROP CONSTRAINT sync_receipt_device_id_sequence_number_key;
-- Also sync_device_state has a unique constraint that is redundant if we make it a PK
ALTER TABLE sync_device_state DROP CONSTRAINT sync_device_state_tenant_id_device_id_key;

-- Add new composite primary keys
ALTER TABLE device_registry ADD PRIMARY KEY (tenant_id, device_id);
ALTER TABLE sync_device_state ADD PRIMARY KEY (tenant_id, device_id);

-- Add new tenant-scoped unique constraints
ALTER TABLE sync_receipt ADD CONSTRAINT sync_receipt_tenant_id_device_id_sequence_number_key UNIQUE (tenant_id, device_id, sequence_number);

-- Re-add foreign keys with composite references
ALTER TABLE authority_package ADD CONSTRAINT authority_package_device_id_fkey FOREIGN KEY (tenant_id, device_id) REFERENCES device_registry(tenant_id, device_id);
ALTER TABLE sync_device_state ADD CONSTRAINT sync_device_state_device_id_fkey FOREIGN KEY (tenant_id, device_id) REFERENCES device_registry(tenant_id, device_id);
ALTER TABLE sync_receipt ADD CONSTRAINT sync_receipt_device_id_fkey FOREIGN KEY (tenant_id, device_id) REFERENCES device_registry(tenant_id, device_id);
ALTER TABLE sync_held_transaction ADD CONSTRAINT sync_held_transaction_device_id_fkey FOREIGN KEY (tenant_id, device_id) REFERENCES device_registry(tenant_id, device_id);
