-- Migration 0029: Parts catalog, service charges, timesheet auto-capture
-- Down migration

DROP POLICY IF EXISTS timesheet_auto_capture_tenant_isolation ON timesheet_auto_capture;
DROP POLICY IF EXISTS service_charge_tenant_isolation ON service_charge;
DROP POLICY IF EXISTS parts_catalog_tenant_isolation ON parts_catalog;

DROP INDEX IF EXISTS idx_timesheet_auto_check_in;
DROP INDEX IF EXISTS idx_timesheet_auto_status;
DROP INDEX IF EXISTS idx_timesheet_auto_assignment;
DROP INDEX IF EXISTS idx_timesheet_auto_work_order_tech;
DROP INDEX IF EXISTS idx_timesheet_auto_tenant_org;

DROP INDEX IF EXISTS idx_service_charge_type;
DROP INDEX IF EXISTS idx_service_charge_inspection;
DROP INDEX IF EXISTS idx_service_charge_work_order;
DROP INDEX IF EXISTS idx_service_charge_tenant_org;

DROP INDEX IF EXISTS idx_parts_catalog_active;
DROP INDEX IF EXISTS idx_parts_catalog_code;
DROP INDEX IF EXISTS idx_parts_catalog_tenant_org;

DROP TABLE IF EXISTS timesheet_auto_capture;
DROP TABLE IF EXISTS service_charge;
DROP TABLE IF EXISTS parts_catalog;

DROP TYPE IF EXISTS timesheet_auto_status;
DROP TYPE IF EXISTS charge_type;