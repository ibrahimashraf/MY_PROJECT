-- INTEGIN Ticket 03 D7-5 down: reverses 0012b_work_order_handover.sql (placeholder).
-- Drops RLS policy, grants, and table. Reverse order 0012 -> 0009 per contract.

DROP POLICY IF EXISTS work_order_handover_tenant_organization_isolation ON work_order_handover;

ALTER TABLE work_order_handover NO FORCE ROW LEVEL SECURITY;
ALTER TABLE work_order_handover DISABLE ROW LEVEL SECURITY;

REVOKE ALL ON TABLE work_order_handover FROM integin_runtime;

DROP INDEX IF EXISTS work_order_handover_work_order_idx;

DROP TABLE IF EXISTS work_order_handover;
