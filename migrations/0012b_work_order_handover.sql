-- integin Ticket 03 D7-5: work-order handover placeholder.
-- Planned persistence for handover lifecycle requested/approved/transferred.
-- FKs from_assignment_id / to_assignment_id reference work_order_assignment (tenant_id, organization_id, id) ON DELETE RESTRICT.
-- Follows 0009 TEXT ids + composite FK and 0010 combined tenant/organization RLS patterns.
-- Candidate style: disposable isolated apply only, never pilot. Requires verified backup before isolated exercise.

BEGIN;

CREATE TABLE IF NOT EXISTS work_order_handover (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    from_assignment_id TEXT NOT NULL,
    to_assignment_id TEXT NOT NULL,
    state TEXT NOT NULL CONSTRAINT work_order_handover_state_ck
        CHECK (state IN ('requested','approved','transferred')),
    revision BIGINT NOT NULL CHECK (revision > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, id),
    CONSTRAINT work_order_handover_work_order_fk
        FOREIGN KEY (tenant_id, organization_id, work_order_id)
        REFERENCES work_order (tenant_id, organization_id, id)
        ON DELETE CASCADE,
    CONSTRAINT work_order_handover_from_assignment_fk
        FOREIGN KEY (tenant_id, organization_id, from_assignment_id)
        REFERENCES work_order_assignment (tenant_id, organization_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT work_order_handover_to_assignment_fk
        FOREIGN KEY (tenant_id, organization_id, to_assignment_id)
        REFERENCES work_order_assignment (tenant_id, organization_id, id)
        ON DELETE RESTRICT,
    CHECK (btrim(tenant_id) <> ''),
    CHECK (btrim(organization_id) <> ''),
    CHECK (btrim(id) <> ''),
    CHECK (btrim(work_order_id) <> ''),
    CHECK (btrim(from_assignment_id) <> ''),
    CHECK (btrim(to_assignment_id) <> '')
);

CREATE INDEX IF NOT EXISTS work_order_handover_work_order_idx
    ON work_order_handover (tenant_id, organization_id, work_order_id, created_at);

-- Least-privilege runtime grants (mirrors 0010:5-15, no BYPASSRLS).
REVOKE ALL ON TABLE work_order_handover FROM PUBLIC, integin_test_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE work_order_handover TO integin_test_runtime;

ALTER TABLE work_order_handover ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_handover FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS work_order_handover_tenant_organization_isolation ON work_order_handover;
CREATE POLICY work_order_handover_tenant_organization_isolation ON work_order_handover
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

COMMIT;

