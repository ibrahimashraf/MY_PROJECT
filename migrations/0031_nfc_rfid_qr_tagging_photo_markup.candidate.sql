-- Migration 0031: NFC/RFID/QR Tagging + Photo Markup/Annotation
-- Compatible with existing schema (TEXT IDs, integin.* settings, FORCE RLS)
-- CANDIDATE ONLY: do not apply without verified pre-apply backup and isolated up/down review.

BEGIN;

-- asset_tag table
CREATE TABLE IF NOT EXISTS asset_tag (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    tag_type TEXT NOT NULL CHECK (tag_type IN ('NFC','RFID','QR')),
    tag_uid TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','RETIRED')),
    provisioned_at TIMESTAMPTZ,
    provisioned_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, tag_type, tag_uid)
);

-- Index on tag_uid for O(1) scan lookup
CREATE INDEX IF NOT EXISTS idx_asset_tag_tag_uid ON asset_tag (tenant_id, organization_id, tag_uid);

-- RLS
ALTER TABLE asset_tag ENABLE ROW LEVEL SECURITY;
ALTER TABLE asset_tag FORCE ROW LEVEL SECURITY;

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

CREATE POLICY photo_markup_tenant_isolation ON photo_markup
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

COMMIT;