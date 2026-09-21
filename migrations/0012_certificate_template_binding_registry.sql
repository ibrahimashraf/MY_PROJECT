-- integin certificate-template binding registry foundation.
-- CANDIDATE ONLY: do not apply without a verified backup and isolated SQL review.

BEGIN;

CREATE TABLE IF NOT EXISTS certificate_template (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    template_code TEXT NOT NULL,
    version BIGINT NOT NULL CHECK (version > 0),
    title TEXT NOT NULL,
    asset_type TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('DRAFT', 'APPROVED', 'RETIRED')),
    catalog_version BIGINT NOT NULL CHECK (catalog_version > 0),
    page_count BIGINT NOT NULL CHECK (page_count > 0),
    page_width_points NUMERIC(10,2) NOT NULL CHECK (page_width_points > 0),
    page_height_points NUMERIC(10,2) NOT NULL CHECK (page_height_points > 0),
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_by TEXT,
    approved_at TIMESTAMPTZ,
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, template_code, version),
    CHECK (
        (status = 'DRAFT' AND approved_by IS NULL AND approved_at IS NULL)
        OR (status IN ('APPROVED', 'RETIRED') AND approved_by IS NOT NULL AND approved_at IS NOT NULL)
    )
);

CREATE TABLE IF NOT EXISTS certificate_template_cell (
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    template_id TEXT NOT NULL,
    cell_id TEXT NOT NULL,
    page_number BIGINT NOT NULL CHECK (page_number > 0),
    x_points NUMERIC(10,2) NOT NULL CHECK (x_points >= 0),
    y_points NUMERIC(10,2) NOT NULL CHECK (y_points >= 0),
    width_points NUMERIC(10,2) NOT NULL CHECK (width_points > 0),
    height_points NUMERIC(10,2) NOT NULL CHECK (height_points > 0),
    kind TEXT NOT NULL CHECK (kind IN ('TEXT', 'CHECKBOX', 'DATE', 'STATIC_TEXT', 'REPEATING_REGION')),
    binding_key TEXT,
    static_text TEXT,
    label TEXT,
    fit_policy TEXT NOT NULL CHECK (fit_policy IN ('WRAP_REQUIRED', 'SINGLE_LINE_REQUIRED', 'CHECKBOX_MAP', 'REPEAT_REQUIRED')),
    max_lines BIGINT,
    required BOOLEAN NOT NULL DEFAULT false,
    checkbox_values JSONB,
    condition_binding_key TEXT,
    condition_operator TEXT CHECK (condition_operator IN ('EQUALS', 'NOT_EQUALS')),
    condition_literal TEXT,
    repeat_source TEXT,
    max_items BIGINT,
    PRIMARY KEY (tenant_id, organization_id, template_id, cell_id),
    FOREIGN KEY (tenant_id, organization_id, template_id)
        REFERENCES certificate_template (tenant_id, organization_id, id) ON DELETE RESTRICT,
    CHECK ((kind = 'STATIC_TEXT') = (binding_key IS NULL)),
    CHECK ((kind = 'REPEATING_REGION') = (repeat_source IS NOT NULL)),
    CHECK ((condition_binding_key IS NULL AND condition_operator IS NULL AND condition_literal IS NULL)
        OR (condition_binding_key IS NOT NULL AND condition_operator IS NOT NULL AND condition_literal IS NOT NULL))
);

CREATE INDEX certificate_template_lookup_idx
    ON certificate_template (tenant_id, organization_id, template_code, version, status);

ALTER TABLE certificate_template ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_template FORCE ROW LEVEL SECURITY;
ALTER TABLE certificate_template_cell ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_template_cell FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS certificate_template_tenant_isolation ON certificate_template;
CREATE POLICY certificate_template_tenant_isolation ON certificate_template
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

DROP POLICY IF EXISTS certificate_template_cell_tenant_isolation ON certificate_template_cell;
CREATE POLICY certificate_template_cell_tenant_isolation ON certificate_template_cell
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;
