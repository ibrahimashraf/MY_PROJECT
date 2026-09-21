-- CANDIDATE ROLLBACK ONLY. Review data preservation before use.
BEGIN;
DROP FUNCTION IF EXISTS integin_resolve_identity_membership(TEXT, TEXT);
CREATE FUNCTION integin_resolve_identity_membership(p_issuer TEXT, p_subject TEXT)
RETURNS TABLE (tenant_id TEXT, organization_id TEXT, capabilities TEXT[])
LANGUAGE sql SECURITY DEFINER SET search_path = pg_catalog, public AS $$
SELECT m.tenant_id, m.organization_id, COALESCE(array_agg(c.capability ORDER BY c.capability) FILTER (WHERE c.status = 'ACTIVE'), ARRAY[]::TEXT[])
FROM public.identity_subject s JOIN public.identity_membership m ON m.subject_id=s.subject_id AND m.status='ACTIVE'
LEFT JOIN public.identity_membership_capability c ON c.membership_id=m.membership_id
WHERE s.issuer=p_issuer AND s.subject=p_subject AND s.status='ACTIVE'
GROUP BY m.membership_id, m.tenant_id, m.organization_id;
$$;
REVOKE ALL ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) TO integin_runtime;
ALTER TABLE identity_membership DROP CONSTRAINT IF EXISTS identity_membership_actor_role_required;
ALTER TABLE identity_membership DROP COLUMN IF EXISTS work_order_role;
ALTER TABLE identity_membership DROP COLUMN IF EXISTS actor_id;
COMMIT;
