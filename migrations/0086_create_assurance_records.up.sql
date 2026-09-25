SET lock_timeout = '2s';

CREATE TABLE IF NOT EXISTS assurance_records (
    id                UUID        NOT NULL DEFAULT gen_random_uuid(),
    tenant_id         TEXT        NOT NULL,
    asset_id          TEXT        NOT NULL,
    PRIMARY KEY (tenant_id, id),
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
    tenant_id         TEXT        NOT NULL,
    asset_id          TEXT        NOT NULL,
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
ALTER TABLE current_assurance_states
    ADD CONSTRAINT fk_current_record FOREIGN KEY (tenant_id, record_id) REFERENCES assurance_records(tenant_id, id) NOT VALID;

-- Validate FKs in a separate pass (does not block writes to parent tables)
ALTER TABLE current_assurance_states VALIDATE CONSTRAINT fk_current_record;

ALTER TABLE assurance_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE assurance_records FORCE ROW LEVEL SECURITY;
ALTER TABLE current_assurance_states ENABLE ROW LEVEL SECURITY;
ALTER TABLE current_assurance_states FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_records ON assurance_records
    FOR ALL USING (tenant_id = NULLIF(current_setting('integin.tenant_id', true), ''));

CREATE POLICY tenant_isolation_current ON current_assurance_states
    FOR ALL USING (tenant_id = NULLIF(current_setting('integin.tenant_id', true), ''));

GRANT SELECT, INSERT, UPDATE ON assurance_records TO integin_test_runtime;
GRANT SELECT, INSERT, UPDATE ON current_assurance_states TO integin_test_runtime;
