-- 0026_custom_docx_templates.candidate.sql
-- Custom DOCX certificate templates + PDF pack per job + batch email

BEGIN;

CREATE TABLE certificate_template_docx (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    org_id UUID NOT NULL REFERENCES organizations(id),
    template_code TEXT NOT NULL,
    version BIGINT NOT NULL CHECK (version > 0),
    title TEXT NOT NULL,
    asset_type TEXT NOT NULL,
    docx_content BYTEA NOT NULL,
    docx_sha256 BYTEA NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('DRAFT', 'APPROVED', 'RETIRED')),
    page_count BIGINT NOT NULL CHECK (page_count > 0),
    page_width_points NUMERIC(10,2) NOT NULL CHECK (page_width_points > 0),
    page_height_points NUMERIC(10,2) NOT NULL CHECK (page_height_points > 0),
    created_by UUID NOT NULL REFERENCES users(id),
    approved_by UUID REFERENCES users(id),
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, org_id, template_code, version),
    CHECK (
        (status = 'DRAFT' AND approved_by IS NULL AND approved_at IS NULL)
        OR (status IN ('APPROVED', 'RETIRED') AND approved_by IS NOT NULL AND approved_at IS NOT NULL)
    )
);

ALTER TABLE certificate_template_docx ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_template_docx FORCE ROW LEVEL SECURITY;

CREATE POLICY certificate_template_docx_tenant_isolation ON certificate_template_docx
    USING (tenant_id = current_setting('app.current_tenant')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

CREATE INDEX certificate_template_docx_lookup_idx
    ON certificate_template_docx (tenant_id, org_id, template_code, version, status);

CREATE TABLE certificate_pack (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    org_id UUID NOT NULL REFERENCES organizations(id),
    work_order_id UUID NOT NULL REFERENCES work_orders(id),
    pack_sha256 BYTEA NOT NULL,
    object_key TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('BUILDING', 'READY', 'SENT')),
    recipient_emails JSONB NOT NULL DEFAULT '[]'::jsonb,
    sent_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE certificate_pack ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_pack FORCE ROW LEVEL SECURITY;

CREATE POLICY certificate_pack_tenant_isolation ON certificate_pack
    USING (tenant_id = current_setting('app.current_tenant')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

CREATE INDEX certificate_pack_work_order_status_idx
    ON certificate_pack (work_order_id, status);

CREATE TABLE certificate_pack_item (
    pack_id UUID NOT NULL REFERENCES certificate_pack(id) ON DELETE CASCADE,
    certificate_id UUID NOT NULL,
    sequence_num BIGINT NOT NULL CHECK (sequence_num > 0),
    PRIMARY KEY (pack_id, certificate_id)
);

ALTER TABLE certificate_pack_item ENABLE ROW LEVEL SECURITY;
ALTER TABLE certificate_pack_item FORCE ROW LEVEL SECURITY;

CREATE POLICY certificate_pack_item_tenant_isolation ON certificate_pack_item
    USING (pack_id IN (SELECT id FROM certificate_pack WHERE tenant_id = current_setting('app.current_tenant')::uuid))
    WITH CHECK (pack_id IN (SELECT id FROM certificate_pack WHERE tenant_id = current_setting('app.current_tenant')::uuid));

COMMIT;