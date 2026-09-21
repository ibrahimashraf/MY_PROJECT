-- integin Ticket 02-02: work-order persistence foundation.
-- Tenant isolation policies and runtime grants are a later separately gated slice.

-- Removed redundant ALTER; created_at defined in table definition
CREATE TABLE IF NOT EXISTS work_order (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    client_id TEXT NOT NULL,
    job_number TEXT NOT NULL,
    request_state TEXT NOT NULL CONSTRAINT work_order_request_state_ck
        CHECK (request_state IN ('draft', 'requested', 'accepted', 'cancelled')),
    execution_state TEXT NOT NULL CONSTRAINT work_order_execution_state_ck
        CHECK (execution_state IN ('ready', 'assigned', 'in_progress', 'partially_submitted', 'awaiting_client', 'awaiting_review', 'completed')),
    commercial_state TEXT NOT NULL CONSTRAINT work_order_commercial_state_ck
        CHECK (commercial_state IN ('not_ready_for_invoice', 'ready_for_office_confirmation', 'released_for_invoice', 'invoiced', 'closed')),
    certificate_state TEXT NOT NULL CONSTRAINT work_order_certificate_state_ck
        CHECK (certificate_state IN ('not_started', 'pending_validation', 'partially_issued', 'issued', 'needs_correction', 'revoked_or_superseded')),
    revision BIGINT NOT NULL CHECK (revision > 0),
    created_by TEXT NOT NULL,
    updated_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, job_number)
);

CREATE TABLE IF NOT EXISTS work_order_scope_item (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    client_id TEXT NOT NULL,
    location_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    asset_type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, id),
    CONSTRAINT work_order_scope_item_order_fk
        FOREIGN KEY (tenant_id, organization_id, work_order_id)
        REFERENCES work_order (tenant_id, organization_id, id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS work_order_scope_item_order_idx
    ON work_order_scope_item (tenant_id, organization_id, work_order_id, id);

CREATE TABLE IF NOT EXISTS work_order_assignment (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    inspector_id TEXT NOT NULL,
    state TEXT NOT NULL CONSTRAINT work_order_assignment_state_ck
        CHECK (state IN ('active', 'transferred', 'completed', 'revoked')),
    revision BIGINT NOT NULL CHECK (revision > 0),
    effective_from TIMESTAMPTZ NOT NULL,
    effective_until TIMESTAMPTZ,
    created_by TEXT NOT NULL,
    updated_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, id),
    CONSTRAINT work_order_assignment_work_order_fk
        FOREIGN KEY (tenant_id, organization_id, work_order_id)
        REFERENCES work_order (tenant_id, organization_id, id)
        ON DELETE CASCADE,
    CHECK (effective_until IS NULL OR effective_until >= effective_from)
);

CREATE INDEX IF NOT EXISTS work_order_assignment_order_idx
    ON work_order_assignment (tenant_id, organization_id, work_order_id, state);

