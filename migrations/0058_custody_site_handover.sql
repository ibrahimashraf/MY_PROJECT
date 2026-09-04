-- INTEGIN Work-Order Field Package: custody chain and site handover transitions.
-- CANDIDATE ONLY: do not apply without a verified backup and isolated SQL review.
-- Builds on 0009 work_order, 0009 work_order_assignment, 0053 asset_entitlement.

BEGIN;

-- custody_chain: immutable append-only ledger tracking physical custody of an asset / work order.
CREATE TABLE custody_chain (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL DEFAULT '',
    location_id TEXT NOT NULL,
    site_name TEXT NOT NULL,
    custodian_id TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('CHECK_IN', 'CHECK_OUT', 'TRANSFER', 'RELOCATE', 'DISPOSE')),
    verification_hash TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    recorded_by TEXT NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, id),
    CHECK (btrim(tenant_id) <> ''),
    CHECK (btrim(organization_id) <> ''),
    CHECK (btrim(id) <> ''),
    CHECK (btrim(asset_id) <> ''),
    CHECK (btrim(location_id) <> ''),
    CHECK (btrim(site_name) <> ''),
    CHECK (btrim(custodian_id) <> ''),
    CHECK (btrim(verification_hash) <> ''),
    CHECK (btrim(recorded_by) <> '')
);

-- work_order_site_handover: formal handover request and approval tracking between assignments and sites.
CREATE TABLE work_order_site_handover (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    asset_id TEXT NOT NULL DEFAULT '',
    from_assignment_id TEXT NOT NULL,
    to_assignment_id TEXT NOT NULL,
    from_site_id TEXT NOT NULL,
    to_site_id TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('REQUESTED', 'ACKNOWLEDGED', 'APPROVED', 'TRANSFERRED', 'REJECTED', 'CANCELLED')),
    reason TEXT NOT NULL DEFAULT '',
    requested_by TEXT NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    acknowledged_by TEXT,
    acknowledged_at TIMESTAMPTZ,
    approved_by TEXT,
    approved_at TIMESTAMPTZ,
    transferred_by TEXT,
    transferred_at TIMESTAMPTZ,
    rejection_reason TEXT NOT NULL DEFAULT '',
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, id),
    FOREIGN KEY (tenant_id, organization_id, work_order_id)
        REFERENCES work_order (tenant_id, organization_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, organization_id, from_assignment_id)
        REFERENCES work_order_assignment (tenant_id, organization_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (tenant_id, organization_id, to_assignment_id)
        REFERENCES work_order_assignment (tenant_id, organization_id, id) ON DELETE RESTRICT,
    CHECK (from_assignment_id <> to_assignment_id),
    CHECK (
        (state = 'ACKNOWLEDGED' AND acknowledged_by IS NOT NULL AND acknowledged_at IS NOT NULL)
        OR (state <> 'ACKNOWLEDGED')
    ),
    CHECK (
        (state = 'APPROVED' AND approved_by IS NOT NULL AND approved_at IS NOT NULL)
        OR (state <> 'APPROVED')
    ),
    CHECK (
        (state = 'TRANSFERRED' AND transferred_by IS NOT NULL AND transferred_at IS NOT NULL)
        OR (state <> 'TRANSFERRED')
    ),
    CHECK (
        (state = 'REJECTED' AND btrim(rejection_reason) <> '')
        OR (state <> 'REJECTED')
    )
);

-- Indexes for efficient lookup.
CREATE INDEX custody_chain_asset_idx
    ON custody_chain (tenant_id, organization_id, asset_id, recorded_at DESC);

CREATE INDEX custody_chain_custodian_idx
    ON custody_chain (tenant_id, organization_id, custodian_id, recorded_at DESC);

CREATE INDEX custody_chain_wo_idx
    ON custody_chain (tenant_id, organization_id, work_order_id, recorded_at DESC);

CREATE INDEX work_order_site_handover_wo_idx
    ON work_order_site_handover (tenant_id, organization_id, work_order_id, state);

CREATE INDEX work_order_site_handover_asg_idx
    ON work_order_site_handover (tenant_id, organization_id, to_assignment_id, state);

-- Forced Row Level Security: tenant + organization isolation.
ALTER TABLE custody_chain ENABLE ROW LEVEL SECURITY;
ALTER TABLE custody_chain FORCE ROW LEVEL SECURITY;
ALTER TABLE work_order_site_handover ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_order_site_handover FORCE ROW LEVEL SECURITY;

CREATE POLICY custody_chain_tenant_organization_isolation ON custody_chain
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

CREATE POLICY work_order_site_handover_tenant_organization_isolation ON work_order_site_handover
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

COMMIT;
