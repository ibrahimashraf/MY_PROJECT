CREATE TABLE IF NOT EXISTS device_registry (
    device_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    key_id TEXT NOT NULL,
    public_key BYTEA NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('PENDING', 'TRUSTED', 'RESTRICTED', 'LOCKED', 'REVOKED', 'RETIRED')),
    authority_epoch BIGINT NOT NULL CHECK (authority_epoch > 0),
    enrolled_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    revocation_reason TEXT,
    UNIQUE (tenant_id, key_id)
);

CREATE INDEX IF NOT EXISTS device_registry_tenant_idx
    ON device_registry (tenant_id, organization_id, state);

CREATE TABLE IF NOT EXISTS authority_package (
    authority_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    device_id TEXT NOT NULL REFERENCES device_registry(device_id),
    user_id TEXT NOT NULL,
    authority_epoch BIGINT NOT NULL CHECK (authority_epoch > 0),
    scopes JSONB NOT NULL DEFAULT '[]'::jsonb,
    procedure_version TEXT NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    signature BYTEA NOT NULL,
    revoked_at TIMESTAMPTZ,
    CHECK (expires_at > issued_at)
);

CREATE INDEX IF NOT EXISTS authority_package_active_idx
    ON authority_package (tenant_id, organization_id, device_id, authority_epoch, expires_at)
    WHERE revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS sync_device_state (
    device_id TEXT PRIMARY KEY REFERENCES device_registry(device_id),
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    last_accepted_sequence BIGINT NOT NULL DEFAULT 0 CHECK (last_accepted_sequence >= 0),
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE (tenant_id, device_id)
);

CREATE TABLE IF NOT EXISTS sync_receipt (
    transaction_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    device_id TEXT NOT NULL REFERENCES device_registry(device_id),
    user_id TEXT NOT NULL,
    sequence_number BIGINT NOT NULL CHECK (sequence_number > 0),
    operation TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    payload_hash TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('APPLIED', 'DUPLICATE', 'QUEUED', 'HELD', 'REJECTED', 'CONFLICT', 'SECURITY_FAILURE')),
    reason TEXT NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL,
    UNIQUE (device_id, sequence_number)
);

CREATE INDEX IF NOT EXISTS sync_receipt_tenant_time_idx
    ON sync_receipt (tenant_id, organization_id, received_at);

CREATE TABLE IF NOT EXISTS sync_held_transaction (
    transaction_id TEXT PRIMARY KEY REFERENCES sync_receipt(transaction_id),
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    device_id TEXT NOT NULL REFERENCES device_registry(device_id),
    sequence_number BIGINT NOT NULL CHECK (sequence_number > 0),
    expected_sequence BIGINT NOT NULL CHECK (expected_sequence > 0),
    envelope JSONB NOT NULL,
    first_held_at TIMESTAMPTZ NOT NULL,
    last_attempt_at TIMESTAMPTZ NOT NULL,
    last_error TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS sync_held_replay_idx
    ON sync_held_transaction (tenant_id, organization_id, device_id, expected_sequence);

ALTER TABLE device_registry ENABLE ROW LEVEL SECURITY;
ALTER TABLE authority_package ENABLE ROW LEVEL SECURITY;
ALTER TABLE sync_device_state ENABLE ROW LEVEL SECURITY;
ALTER TABLE sync_receipt ENABLE ROW LEVEL SECURITY;
ALTER TABLE sync_held_transaction ENABLE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = current_schema() AND tablename = 'device_registry' AND policyname = 'device_registry_tenant_isolation') THEN
        CREATE POLICY device_registry_tenant_isolation ON device_registry
            USING (tenant_id = current_setting('integin.tenant_id', true));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = current_schema() AND tablename = 'authority_package' AND policyname = 'authority_package_tenant_isolation') THEN
        CREATE POLICY authority_package_tenant_isolation ON authority_package
            USING (tenant_id = current_setting('integin.tenant_id', true));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = current_schema() AND tablename = 'sync_device_state' AND policyname = 'sync_device_state_tenant_isolation') THEN
        CREATE POLICY sync_device_state_tenant_isolation ON sync_device_state
            USING (tenant_id = current_setting('integin.tenant_id', true));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = current_schema() AND tablename = 'sync_receipt' AND policyname = 'sync_receipt_tenant_isolation') THEN
        CREATE POLICY sync_receipt_tenant_isolation ON sync_receipt
            USING (tenant_id = current_setting('integin.tenant_id', true));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = current_schema() AND tablename = 'sync_held_transaction' AND policyname = 'sync_held_transaction_tenant_isolation') THEN
        CREATE POLICY sync_held_transaction_tenant_isolation ON sync_held_transaction
            USING (tenant_id = current_setting('integin.tenant_id', true));
    END IF;
END $$;
