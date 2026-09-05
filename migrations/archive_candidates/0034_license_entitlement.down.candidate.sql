-- Down migration 0034: License Entitlement Engine
BEGIN;
DROP TABLE IF EXISTS license_audit;
DROP TABLE IF EXISTS tenant_license;
COMMIT;
