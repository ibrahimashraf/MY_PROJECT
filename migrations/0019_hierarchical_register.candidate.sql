-- INTEGIN hierarchical equipment register (branch/area/zone) + custom fields per equipment type.
-- CANDIDATE ONLY: do not apply without a verified pre-apply backup and isolated up/down review.
BEGIN;

-- Location hierarchy: branch -> area -> zone
CREATE TABLE location_branch (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    code TEXT NOT NULL CHECK (char_length(code) BETWEEN 1 AND 32),
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 2 AND 200),
    address TEXT,
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'RETIRED')) DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, code)
);

CREATE TABLE location_area (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    branch_id TEXT NOT NULL REFERENCES location_branch(id) ON DELETE RESTRICT,
    code TEXT NOT NULL CHECK (char_length(code) BETWEEN 1 AND 32),
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 2 AND 200),
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'RETIRED')) DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, branch_id, code)
);

CREATE TABLE location_zone (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    area_id TEXT NOT NULL REFERENCES location_area(id) ON DELETE RESTRICT,
    code TEXT NOT NULL CHECK (char_length(code) BETWEEN 1 AND 32),
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 2 AND 200),
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'RETIRED')) DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, area_id, code)
);

CREATE INDEX location_area_branch_idx ON location_area (tenant_id, organization_id, branch_id);
CREATE INDEX location_zone_area_idx ON location_zone (tenant_id, organization_id, area_id);

-- Equipment type custom fields (dynamic schema per type)
CREATE TABLE equipment_type (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    code TEXT NOT NULL CHECK (char_length(code) BETWEEN 1 AND 64),
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 2 AND 200),
    category TEXT NOT NULL CHECK (category IN ('CHAINS', 'SLINGS', 'SHACKLES', 'HOISTS', 'CRANES', 'HARNESSES', 'LANYARDS', 'ANCHORS', 'VEHICLES', 'FIXED_PLANT', 'OTHER')),
    standard_interval_days INT NOT NULL CHECK (standard_interval_days > 0),
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'RETIRED')) DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, code)
);

CREATE TABLE equipment_type_field (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    equipment_type_id TEXT NOT NULL REFERENCES equipment_type(id) ON DELETE CASCADE,
    field_code TEXT NOT NULL CHECK (field_code ~ '^[a-z_][a-z0-9_]{0,63}$'),
    label TEXT NOT NULL CHECK (char_length(label) BETWEEN 1 AND 100),
    data_type TEXT NOT NULL CHECK (data_type IN ('TEXT', 'INTEGER', 'NUMERIC', 'BOOLEAN', 'DATE', 'ENUM')),
    enum_values JSONB CHECK (data_type != 'ENUM' OR (enum_values IS NOT NULL AND jsonb_typeof(enum_values) = 'array')),
    required BOOLEAN NOT NULL DEFAULT FALSE,
    display_order INT NOT NULL DEFAULT 0,
    help_text TEXT,
    UNIQUE (tenant_id, organization_id, equipment_type_id, field_code)
);

CREATE INDEX equipment_type_field_type_idx ON equipment_type_field (tenant_id, organization_id, equipment_type_id);

-- Asset extended with hierarchy + type + custom values
ALTER TABLE asset_registry ADD COLUMN branch_id TEXT REFERENCES location_branch(id) ON DELETE SET NULL;
ALTER TABLE asset_registry ADD COLUMN area_id TEXT REFERENCES location_area(id) ON DELETE SET NULL;
ALTER TABLE asset_registry ADD COLUMN zone_id TEXT REFERENCES location_zone(id) ON DELETE SET NULL;
ALTER TABLE asset_registry ADD COLUMN equipment_type_id TEXT REFERENCES equipment_type(id) ON DELETE SET NULL;
ALTER TABLE asset_registry ADD COLUMN custom_fields JSONB DEFAULT '{}'::jsonb;

CREATE INDEX asset_registry_branch_idx ON asset_registry (tenant_id, organization_id, branch_id) WHERE branch_id IS NOT NULL;
CREATE INDEX asset_registry_area_idx ON asset_registry (tenant_id, organization_id, area_id) WHERE area_id IS NOT NULL;
CREATE INDEX asset_registry_zone_idx ON asset_registry (tenant_id, organization_id, zone_id) WHERE zone_id IS NOT NULL;
CREATE INDEX asset_registry_type_idx ON asset_registry (tenant_id, organization_id, equipment_type_id) WHERE equipment_type_id IS NOT NULL;
CREATE INDEX asset_registry_custom_gin ON asset_registry USING GIN (custom_fields) WHERE custom_fields IS NOT NULL AND custom_fields != '{}'::jsonb;

-- RLS
ALTER TABLE location_branch ENABLE ROW LEVEL SECURITY;
ALTER TABLE location_branch FORCE ROW LEVEL SECURITY;
ALTER TABLE location_area ENABLE ROW LEVEL SECURITY;
ALTER TABLE location_area FORCE ROW LEVEL SECURITY;
ALTER TABLE location_zone ENABLE ROW LEVEL SECURITY;
ALTER TABLE location_zone FORCE ROW LEVEL SECURITY;
ALTER TABLE equipment_type ENABLE ROW LEVEL SECURITY;
ALTER TABLE equipment_type FORCE ROW LEVEL SECURITY;
ALTER TABLE equipment_type_field ENABLE ROW LEVEL SECURITY;
ALTER TABLE equipment_type_field FORCE ROW LEVEL SECURITY;

CREATE POLICY location_branch_tenant_isolation ON location_branch USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));
CREATE POLICY location_area_tenant_isolation ON location_area USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));
CREATE POLICY location_zone_tenant_isolation ON location_zone USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));
CREATE POLICY equipment_type_tenant_isolation ON equipment_type USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));
CREATE POLICY equipment_type_field_tenant_isolation ON equipment_type_field USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)) WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;