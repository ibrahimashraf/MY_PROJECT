-- INTEGIN Work-Order Field Package: versioned form definitions, field catalog, and evidence policy.
-- CANDIDATE ONLY: do not apply without a verified backup and isolated SQL review.
-- Builds on 0012 certificate_template binding registry (nine-key catalog).

BEGIN;

-- form_version: immutable versioned form definitions for work-order field packages.
CREATE TABLE IF NOT EXISTS form_version (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    form_code TEXT NOT NULL,
    version BIGINT NOT NULL CHECK (version > 0),
    title TEXT NOT NULL,
    description TEXT,
    asset_type TEXT,
    status TEXT NOT NULL CHECK (status IN ('DRAFT', 'APPROVED', 'RETIRED')),
    catalog_version BIGINT NOT NULL DEFAULT 1,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_by TEXT,
    approved_at TIMESTAMPTZ,
    retired_by TEXT,
    retired_at TIMESTAMPTZ,
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, form_code, version),
    CHECK (
        (status = 'DRAFT' AND approved_by IS NULL AND approved_at IS NULL
            AND retired_by IS NULL AND retired_at IS NULL)
        OR (status = 'APPROVED' AND approved_by IS NOT NULL AND approved_at IS NOT NULL
            AND retired_by IS NULL AND retired_at IS NULL)
        OR (status = 'RETIRED' AND retired_by IS NOT NULL AND retired_at IS NOT NULL)
    )
);

-- form_field: field definitions within a form version, referencing the nine-key catalog.
CREATE TABLE IF NOT EXISTS form_field (
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    form_id TEXT NOT NULL,
    field_id TEXT NOT NULL,
    field_code TEXT NOT NULL,
    field_type TEXT NOT NULL CHECK (field_type IN (
        'TEXT', 'NUMBER', 'DATE', 'BOOLEAN',
        'SINGLE_SELECT', 'MULTI_SELECT',
        'FILE_REFERENCE', 'SIGNATURE', 'MEASUREMENT'
    )),
    label TEXT NOT NULL,
    description TEXT,
    required BOOLEAN NOT NULL DEFAULT false,
    critical BOOLEAN NOT NULL DEFAULT false,
    sort_order BIGINT NOT NULL DEFAULT 0,
    validation_rules JSONB,
    options JSONB,
    default_value TEXT,
    section_id TEXT,
    section_label TEXT,
    PRIMARY KEY (tenant_id, organization_id, form_id, field_id),
    FOREIGN KEY (tenant_id, organization_id, form_id)
        REFERENCES form_version (tenant_id, organization_id, id) ON DELETE RESTRICT,
    UNIQUE (tenant_id, organization_id, form_id, field_code),
    CHECK (sort_order >= 0),
    CHECK (
        (field_type IN ('SINGLE_SELECT', 'MULTI_SELECT') AND options IS NOT NULL)
        OR (field_type NOT IN ('SINGLE_SELECT', 'MULTI_SELECT'))
    )
);

-- evidence_policy: per-field evidence requirements for form fields.
CREATE TABLE IF NOT EXISTS evidence_policy (
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    form_id TEXT NOT NULL,
    field_id TEXT NOT NULL,
    evidence_type TEXT NOT NULL CHECK (evidence_type IN (
        'PHOTO', 'DOCUMENT', 'SIGNATURE', 'VIDEO', 'AUDIO', 'MEASUREMENT'
    )),
    required BOOLEAN NOT NULL DEFAULT false,
    max_count BIGINT CHECK (max_count > 0),
    max_size_bytes BIGINT CHECK (max_size_bytes > 0),
    allowed_content_types JSONB,
    retention_days BIGINT CHECK (retention_days > 0),
    classification TEXT NOT NULL DEFAULT 'CONFIDENTIAL',
    PRIMARY KEY (tenant_id, organization_id, form_id, field_id, evidence_type),
    FOREIGN KEY (tenant_id, organization_id, form_id)
        REFERENCES form_version (tenant_id, organization_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (tenant_id, organization_id, form_id, field_id)
        REFERENCES form_field (tenant_id, organization_id, form_id, field_id) ON DELETE RESTRICT,
    CHECK (classification IN ('PUBLIC', 'INTERNAL', 'CONFIDENTIAL', 'RESTRICTED'))
);

-- Indexes for lookup patterns.
CREATE INDEX form_version_lookup_idx
    ON form_version (tenant_id, organization_id, form_code, version, status);

CREATE INDEX form_version_asset_type_idx
    ON form_version (tenant_id, organization_id, asset_type)
    WHERE asset_type IS NOT NULL;

CREATE INDEX form_field_form_idx
    ON form_field (tenant_id, organization_id, form_id, sort_order);

CREATE INDEX form_field_section_idx
    ON form_field (tenant_id, organization_id, form_id, section_id)
    WHERE section_id IS NOT NULL;

-- Forced Row Level Security: tenant + organization isolation.
ALTER TABLE form_version ENABLE ROW LEVEL SECURITY;
ALTER TABLE form_version FORCE ROW LEVEL SECURITY;
ALTER TABLE form_field ENABLE ROW LEVEL SECURITY;
ALTER TABLE form_field FORCE ROW LEVEL SECURITY;
ALTER TABLE evidence_policy ENABLE ROW LEVEL SECURITY;
ALTER TABLE evidence_policy FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS form_version_tenant_organization_isolation ON form_version;
CREATE POLICY form_version_tenant_organization_isolation ON form_version
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

DROP POLICY IF EXISTS form_field_tenant_organization_isolation ON form_field;
CREATE POLICY form_field_tenant_organization_isolation ON form_field
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

DROP POLICY IF EXISTS evidence_policy_tenant_organization_isolation ON evidence_policy;
CREATE POLICY evidence_policy_tenant_organization_isolation ON evidence_policy
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

COMMIT;
