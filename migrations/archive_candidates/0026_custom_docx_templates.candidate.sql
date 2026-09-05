-- Migration 0026: Custom DOCX Templates + Certificate Packs + Batch Email
-- Compatible with existing schema (TEXT IDs, integin.* settings, FORCE RLS)
-- CANDIDATE ONLY: do not apply without verified pre-apply backup and isolated up/down review.

BEGIN;

CREATE TABLE IF NOT EXISTS certificate_template_docx (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    template_code TEXT NOT NULL,
    version BIGINT NOT NULL CHECK (version > 0),
    title TEXT NOT NULL,
    asset_type TEXT NOT NULL,
    docx_content BYTEA NOT NULL,
    docx_sha256 BYTEA NOT NULL CHECK (octet_length(docx_sha256) = 32),
    status TEXT NOT NULL CHECK (status IN ('DRAFT','APPROVED','RETIRED')),
    page_count BIGINT NOT NULL CHECK (page_count > 0),
    page_width_points NUMERIC(10,2) NOT NULL CHECK (page_width_points > 0),
    page_height_points NUMERIC(10,2) NOT NULL CHECK (page_height_points > 0),
    created_by TEXT NOT NULL,
    approved_by TEXT,
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, template_code, version),
    CHECK (
        (status = 'DRAFT' AND approved_by IS NULL AND approved_at IS NULL)
        OR (status IN ('APPROVED','RETIRED') AND approved_by IS NOT NULL AND approved_at IS NOT NULL)
    )
);

ALTER TABLE certificate_template_docx ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_template_docx FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS certificate_template_docx_tenant_isolation ON certificate_template_docx;
CREATE POLICY certificate_template_docx_tenant_isolation ON certificate_template_docx
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE INDEX IF NOT EXISTS certificate_template_docx_lookup_idx ON certificate_template_docx (tenant_id, organization_id, template_code, version, status);

CREATE TABLE IF NOT EXISTS certificate_pack (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    pack_sha256 BYTEA NOT NULL CHECK (octet_length(pack_sha256) = 32),
    object_key TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('BUILDING','READY','SENT')),
    recipient_emails JSONB NOT NULL DEFAULT '[]'::jsonb,
    sent_at TIMESTAMPTZ,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE certificate_pack ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_pack FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS certificate_pack_tenant_isolation ON certificate_pack;
CREATE POLICY certificate_pack_tenant_isolation ON certificate_pack
    USING (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true))
    WITH CHECK (tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true));

CREATE INDEX IF NOT EXISTS certificate_pack_work_order_status_idx ON certificate_pack (tenant_id, organization_id, work_order_id, status);

CREATE TABLE IF NOT EXISTS certificate_pack_item (
    pack_id TEXT NOT NULL,
    certificate_id TEXT NOT NULL,
    sequence_num BIGINT NOT NULL CHECK (sequence_num > 0),
    PRIMARY KEY (pack_id, certificate_id)
);

ALTER TABLE certificate_pack_item ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_pack_item FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS certificate_pack_item_tenant_isolation ON certificate_pack_item;
CREATE POLICY certificate_pack_item_tenant_isolation ON certificate_pack_item
    USING (pack_id IN (SELECT id FROM certificate_pack WHERE tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)))
    WITH CHECK (pack_id IN (SELECT id FROM certificate_pack WHERE tenant_id = current_setting('integin.tenant_id', true) AND organization_id = current_setting('integin.organization_id', true)));

COMMIT;