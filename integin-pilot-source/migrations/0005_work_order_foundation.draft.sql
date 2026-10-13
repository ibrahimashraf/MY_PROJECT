-- integin Stage 0 Work-Order Foundation
-- DRAFT ONLY: do not apply without explicit migration authorization and independent review.
-- Database tenant key follows the current repository convention: organization_id.

BEGIN;

CREATE TABLE IF NOT EXISTS work_order (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    client_id uuid NOT NULL,
    job_number text NOT NULL,
    request_state text NOT NULL,
    execution_state text NOT NULL,
    commercial_state text NOT NULL,
    certificate_state text NOT NULL,
    revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    UNIQUE (organization_id, id),
    UNIQUE (organization_id, job_number)
);

CREATE TABLE IF NOT EXISTS work_order_scope_item (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    work_order_id uuid NOT NULL,
    client_id uuid NOT NULL,
    location_id uuid NOT NULL,
    asset_id uuid NOT NULL,
    asset_type text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, id),
    UNIQUE (organization_id, work_order_id, asset_id),
    FOREIGN KEY (organization_id, work_order_id)
        REFERENCES work_order (organization_id, id)
);

CREATE TABLE IF NOT EXISTS work_order_assignment (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    work_order_id uuid NOT NULL,
    inspector_id uuid NOT NULL,
    state text NOT NULL,
    revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
    effective_from timestamptz NOT NULL,
    effective_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    UNIQUE (organization_id, id),
    FOREIGN KEY (organization_id, work_order_id)
        REFERENCES work_order (organization_id, id),
    CHECK (effective_until IS NULL OR effective_until > effective_from)
);

CREATE TABLE IF NOT EXISTS work_order_assignment_scope (
    organization_id uuid NOT NULL,
    assignment_id uuid NOT NULL,
    scope_item_id uuid NOT NULL,
    PRIMARY KEY (organization_id, assignment_id, scope_item_id),
    FOREIGN KEY (organization_id, assignment_id)
        REFERENCES work_order_assignment (organization_id, id),
    FOREIGN KEY (organization_id, scope_item_id)
        REFERENCES work_order_scope_item (organization_id, id)
);

CREATE TABLE IF NOT EXISTS work_order_operation (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    operation_id text NOT NULL,
    idempotency_key text NOT NULL,
    request_hash text NOT NULL,
    operation_type text NOT NULL,
    aggregate_id uuid,
    expected_revision bigint NOT NULL CHECK (expected_revision > 0),
    resulting_revision bigint,
    status text NOT NULL,
    conflict_code text,
    receipt jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    UNIQUE (organization_id, idempotency_key),
    UNIQUE (organization_id, operation_id)
);

CREATE TABLE IF NOT EXISTS work_order_submission_segment (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    work_order_id uuid NOT NULL,
    assignment_id uuid NOT NULL,
    submitted_by uuid NOT NULL,
    state text NOT NULL,
    submitted_at timestamptz NOT NULL DEFAULT now(),
    revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
    inspection_ids jsonb NOT NULL,
    UNIQUE (organization_id, id),
    FOREIGN KEY (organization_id, work_order_id)
        REFERENCES work_order (organization_id, id),
    FOREIGN KEY (organization_id, assignment_id)
        REFERENCES work_order_assignment (organization_id, id)
);

CREATE TABLE IF NOT EXISTS work_order_provisional_record (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    local_id text NOT NULL,
    record_kind text NOT NULL,
    work_order_id uuid,
    client_id uuid,
    canonical_id uuid,
    reconcile_state text NOT NULL,
    candidate_fingerprint text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    reconciled_at timestamptz,
    reconciled_by uuid,
    UNIQUE (organization_id, local_id),
    UNIQUE (organization_id, id)
);

CREATE TABLE IF NOT EXISTS work_order_state_event (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    work_order_id uuid NOT NULL,
    actor_id uuid NOT NULL,
    event_type text NOT NULL,
    previous_state jsonb,
    new_state jsonb,
    reason_code text,
    operation_id text,
    correlation_id text,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, id),
    FOREIGN KEY (organization_id, work_order_id)
        REFERENCES work_order (organization_id, id)
);

CREATE INDEX work_order_scope_lookup
    ON work_order_scope_item (organization_id, work_order_id, client_id, location_id);
CREATE INDEX work_order_assignment_lookup
    ON work_order_assignment (organization_id, work_order_id, inspector_id, state);
CREATE INDEX work_order_operation_lookup
    ON work_order_operation (organization_id, aggregate_id, status);
CREATE INDEX work_order_event_lookup
    ON work_order_state_event (organization_id, work_order_id, occurred_at);

-- RLS enablement and policies are intentionally not included in this first draft until
-- the repository's exact organization-setting function and runtime role are confirmed.
-- The migration must not be considered executable until those policies are added and reviewed.

COMMIT;
