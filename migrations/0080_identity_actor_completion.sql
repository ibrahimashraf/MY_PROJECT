-- integin Ticket 02-01 completion: finish 0008_identity_actor_alignment on databases
-- where only the identity_actor table landed (partial apply).
-- Fully idempotent and rerunnable: every step guards on current state.
-- Mirrors 0008 exactly (same columns, constraints, resolver function, grants).
-- 0008 itself is superseded by this file on such databases: record 0008 in
-- schema_migrations with its real checksum (baseline marker) so the ledger
-- engine never attempts its non-idempotent statements; the effective schema
-- equals 0008's intent via the steps below plus least-privilege table grants.

-- 1. Canonical actor projection (no-op if the partial apply created it).
CREATE TABLE IF NOT EXISTS identity_actor (
    actor_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (actor_id, tenant_id, organization_id)
);

-- 2. Membership columns (no-op if a prior partial apply added them).
ALTER TABLE identity_membership
    ADD COLUMN IF NOT EXISTS actor_id TEXT;

ALTER TABLE identity_membership
    ADD COLUMN IF NOT EXISTS work_order_role TEXT;

-- 3. Hard stop unless every existing membership has an approved mapping.
-- Same guard as 0008: never guess mappings.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM public.identity_membership
         WHERE actor_id IS NULL OR work_order_role IS NULL
    ) THEN
        RAISE EXCEPTION 'identity membership actor and work-order-role mappings are required before Ticket 02-01 alignment (backfill actor_id/work_order_role first)';
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

-- 4. Nullability (safe after the guard above).
ALTER TABLE identity_membership
    ALTER COLUMN actor_id SET NOT NULL,
    ALTER COLUMN work_order_role SET NOT NULL;

-- 5. FK + role check, idempotent (0008 used bare ADD CONSTRAINT).
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'identity_membership_actor_tenant_org_fk'
    ) THEN
        ALTER TABLE identity_membership
            ADD CONSTRAINT identity_membership_actor_tenant_org_fk
            FOREIGN KEY (actor_id, tenant_id, organization_id)
            REFERENCES identity_actor (actor_id, tenant_id, organization_id)
            ON DELETE RESTRICT;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'identity_membership_work_order_role_ck'
    ) THEN
        ALTER TABLE identity_membership
            ADD CONSTRAINT identity_membership_work_order_role_ck
            CHECK (work_order_role IN ('inspector', 'reviewer', 'manager', 'administrator'));
    END IF;
END;
$$;

CREATE INDEX IF NOT EXISTS identity_membership_actor_idx
    ON identity_membership(actor_id);

-- 6. Canonical resolver (drop + recreate mirrors 0008; rerunnable).
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

-- Least-privilege runtime grants: the pilot/test path writes membership rows
-- directly (shared integration helpers), so restore the effective rights the
-- pre-0080 schema carried. Reads in production go through the SECURITY DEFINER
-- resolver above; these grants keep that path intact while allowing direct writes.
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE identity_actor, identity_subject, identity_membership, identity_membership_capability TO integin_test_runtime;

