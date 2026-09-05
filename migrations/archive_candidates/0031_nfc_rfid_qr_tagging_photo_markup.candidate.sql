-- Migration 0031: NFC/RFID/QR Tagging + Photo Markup/Annotation
-- Compatible with existing schema (TEXT IDs, integin.* settings, FORCE RLS)
-- CANDIDATE ONLY: do not apply without verified pre-apply backup and isolated up/down review.

BEGIN;

-- asset_tag table (unified)
CREATE TABLE IF NOT EXISTS asset_tag (
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    tag_type TEXT NOT NULL CHECK (tag_type IN ('NFC','RFID','QR','BARCODE','SERIAL_PLATE')),
    tag_value TEXT NOT NULL,
    tag_digest TEXT NOT NULL,
    provisioned_at TIMESTAMPTZ,
    provisioned_by TEXT,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    assigned_by TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','RETIRED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, asset_id, tag_type, tag_value),
    UNIQUE (tenant_id, organization_id, tag_type, tag_value)
);
CREATE INDEX IF NOT EXISTS idx_asset_tag_asset_idx ON asset_tag (tenant_id, organization_id, asset_id);
CREATE INDEX IF NOT EXISTS idx_asset_tag_digest_idx ON asset_tag (tenant_id, organization_id, tag_type, tag_digest);
ALTER TABLE asset_tag ENABLE ROW LEVEL SECURITY;
ALTER TABLE asset_tag FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS asset_tag_tenant_isolation ON asset_tag;
CREATE POLICY asset_tag_tenant_isolation ON asset_tag
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

-- photo_markup table
CREATE TABLE IF NOT EXISTS photo_markup (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    evidence_id TEXT NOT NULL,
    markup_type TEXT NOT NULL CHECK (markup_type IN ('RECTANGLE','CIRCLE','ARROW','TEXT','FREEHAND')),
    coordinates JSONB NOT NULL,
    color TEXT,
    label TEXT,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index on evidence_id for fast lookups (linked via #q= fragment)
CREATE INDEX IF NOT EXISTS idx_photo_markup_evidence_id ON photo_markup (tenant_id, organization_id, evidence_id);

-- RLS
ALTER TABLE photo_markup ENABLE ROW LEVEL SECURITY;
ALTER TABLE photo_markup FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS photo_markup_tenant_isolation ON photo_markup;
CREATE POLICY photo_markup_tenant_isolation ON photo_markup
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;