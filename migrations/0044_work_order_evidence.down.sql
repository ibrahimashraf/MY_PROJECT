-- integin Ticket 03 D7-6 down: reverses 0044_work_order_evidence.sql (candidate style).
-- Drops RLS policy, grants, and table. Reverse order 0044 -> 0008 per contract 6.

DROP POLICY IF EXISTS work_order_evidence_tenant_organization_isolation ON work_order_evidence;

ALTER TABLE work_order_evidence NO FORCE ROW LEVEL SECURITY;
ALTER TABLE work_order_evidence DISABLE ROW LEVEL SECURITY;

REVOKE ALL ON TABLE work_order_evidence FROM integin_test_runtime;

DROP INDEX IF EXISTS work_order_evidence_content_hash_idx;
DROP INDEX IF EXISTS work_order_evidence_work_order_idx;

DROP TABLE IF EXISTS work_order_evidence;

