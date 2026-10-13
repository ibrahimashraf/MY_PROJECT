-- integin Work-Order Field Package: asset entitlement and offline package.
-- CANDIDATE ONLY: do not apply without a verified backup and isolated SQL review.
-- Builds on 0009 work_order_scope_item, 0012 certificate_template, 0052 form_version.

BEGIN;

-- asset_entitlement: server-authoritative entitlement for an asset within a work order scope.
-- The asset is a durable domain anchor, not a second source of truth.
CREATE TABLE IF NOT EXISTS asset_entitlement (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    work_order_id TEXT NOT NULL,
    scope_item_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    asset_type TEXT NOT NULL,
    entitlement_type TEXT NOT NULL CHECK (entitlement_type IN ('INSPECTION', 'MAINTENANCE', 'AUDIT', 'REVIEW')),
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'COMPLETED', 'CANCELLED', 'EXPIRED')),
    form_version_id TEXT,
    assigned_inspector_id TEXT,
    entitled_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, work_order_id, scope_item_id, asset_id),
    FOREIGN KEY (tenant_id, organization_id, work_order_id)
        REFERENCES work_order (tenant_id, organization_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (tenant_id, organization_id, scope_item_id)
        REFERENCES work_order_scope_item (tenant_id, organization_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (tenant_id, organization_id, form_version_id)
        REFERENCES form_version (tenant_id, organization_id, id) ON DELETE SET NULL,
    CHECK (
        (status = 'COMPLETED' AND completed_at IS NOT NULL)
        OR (status != 'COMPLETED' AND completed_at IS NULL)
    ),
    CHECK (
        (status = 'EXPIRED' AND expires_at IS NOT NULL AND expires_at <= now())
        OR (status != 'EXPIRED')
    )
);

-- asset_tag: untrusted physical pointers (QR, barcode, NFC) for assets.
-- Tags are untrusted; the asset_id is the authoritative anchor.
CREATE TABLE IF NOT EXISTS asset_tag (
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    tag_type TEXT NOT NULL CHECK (tag_type IN ('QR', 'BARCODE', 'NFC', 'RFID', 'SERIAL_PLATE')),
    tag_value TEXT NOT NULL,
    tag_digest TEXT NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    assigned_by TEXT NOT NULL,
    PRIMARY KEY (tenant_id, organization_id, asset_id, tag_type, tag_value),
    UNIQUE (tenant_id, organization_id, tag_type, tag_value)
);

-- offline_package: server-reconciled, bounded, never client-authoritative offline data package.
CREATE TABLE IF NOT EXISTS offline_package (
    id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    entitlement_id TEXT NOT NULL,
    package_version BIGINT NOT NULL CHECK (package_version > 0),
    form_snapshot JSONB NOT NULL,
    evidence_policy_snapshot JSONB NOT NULL,
    asset_context JSONB NOT NULL,
    package_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'GENERATED', 'DELIVERED', 'EXPIRED', 'REVOKED')),
    generated_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    delivered_to_device_id TEXT,
    delivered_to_inspector_id TEXT,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    revoked_by TEXT,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, organization_id, id),
    UNIQUE (tenant_id, organization_id, entitlement_id, package_version),
    FOREIGN KEY (tenant_id, organization_id, entitlement_id)
        REFERENCES asset_entitlement (tenant_id, organization_id, id) ON DELETE RESTRICT,
    CHECK (
        (status = 'GENERATED' AND generated_at IS NOT NULL AND package_hash IS NOT NULL)
        OR (status != 'GENERATED')
    ),
    CHECK (
        (status = 'DELIVERED' AND delivered_at IS NOT NULL AND delivered_to_device_id IS NOT NULL AND delivered_to_inspector_id IS NOT NULL)
        OR (status != 'DELIVERED')
    ),
    CHECK (
        (status = 'REVOKED' AND revoked_at IS NOT NULL AND revoked_by IS NOT NULL)
        OR (status != 'REVOKED')
    )
);

-- Indexes for lookup patterns.
CREATE INDEX IF NOT EXISTS asset_entitlement_work_order_idx
    ON asset_entitlement (tenant_id, organization_id, work_order_id, status);

CREATE INDEX IF NOT EXISTS asset_entitlement_asset_idx
    ON asset_entitlement (tenant_id, organization_id, asset_id, status);

CREATE INDEX IF NOT EXISTS asset_entitlement_inspector_idx
    ON asset_entitlement (tenant_id, organization_id, assigned_inspector_id)
    WHERE assigned_inspector_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS asset_entitlement_form_idx
    ON asset_entitlement (tenant_id, organization_id, form_version_id)
    WHERE form_version_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS asset_tag_asset_idx
    ON asset_tag (tenant_id, organization_id, asset_id);

CREATE INDEX IF NOT EXISTS asset_tag_digest_idx
    ON asset_tag (tenant_id, organization_id, tag_type, tag_digest);

CREATE INDEX IF NOT EXISTS offline_package_entitlement_idx
    ON offline_package (tenant_id, organization_id, entitlement_id, status);

CREATE INDEX IF NOT EXISTS offline_package_status_idx
    ON offline_package (tenant_id, organization_id, status, generated_at);

CREATE INDEX IF NOT EXISTS offline_package_device_idx
    ON offline_package (tenant_id, organization_id, delivered_to_device_id)
    WHERE delivered_to_device_id IS NOT NULL;

-- Forced Row Level Security: tenant + organization isolation.
ALTER TABLE asset_entitlement ENABLE ROW LEVEL SECURITY;
ALTER TABLE asset_entitlement FORCE ROW LEVEL SECURITY;
ALTER TABLE asset_tag ENABLE ROW LEVEL SECURITY;
ALTER TABLE asset_tag FORCE ROW LEVEL SECURITY;
ALTER TABLE offline_package ENABLE ROW LEVEL SECURITY;
ALTER TABLE offline_package FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS asset_entitlement_tenant_organization_isolation ON asset_entitlement;
CREATE POLICY asset_entitlement_tenant_organization_isolation ON asset_entitlement
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

DROP POLICY IF EXISTS asset_tag_tenant_organization_isolation ON asset_tag;
CREATE POLICY asset_tag_tenant_organization_isolation ON asset_tag
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

DROP POLICY IF EXISTS offline_package_tenant_organization_isolation ON offline_package;
CREATE POLICY offline_package_tenant_organization_isolation ON offline_package
    USING (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('integin.tenant_id', true)
        AND organization_id = current_setting('integin.organization_id', true)
    );

COMMIT;
