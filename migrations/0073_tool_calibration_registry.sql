-- ISO 17020 §6.2 tool calibration registry: persistent, tenant-isolated source
-- of truth for equipment calibration validity. Expired-or-missing calibration
-- hard-blocks submission; the store (internal/platform/calibration/postgres.go)
-- checks for an ACTIVE row whose next_due_date is still in the future.

BEGIN;

CREATE TABLE IF NOT EXISTS tool_calibration_registry (
    id                    TEXT        NOT NULL,
    tenant_id             TEXT        NOT NULL,
    organization_id       TEXT        NOT NULL,
    equipment_id          TEXT        NOT NULL,
    serial_number         TEXT        NOT NULL DEFAULT '',
    lab_certificate_ref   TEXT        NOT NULL DEFAULT '',
    uncertainty_tolerance TEXT        NOT NULL DEFAULT '',
    calibration_date      TIMESTAMPTZ NOT NULL,
    next_due_date         TIMESTAMPTZ NOT NULL,
    technician_id         TEXT        NOT NULL,
    result                TEXT        NOT NULL,
    status                TEXT        NOT NULL DEFAULT 'ACTIVE',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, organization_id, id),
    CHECK (next_due_date > calibration_date),
    CHECK (btrim(id) <> ''),
    CHECK (btrim(tenant_id) <> ''),
    CHECK (btrim(organization_id) <> ''),
    CHECK (btrim(equipment_id) <> ''),
    CHECK (btrim(technician_id) <> ''),
    CHECK (status IN ('ACTIVE', 'EXPIRED', 'SUPERSEDED'))
);

CREATE INDEX IF NOT EXISTS tool_calibration_registry_equipment_due_idx
    ON tool_calibration_registry (tenant_id, organization_id, equipment_id, next_due_date);

CREATE INDEX IF NOT EXISTS tool_calibration_registry_status_idx
    ON tool_calibration_registry (tenant_id, organization_id, status);

ALTER TABLE tool_calibration_registry ENABLE ROW LEVEL SECURITY;
ALTER TABLE tool_calibration_registry FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tool_calibration_registry_tenant_isolation ON tool_calibration_registry;
CREATE POLICY tool_calibration_registry_tenant_isolation ON tool_calibration_registry
    USING (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    )
    WITH CHECK (
        tenant_id = NULLIF(current_setting('integin.tenant_id', true), '')
        AND organization_id = NULLIF(current_setting('integin.organization_id', true), '')
        AND tenant_id IS NOT NULL
        AND organization_id IS NOT NULL
    );

GRANT SELECT, INSERT, UPDATE ON tool_calibration_registry TO integin_runtime;

COMMIT;