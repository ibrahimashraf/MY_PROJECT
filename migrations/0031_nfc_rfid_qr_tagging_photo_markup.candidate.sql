-- 0031_nfc_rfid_qr_tagging_photo_markup.candidate.sql
-- NFC/RFID/QR Tagging + Photo Markup/Annotation tables

BEGIN;

-- asset_tag table
CREATE TABLE asset_tag (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    org_id UUID NOT NULL,
    asset_id UUID NOT NULL,
    tag_type VARCHAR(10) NOT NULL CHECK (tag_type IN ('NFC', 'RFID', 'QR')),
    tag_uid TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'RETIRED')),
    provisioned_at TIMESTAMPTZ,
    provisioned_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, org_id, tag_type, tag_uid)
);

-- Enable RLS
ALTER TABLE asset_tag ENABLE ROW LEVEL SECURITY;
ALTER TABLE asset_tag FORCE ROW LEVEL SECURITY;

-- Index on tag_uid for O(1) scan lookup
CREATE INDEX idx_asset_tag_tag_uid ON asset_tag(tag_uid);

-- RLS Policy
CREATE POLICY asset_tag_tenant_isolation ON asset_tag
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND org_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND org_id = current_setting('integin.organization_id', true)
    );

-- photo_markup table
CREATE TABLE photo_markup (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    org_id UUID NOT NULL,
    evidence_id TEXT NOT NULL,
    markup_type VARCHAR(20) NOT NULL CHECK (markup_type IN ('RECTANGLE', 'CIRCLE', 'ARROW', 'TEXT', 'FREEHAND')),
    coordinates JSONB NOT NULL,
    color VARCHAR(20),
    label VARCHAR(200),
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Enable RLS
ALTER TABLE photo_markup ENABLE ROW LEVEL SECURITY;
ALTER TABLE photo_markup FORCE ROW LEVEL SECURITY;

-- Index on evidence_id for fast lookups (linked via #q= fragment)
CREATE INDEX idx_photo_markup_evidence_id ON photo_markup(evidence_id);

-- RLS Policy
CREATE POLICY photo_markup_tenant_isolation ON photo_markup
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND org_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND org_id = current_setting('integin.organization_id', true)
    );

COMMIT;