-- INTEGIN Stage 0 Work-Order Foundation
-- CANDIDATE ONLY: do not apply without explicit migration authorization.
-- Identifier convention follows the existing work-package persistence migration:
-- tenant_id and organization_id are TEXT and are set in transaction-local context.

BEGIN;

CREATE TABLE IF NOT EXISTS work_order (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    client_id TEXT NOT NULL,
    job_number TEXT NOT NULL,
    request_state TEXT NOT NULL,
    execution_state TEXT NOT NULL,
    commercial_state TEXT NOT NULL,
    certificate_state TEXT NOT NULL,
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by TEXT NOT NULL,
    updated_by TEXT NOT NULL,
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, job_number)
);

CREATE TABLE IF NOT EXISTS work_order_scope_item (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    client_id TEXT NOT NULL,
    location_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    asset_type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, work_order_id, asset_id),
    FOREIGN KEY (tenant_id, organization_id, work_order_id)
        REFERENCES work_order (tenant_id, organization_id, id)
);

CREATE TABLE IF NOT EXISTS work_order_assignment (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    inspector_id TEXT NOT NULL,
    state TEXT NOT NULL,
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    effective_from TIMESTAMPTZ NOT NULL,
    effective_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by TEXT NOT NULL,
    updated_by TEXT NOT NULL,
    UNIQUE (tenant_id, organization_id, id),
    FOREIGN KEY (tenant_id, organization_id, work_order_id)
        REFERENCES work_order (tenant_id, organization_id, id),
    CHECK (effective_until IS NULL OR effective_until > effective_from)
);

CREATE TABLE IF NOT EXISTS work_order_assignment_scope (
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    assignment_id TEXT NOT NULL,
    scope_item_id TEXT NOT NULL,
    PRIMARY KEY (tenant_id, organization_id, assignment_id, scope_item_id),
    FOREIGN KEY (tenant_id, organization_id, assignment_id)
        REFERENCES work_order_assignment (tenant_id, organization_id, id),
    FOREIGN KEY (tenant_id, organization_id, scope_item_id)
        REFERENCES work_order_scope_item (tenant_id, organization_id, id)
);

CREATE TABLE IF NOT EXISTS work_order_operation (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    operation_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    operation_type TEXT NOT NULL,
    aggregate_id TEXT,
    expected_revision BIGINT NOT NULL CHECK (expected_revision > 0),
    resulting_revision BIGINT,
    status TEXT NOT NULL,
    conflict_code TEXT,
    receipt JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    UNIQUE (tenant_id, organization_id, idempotency_key),
    UNIQUE (tenant_id, organization_id, operation_id)
);

CREATE TABLE IF NOT EXISTS work_order_submission_segment (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    assignment_id TEXT NOT NULL,
    submitted_by TEXT NOT NULL,
    state TEXT NOT NULL,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    inspection_ids JSONB NOT NULL,
    UNIQUE (tenant_id, organization_id, id),
    FOREIGN KEY (tenant_id, organization_id, work_order_id)
        REFERENCES work_order (tenant_id, organization_id, id),
    FOREIGN KEY (tenant_id, organization_id, assignment_id)
        REFERENCES work_order_assignment (tenant_id, organization_id, id)
);

CREATE TABLE IF NOT EXISTS work_order_provisional_record (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    local_id TEXT NOT NULL,
    record_kind TEXT NOT NULL,
    work_order_id TEXT,
    client_id TEXT,
    canonical_id TEXT,
    reconcile_state TEXT NOT NULL,
    candidate_fingerprint TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reconciled_at TIMESTAMPTZ,
    reconciled_by TEXT,
    UNIQUE (tenant_id, organization_id, local_id),
    UNIQUE (tenant_id, organization_id, id)
);

CREATE TABLE IF NOT EXISTS work_order_state_event (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    previous_state JSONB,
    new_state JSONB,
    reason_code TEXT,
    operation_id TEXT,
    correlation_id TEXT,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, id),
    FOREIGN KEY (tenant_id, organization_id, work_order_id)
        REFERENCES work_order (tenant_id, organization_id, id)
);

CREATE INDEX work_order_scope_lookup
    ON work_order_scope_item (tenant_id, organization_id, work_order_id, client_id, location_id);
CREATE INDEX work_order_assignment_lookup
    ON work_order_assignment (tenant_id, organization_id, work_order_id, inspector_id, state);
CREATE INDEX work_order_operation_lookup
    ON work_order_operation (tenant_id, organization_id, aggregate_id, status);
CREATE INDEX work_order_event_lookup
    ON work_order_state_event (tenant_id, organization_id, work_order_id, occurred_at);

-- RLS uses the established repository settings and is forced for table owners.
DO $$
DECLARE
    table_name TEXT;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'work_order', 'work_order_scope_item', 'work_order_assignment',
        'work_order_assignment_scope', 'work_order_operation',
        'work_order_submission_segment', 'work_order_provisional_record',
        'work_order_state_event'
    ] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', table_name);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', table_name);
        EXECUTE format($policy$
            DROP POLICY IF EXISTS %I ON %I;
            CREATE POLICY %I ON %I
            USING (
                tenant_id = current_setting('integin.tenant_id', true)
                AND organization_id = current_setting('integin.organization_id', true)
            )
            WITH CHECK (
                tenant_id = current_setting('integin.tenant_id', true)
                AND organization_id = current_setting('integin.organization_id', true)
            );
        $policy$, table_name || '_tenant_isolation', table_name, table_name || '_tenant_isolation', table_name);
    END LOOP;
END $$;

COMMIT;
