-- Migration 0077 down: reverses 0077_retention_and_legal_hold.sql.
-- Drop triggers/functions and tables in reverse dependency order.

BEGIN;

DROP TRIGGER IF EXISTS deletion_evidence_receipt_immutable_trigger ON deletion_evidence_receipt;
DROP FUNCTION IF EXISTS deletion_evidence_receipt_immutable();

DROP POLICY IF EXISTS deletion_evidence_receipt_tenant_isolation ON deletion_evidence_receipt;
DROP POLICY IF EXISTS export_approval_registry_tenant_isolation ON export_approval_registry;
DROP POLICY IF EXISTS legal_hold_registry_tenant_isolation ON legal_hold_registry;
DROP POLICY IF EXISTS tenant_retention_policy_tenant_isolation ON tenant_retention_policy;

DROP TABLE IF EXISTS deletion_evidence_receipt;
DROP TABLE IF EXISTS export_approval_registry;
DROP TABLE IF EXISTS legal_hold_registry;
DROP TABLE IF EXISTS tenant_retention_policy;

COMMIT;