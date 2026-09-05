-- Drop dependent foreign keys
ALTER TABLE authority_package DROP CONSTRAINT authority_package_device_id_fkey;
ALTER TABLE sync_device_state DROP CONSTRAINT sync_device_state_device_id_fkey;
ALTER TABLE sync_receipt DROP CONSTRAINT sync_receipt_device_id_fkey;
ALTER TABLE sync_held_transaction DROP CONSTRAINT sync_held_transaction_device_id_fkey;

-- Drop composite primary keys
ALTER TABLE device_registry DROP CONSTRAINT device_registry_pkey;
ALTER TABLE sync_device_state DROP CONSTRAINT sync_device_state_pkey;

-- Drop tenant-scoped unique constraints
ALTER TABLE sync_receipt DROP CONSTRAINT sync_receipt_tenant_id_device_id_sequence_number_key;

-- Restore original primary keys
ALTER TABLE device_registry ADD PRIMARY KEY (device_id);
ALTER TABLE sync_device_state ADD PRIMARY KEY (device_id);

-- Restore original unique constraints
ALTER TABLE sync_device_state ADD CONSTRAINT sync_device_state_tenant_id_device_id_key UNIQUE (tenant_id, device_id);
ALTER TABLE sync_receipt ADD CONSTRAINT sync_receipt_device_id_sequence_number_key UNIQUE (device_id, sequence_number);

-- Restore original foreign keys
ALTER TABLE authority_package ADD CONSTRAINT authority_package_device_id_fkey FOREIGN KEY (device_id) REFERENCES device_registry(device_id);
ALTER TABLE sync_device_state ADD CONSTRAINT sync_device_state_device_id_fkey FOREIGN KEY (device_id) REFERENCES device_registry(device_id);
ALTER TABLE sync_receipt ADD CONSTRAINT sync_receipt_device_id_fkey FOREIGN KEY (device_id) REFERENCES device_registry(device_id);
ALTER TABLE sync_held_transaction ADD CONSTRAINT sync_held_transaction_device_id_fkey FOREIGN KEY (device_id) REFERENCES device_registry(device_id);
