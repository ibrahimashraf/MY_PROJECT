-- INTEGIN work-package persistence: approved immutable definitions and current device assignments.
-- Server-side sync enforcement must resolve the exact package/version/hash from this store.

CREATE TABLE IF NOT EXISTS work_package (
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    package_id TEXT NOT NULL,
    package_version INTEGER NOT NULL CHECK (package_version > 0),
    template_code TEXT NOT NULL,
    template_version INTEGER NOT NULL CHECK (template_version > 0),
    schema_version INTEGER NOT NULL CHECK (schema_version > 0),
    publication_state TEXT NOT NULL CHECK (publication_state IN ('draft', 'approved', 'retired', 'superseded')),
    package_hash TEXT NOT NULL CHECK (package_hash ~ '^sha256:[0-9a-f]{64}$'),
    definition JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_at TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, organization_id, package_id, package_version),
    UNIQUE (tenant_id, organization_id, package_hash),
    CHECK ((publication_state = 'approved' AND approved_at IS NOT NULL) OR publication_state <> 'approved')
);

CREATE INDEX IF NOT EXISTS work_package_lookup_idx
    ON work_package (tenant_id, organization_id, package_id, package_version DESC);

CREATE TABLE IF NOT EXISTS work_package_assignment (
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    inspection_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    package_id TEXT NOT NULL,
    package_version INTEGER NOT NULL CHECK (package_version > 0),
    authority_epoch BIGINT NOT NULL CHECK (authority_epoch > 0),
    expires_at TIMESTAMPTZ NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, inspection_id),
    FOREIGN KEY (tenant_id, organization_id, package_id, package_version)
        REFERENCES work_package (tenant_id, organization_id, package_id, package_version)
);

CREATE INDEX IF NOT EXISTS work_package_assignment_device_idx
    ON work_package_assignment (tenant_id, organization_id, device_id, expires_at);

ALTER TABLE work_package ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_package_assignment ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS work_package_tenant_isolation ON work_package;
CREATE POLICY work_package_tenant_isolation ON work_package
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

DROP POLICY IF EXISTS work_package_assignment_tenant_isolation ON work_package_assignment;
CREATE POLICY work_package_assignment_tenant_isolation ON work_package_assignment
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );
