-- INTEGIN Ticket 02-01: add the canonical actor projection without guessing existing mappings.
-- This migration must run with an approved mapping for every existing membership.

CREATE TABLE IF NOT EXISTS identity_actor (
    actor_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (actor_id, tenant_id, organization_id)
);

ALTER TABLE identity_membership
    ADD COLUMN actor_id TEXT;

ALTER TABLE identity_membership
    ADD COLUMN work_order_role TEXT;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM public.identity_membership
         WHERE actor_id IS NULL OR work_order_role IS NULL
    ) THEN
        RAISE EXCEPTION 'identity membership actor and work-order-role mappings are required before Ticket 02-01 alignment';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM public.identity_actor
         WHERE actor_id IS NULL OR tenant_id IS NULL OR organization_id IS NULL
    ) THEN
        RAISE EXCEPTION 'canonical actor identity fields cannot be null';
    END IF;
END;
$$;

ALTER TABLE identity_membership
    ALTER COLUMN actor_id SET NOT NULL,
    ALTER COLUMN work_order_role SET NOT NULL;

ALTER TABLE identity_membership
    ADD CONSTRAINT identity_membership_actor_tenant_org_fk
    FOREIGN KEY (actor_id, tenant_id, organization_id)
    REFERENCES identity_actor (actor_id, tenant_id, organization_id)
    ON DELETE RESTRICT;

ALTER TABLE identity_membership
    ADD CONSTRAINT identity_membership_work_order_role_ck
    CHECK (work_order_role IN ('inspector', 'reviewer', 'manager', 'administrator'));

CREATE INDEX IF NOT EXISTS identity_membership_actor_idx
    ON identity_membership(actor_id);

REVOKE ALL ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) FROM PUBLIC;
REVOKE ALL ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) FROM integin_test_runtime;
DROP FUNCTION IF EXISTS integin_resolve_identity_membership(TEXT, TEXT);

CREATE FUNCTION integin_resolve_identity_membership(p_issuer TEXT, p_subject TEXT)
RETURNS TABLE (
    actor_id TEXT,
    tenant_id TEXT,
    organization_id TEXT,
    work_order_role TEXT,
    capabilities TEXT[]
)
LANGUAGE sql
SECURITY DEFINER
SET search_path = pg_catalog, public
AS $$
    SELECT m.actor_id,
           m.tenant_id,
           m.organization_id,
           m.work_order_role,
           COALESCE(array_agg(c.capability ORDER BY c.capability) FILTER (WHERE c.status = 'ACTIVE'), ARRAY[]::TEXT[]) AS capabilities
      FROM public.identity_subject s
      JOIN public.identity_membership m
        ON m.subject_id = s.subject_id
       AND m.status = 'ACTIVE'
      JOIN public.identity_actor a
        ON a.actor_id = m.actor_id
       AND a.tenant_id = m.tenant_id
       AND a.organization_id = m.organization_id
       AND a.status = 'ACTIVE'
 LEFT JOIN public.identity_membership_capability c
        ON c.membership_id = m.membership_id
     WHERE s.issuer = p_issuer
       AND s.subject = p_subject
       AND s.status = 'ACTIVE'
  GROUP BY m.membership_id,
           m.actor_id,
           m.tenant_id,
           m.organization_id,
           m.work_order_role;
$$;

REVOKE ALL ON TABLE identity_actor, identity_subject, identity_membership, identity_membership_capability FROM PUBLIC;
REVOKE ALL ON TABLE identity_actor, identity_subject, identity_membership, identity_membership_capability FROM integin_test_runtime;
REVOKE ALL ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) TO integin_test_runtime;

