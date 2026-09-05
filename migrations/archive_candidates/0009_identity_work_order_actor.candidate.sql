-- CANDIDATE ONLY. Do not apply before disposable review and explicit pilot preflight.
BEGIN;
ALTER TABLE identity_membership ADD COLUMN IF NOT EXISTS actor_id TEXT;
ALTER TABLE identity_membership ADD COLUMN IF NOT EXISTS work_order_role TEXT CHECK (work_order_role IN ('inspector', 'manager', 'reviewer', 'administrator'));
-- Backfill is deliberately omitted: existing memberships require explicit local actor and role assignment.
DO $$BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'identity_membership_actor_role_required') THEN ALTER TABLE identity_membership ADD CONSTRAINT identity_membership_actor_role_required CHECK ((actor_id IS NULL AND work_order_role IS NULL) OR (actor_id IS NOT NULL AND work_order_role IS NOT NULL)); END IF; END$$;
DROP FUNCTION IF EXISTS integin_resolve_identity_membership(TEXT, TEXT);
CREATE FUNCTION integin_resolve_identity_membership(p_issuer TEXT, p_subject TEXT)
RETURNS TABLE (actor_id TEXT, tenant_id TEXT, organization_id TEXT, work_order_role TEXT, capabilities TEXT[])
LANGUAGE sql SECURITY DEFINER SET search_path = pg_catalog, public AS $$
SELECT m.actor_id, m.tenant_id, m.organization_id, m.work_order_role,
COALESCE(array_agg(c.capability ORDER BY c.capability) FILTER (WHERE c.status = 'ACTIVE'), ARRAY[]::TEXT[])
FROM public.identity_subject s JOIN public.identity_membership m ON m.subject_id=s.subject_id AND m.status='ACTIVE'
LEFT JOIN public.identity_membership_capability c ON c.membership_id=m.membership_id
WHERE s.issuer=p_issuer AND s.subject=p_subject AND s.status='ACTIVE' AND m.actor_id IS NOT NULL AND m.work_order_role IS NOT NULL
GROUP BY m.membership_id, m.actor_id, m.tenant_id, m.organization_id, m.work_order_role;
$$;
REVOKE ALL ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) TO integin_runtime;
COMMIT;
