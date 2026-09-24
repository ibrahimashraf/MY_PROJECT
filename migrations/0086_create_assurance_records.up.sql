SET lock_timeout = '2s';

CREATE TABLE IF NOT EXISTS assurance_records (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    asset_id          UUID        NOT NULL,
    state             TEXT        NOT NULL,
    previous_state    TEXT,
    transition_guard  TEXT,
    lamport_clock     BIGINT      NOT NULL DEFAULT 1,
    revision          INT         NOT NULL DEFAULT 1,
    effective_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at        TIMESTAMPTZ,
    reasons           JSONB       NOT NULL DEFAULT '[]'::jsonb,
    evidence_chain    JSONB       NOT NULL DEFAULT '{}'::jsonb,
    evaluator_version TEXT        NOT NULL,
    rule_hash         TEXT        NOT NULL,
    config_hash       TEXT        NOT NULL,
    evaluated_by      JSONB       NOT NULL,
    signature         BYTEA,
    signed_by         TEXT,
    audit_ledger_cid  TEXT,
    is_offline_origin BOOLEAN     NOT NULL DEFAULT FALSE,
    trigger_event_id  UUID,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS current_assurance_states (
    tenant_id         UUID        NOT NULL,
    asset_id          UUID        NOT NULL,
    record_id         UUID        NOT NULL,
    state             TEXT        NOT NULL,
    lamport_clock     BIGINT      NOT NULL,
    revision          INT         NOT NULL,
    effective_at      TIMESTAMPTZ NOT NULL,
    expires_at        TIMESTAMPTZ,
    reasons           JSONB       NOT NULL DEFAULT '[]'::jsonb,
    evidence_chain    JSONB       NOT NULL DEFAULT '{}'::jsonb,
    is_signed         BOOLEAN     NOT NULL DEFAULT FALSE,
    audit_ledger_cid  TEXT,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (tenant_id, asset_id)
);

-- Zero-Downtime Safe DDL: Add FKs as NOT VALID to prevent ShareRowExclusiveLock on parent tables from blocking
ALTER TABLE assurance_records
    ADD CONSTRAINT fk_assurance_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) NOT VALID;

ALTER TABLE current_assurance_states
    ADD CONSTRAINT fk_current_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) NOT VALID;

ALTER TABLE current_assurance_states
    ADD CONSTRAINT fk_current_record FOREIGN KEY (record_id) REFERENCES assurance_records(id) NOT VALID;

-- Validate FKs in a separate pass (does not block writes to parent tables)
ALTER TABLE assurance_records VALIDATE CONSTRAINT fk_assurance_tenant;
ALTER TABLE current_assurance_states VALIDATE CONSTRAINT fk_current_tenant;
ALTER TABLE current_assurance_states VALIDATE CONSTRAINT fk_current_record;

ALTER TABLE assurance_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE current_assurance_states ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_records ON assurance_records
    FOR ALL USING (tenant_id = current_setting('app.current_tenant')::UUID);

CREATE POLICY tenant_isolation_current ON current_assurance_states
    FOR ALL USING (tenant_id = current_setting('app.current_tenant')::UUID);
