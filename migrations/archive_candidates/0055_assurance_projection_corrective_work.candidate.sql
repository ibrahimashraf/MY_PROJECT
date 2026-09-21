-- integin Work-Order Field Package: assurance projection and corrective-work links.
-- CANDIDATE ONLY: do not apply without a verified backup and isolated SQL review.
-- Builds on 0009 inspection_record, 0013 certificate lifecycle, 0053 asset_entitlement.

BEGIN;

-- assurance_projection: read-only projection of certificate authority status.
-- Derived from certificate_record and inspection_record; never mutates authority.
CREATE TABLE IF NOT EXISTS assurance_projection (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    certificate_id TEXT NOT NULL,
    inspection_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    asset_type TEXT NOT NULL,
    certificate_number TEXT NOT NULL,
    certificate_status TEXT NOT NULL,
    inspection_status TEXT NOT NULL,
    inspection_verdict TEXT NOT NULL,
    inspector_id TEXT NOT NULL,
    issued_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    superseded_at TIMESTAMPTZ,
    projection_hash TEXT NOT NULL,
    projected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, certificate_id)
);

-- corrective_work: actionable work items linked to inspection findings.
CREATE TABLE IF NOT EXISTS corrective_work (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    inspection_id TEXT NOT NULL,
    finding_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('ADVISORY', 'MINOR', 'MAJOR', 'CRITICAL')),
    description TEXT NOT NULL,
    required_by TIMESTAMPTZ,
    assigned_to TEXT,
    status TEXT NOT NULL CHECK (status IN ('OPEN', 'IN_PROGRESS', 'COMPLETED', 'VERIFIED', 'CLOSED')),
    completed_at TIMESTAMPTZ,
    verified_at TIMESTAMPTZ,
    verified_by TEXT,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, inspection_id, finding_id),
    FOREIGN KEY (tenant_id, organization_id, inspection_id)
        REFERENCES inspection_record (tenant_id, organization_id, id) ON DELETE RESTRICT,
    CHECK (
        (status = 'COMPLETED' AND completed_at IS NOT NULL)
        OR (status != 'COMPLETED')
    ),
    CHECK (
        (status = 'VERIFIED' AND verified_at IS NOT NULL AND verified_by IS NOT NULL)
        OR (status != 'VERIFIED')
    )
);

-- Indexes for lookup patterns.
CREATE INDEX assurance_projection_certificate_idx
    ON assurance_projection (tenant_id, organization_id, certificate_id);

CREATE INDEX assurance_projection_inspection_idx
    ON assurance_projection (tenant_id, organization_id, inspection_id);

CREATE INDEX assurance_projection_asset_idx
    ON assurance_projection (tenant_id, organization_id, asset_id, certificate_status);

CREATE INDEX assurance_projection_status_idx
    ON assurance_projection (tenant_id, organization_id, certificate_status, projected_at);

CREATE INDEX corrective_work_inspection_idx
    ON corrective_work (tenant_id, organization_id, inspection_id, status);

CREATE INDEX corrective_work_asset_idx
    ON corrective_work (tenant_id, organization_id, asset_id, status);

CREATE INDEX corrective_work_assigned_idx
    ON corrective_work (tenant_id, organization_id, assigned_to, status)
    WHERE assigned_to IS NOT NULL;

CREATE INDEX corrective_work_severity_idx
    ON corrective_work (tenant_id, organization_id, severity, status);

CREATE INDEX corrective_work_required_by_idx
    ON corrective_work (tenant_id, organization_id, required_by, status)
    WHERE required_by IS NOT NULL;

-- Forced Row Level Security: tenant + organization isolation.
ALTER TABLE assurance_projection ENABLE ROW LEVEL SECURITY;
ALTER TABLE assurance_projection FORCE ROW LEVEL SECURITY;
ALTER TABLE corrective_work ENABLE ROW LEVEL SECURITY;
ALTER TABLE corrective_work FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS assurance_projection_tenant_organization_isolation ON assurance_projection;
CREATE POLICY assurance_projection_tenant_organization_isolation ON assurance_projection
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

DROP POLICY IF EXISTS corrective_work_tenant_organization_isolation ON corrective_work;
CREATE POLICY corrective_work_tenant_organization_isolation ON corrective_work
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

COMMIT;