CREATE TABLE IF NOT EXISTS work_order_assignment_scope (
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    assignment_id TEXT NOT NULL,
    scope_item_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, assignment_id, scope_item_id),
    CONSTRAINT work_order_assignment_scope_assignment_fk
        FOREIGN KEY (tenant_id, organization_id, assignment_id)
        REFERENCES work_order_assignment (tenant_id, organization_id, id)
        ON DELETE CASCADE,
    CONSTRAINT work_order_assignment_scope_item_fk
        FOREIGN KEY (tenant_id, organization_id, scope_item_id)
        REFERENCES work_order_scope_item (tenant_id, organization_id, id)
        ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS inspection_record (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    scope_item_id TEXT NOT NULL,
    assignment_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    inspector_id TEXT NOT NULL,
    lifecycle_state TEXT NOT NULL CONSTRAINT inspection_record_lifecycle_state_ck
        CHECK (lifecycle_state IN ('PLANNED', 'IN_PROGRESS', 'COMPLETED', 'PENDING_REVIEW', 'CANCELLED')),
    revision BIGINT NOT NULL CHECK (revision > 0),
    finalization_state TEXT NOT NULL CONSTRAINT inspection_record_finalization_state_ck
        CHECK (finalization_state IN ('OPEN', 'SUBMITTED', 'VOIDED')),
    created_by TEXT NOT NULL,
    updated_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, id),
    UNIQUE (id),
    CONSTRAINT inspection_record_order_fk
        FOREIGN KEY (tenant_id, organization_id, work_order_id)
        REFERENCES work_order (tenant_id, organization_id, id)
        ON DELETE CASCADE,
    CONSTRAINT inspection_record_assignment_fk
        FOREIGN KEY (tenant_id, organization_id, assignment_id)
        REFERENCES work_order_assignment (tenant_id, organization_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT inspection_record_scope_item_fk
        FOREIGN KEY (tenant_id, organization_id, scope_item_id)
        REFERENCES work_order_scope_item (tenant_id, organization_id, id)
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS inspection_record_membership_idx
    ON inspection_record (tenant_id, organization_id, work_order_id, assignment_id, id, finalization_state);

CREATE TABLE IF NOT EXISTS work_order_submission_segment (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    assignment_id TEXT NOT NULL,
    submitted_by TEXT NOT NULL,
    state TEXT NOT NULL CONSTRAINT work_order_submission_segment_state_ck
        CHECK (state IN ('submitted', 'rejected')),
    revision BIGINT NOT NULL CHECK (revision > 0),
    inspection_ids JSONB NOT NULL CHECK (jsonb_typeof(inspection_ids) = 'array'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, id),
    CONSTRAINT work_order_submission_segment_order_fk
        FOREIGN KEY (tenant_id, organization_id, work_order_id)
        REFERENCES work_order (tenant_id, organization_id, id)
        ON DELETE CASCADE,
    CONSTRAINT work_order_submission_segment_assignment_fk
        FOREIGN KEY (tenant_id, organization_id, assignment_id)
        REFERENCES work_order_assignment (tenant_id, organization_id, id)
        ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS work_order_submission_item (
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    submission_segment_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    assignment_id TEXT NOT NULL,
    scope_item_id TEXT NOT NULL,
    inspection_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, submission_segment_id, inspection_id),
    CONSTRAINT work_order_submission_item_segment_fk
        FOREIGN KEY (tenant_id, organization_id, submission_segment_id)
        REFERENCES work_order_submission_segment (tenant_id, organization_id, id)
        ON DELETE CASCADE,
    CONSTRAINT work_order_submission_item_order_fk
        FOREIGN KEY (tenant_id, organization_id, work_order_id)
        REFERENCES work_order (tenant_id, organization_id, id)
        ON DELETE CASCADE,
    CONSTRAINT work_order_submission_item_assignment_fk
        FOREIGN KEY (tenant_id, organization_id, assignment_id)
        REFERENCES work_order_assignment (tenant_id, organization_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT work_order_submission_item_scope_fk
        FOREIGN KEY (tenant_id, organization_id, scope_item_id)
        REFERENCES work_order_scope_item (tenant_id, organization_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT work_order_submission_item_inspection_fk
        FOREIGN KEY (tenant_id, organization_id, inspection_id)
        REFERENCES inspection_record (tenant_id, organization_id, id)
        ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS work_order_operation (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    operation_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    operation_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    expected_revision BIGINT NOT NULL CHECK (expected_revision > 0),
    resulting_revision BIGINT NOT NULL CHECK (resulting_revision > 0),
    status TEXT NOT NULL CHECK (status IN ('accepted', 'rejected')),
    receipt JSONB NOT NULL CONSTRAINT work_order_operation_receipt_jsonb_ck
        CHECK (jsonb_typeof(receipt) = 'object'),
    completed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, operation_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS work_order_operation_idempotency_idx
    ON work_order_operation (tenant_id, organization_id, idempotency_key);

CREATE TABLE IF NOT EXISTS work_order_state_event (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    previous_state JSONB NOT NULL,
    new_state JSONB NOT NULL,
    operation_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, id),
    CONSTRAINT work_order_state_event_order_fk
        FOREIGN KEY (tenant_id, organization_id, work_order_id)
        REFERENCES work_order (tenant_id, organization_id, id)
        ON DELETE CASCADE,
    CONSTRAINT work_order_state_event_operation_fk
        FOREIGN KEY (tenant_id, organization_id, operation_id)
        REFERENCES work_order_operation (tenant_id, organization_id, operation_id)
        ON DELETE RESTRICT
        DEFERRABLE INITIALLY DEFERRED
);

ALTER TABLE work_order_state_event ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE INDEX IF NOT EXISTS work_order_state_event_order_idx
    ON work_order_state_event (tenant_id, organization_id, work_order_id, created_at);

CREATE TABLE IF NOT EXISTS work_order_provisional_record (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    local_id TEXT NOT NULL,
    record_kind TEXT NOT NULL CHECK (record_kind IN ('client', 'work_order', 'location', 'asset', 'inspection')),
    work_order_id TEXT,
    client_id TEXT,
    canonical_id TEXT,
    reconcile_state TEXT NOT NULL CONSTRAINT work_order_provisional_record_state_ck
        CHECK (reconcile_state IN ('pending', 'matched', 'created', 'conflict', 'rejected')),
    candidate_fingerprint TEXT NOT NULL,
    reconciled_at TIMESTAMPTZ,
    reconciled_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS work_order_provisional_record_local_idx
    ON work_order_provisional_record (tenant_id, organization_id, local_id);

CREATE INDEX IF NOT EXISTS work_order_lookup_idx
    ON work_order (tenant_id, organization_id, id);
