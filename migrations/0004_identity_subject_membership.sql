-- INTEGIN OIDC foundation: maps validated issuer-subject identities to exactly one local INTEGIN membership without granting runtime direct table access.
CREATE TABLE IF NOT EXISTS identity_subject (
    subject_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    issuer TEXT NOT NULL,
    subject TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (issuer, subject)
);

CREATE TABLE IF NOT EXISTS identity_membership (
    membership_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    subject_id BIGINT NOT NULL REFERENCES identity_subject(subject_id) ON DELETE RESTRICT,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS identity_membership_one_active_per_subject_idx
    ON identity_membership(subject_id)
    WHERE status = 'ACTIVE';

CREATE TABLE IF NOT EXISTS identity_membership_capability (
    membership_id BIGINT NOT NULL REFERENCES identity_membership(membership_id) ON DELETE CASCADE,
    capability TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (membership_id, capability)
);

CREATE OR REPLACE FUNCTION integin_resolve_identity_membership(p_issuer TEXT, p_subject TEXT)
RETURNS TABLE (tenant_id TEXT, organization_id TEXT, capabilities TEXT[])
LANGUAGE sql
SECURITY DEFINER
SET search_path = pg_catalog, public
AS $$
    SELECT m.tenant_id,
           m.organization_id,
           COALESCE(array_agg(c.capability ORDER BY c.capability) FILTER (WHERE c.status = 'ACTIVE'), ARRAY[]::TEXT[]) AS capabilities
      FROM public.identity_subject s
      JOIN public.identity_membership m ON m.subject_id = s.subject_id AND m.status = 'ACTIVE'
 LEFT JOIN public.identity_membership_capability c ON c.membership_id = m.membership_id
     WHERE s.issuer = p_issuer
       AND s.subject = p_subject
       AND s.status = 'ACTIVE'
  GROUP BY m.membership_id, m.tenant_id, m.organization_id;
$$;

REVOKE ALL ON TABLE identity_subject, identity_membership, identity_membership_capability FROM PUBLIC;
REVOKE ALL ON TABLE identity_subject, identity_membership, identity_membership_capability FROM integin_test_runtime;
REVOKE ALL ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION integin_resolve_identity_membership(TEXT, TEXT) TO integin_test_runtime;

