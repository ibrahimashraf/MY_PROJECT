-- Migration 0031: NFC/RFID/QR Tagging + Photo Markup/Annotation
-- Compatible with existing schema (TEXT IDs, integin.* settings, FORCE RLS)
-- CANDIDATE ONLY: do not apply without verified pre-apply backup and isolated up/down review.

BEGIN;

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