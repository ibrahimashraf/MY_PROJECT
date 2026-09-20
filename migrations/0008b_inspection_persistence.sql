-- CANDIDATE ONLY. DO NOT APPLY without independent review and explicit migration execution evidence.
-- Depends on the Work-Order foundation candidate being present.

BEGIN;

-- Enables composite containment references from inspection records to existing Work-Order scope and assignment rows.
ALTER TABLE work_order_scope_item
    ADD CONSTRAINT work_order_scope_item_order_identity_unique
    UNIQUE (tenant_id, organization_id, work_order_id, id);

ALTER TABLE work_order_assignment
    ADD CONSTRAINT work_order_assignment_order_identity_unique
    UNIQUE (tenant_id, organization_id, work_order_id, id);

ALTER TABLE work_order_submission_segment
    ADD CONSTRAINT work_order_submission_segment_identity_unique
    UNIQUE (tenant_id, organization_id, id, work_order_id, assignment_id);

CREATE TABLE IF NOT EXISTS inspection_record (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    scope_item_id TEXT NOT NULL,
    assignment_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    inspector_id TEXT NOT NULL,
    lifecycle_state TEXT NOT NULL CHECK (lifecycle_state IN ('SCHEDULED', 'ASSIGNED', 'IN_PROGRESS', 'COMPLETED', 'PENDING_REVIEW', 'APPROVED', 'REJECTED', 'CLOSED')),
    revision BIGINT NOT NULL CHECK (revision > 0),
    finalization_state TEXT NOT NULL CHECK (finalization_state IN ('OPEN', 'SUBMITTED', 'FINALIZED', 'VOIDED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by TEXT NOT NULL,
    updated_by TEXT NOT NULL,
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, id, work_order_id, assignment_id, scope_item_id),
    FOREIGN KEY (tenant_id, organization_id, work_order_id, scope_item_id)
        REFERENCES work_order_scope_item (tenant_id, organization_id, work_order_id, id),
    FOREIGN KEY (tenant_id, organization_id, work_order_id, assignment_id)
        REFERENCES work_order_assignment (tenant_id, organization_id, work_order_id, id)
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
    UNIQUE (tenant_id, organization_id, inspection_id),
    FOREIGN KEY (tenant_id, organization_id, submission_segment_id, work_order_id, assignment_id)
        REFERENCES work_order_submission_segment (tenant_id, organization_id, id, work_order_id, assignment_id),
    FOREIGN KEY (tenant_id, organization_id, inspection_id, work_order_id, assignment_id, scope_item_id)
        REFERENCES inspection_record (tenant_id, organization_id, id, work_order_id, assignment_id, scope_item_id),
    FOREIGN KEY (tenant_id, organization_id, assignment_id, scope_item_id)
        REFERENCES work_order_assignment_scope (tenant_id, organization_id, assignment_id, scope_item_id)
 );

CREATE INDEX inspection_record_membership_lookup
    ON inspection_record (tenant_id, organization_id, work_order_id, assignment_id, lifecycle_state, finalization_state);

ALTER TABLE inspection_record ENABLE ROW LEVEL SECURITY;
ALTER TABLE inspection_record FORCE ROW LEVEL SECURITY;
ALTER TABLE work_order_submission_item ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_submission_item FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS inspection_record_tenant_isolation ON inspection_record;
CREATE POLICY inspection_record_tenant_isolation ON inspection_record
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

DROP POLICY IF EXISTS work_order_submission_item_tenant_isolation ON work_order_submission_item;
CREATE POLICY work_order_submission_item_tenant_isolation ON work_order_submission_item
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;
