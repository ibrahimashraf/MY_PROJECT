-- Reverse 0085_rename_tenant_gucs_to_integin: restore integin.* GUC names.
-- Only valid while application code still dual-sets both names.

BEGIN;

DO $$
DECLARE
  r RECORD;
  using_expr TEXT;
  check_expr TEXT;
  roles_sql TEXT;
  cmd_sql TEXT;
  kind_sql TEXT;
  stmt TEXT;
BEGIN
  FOR r IN
    SELECT p.polname, p.polcmd, p.polroles, p.polpermissive,
           n.nspname AS schemaname, c.relname AS tablename,
           pg_get_expr(p.polqual, p.polrelid) AS using_src,
           pg_get_expr(p.polwithcheck, p.polrelid) AS check_src
    FROM pg_policy p
    JOIN pg_class c ON c.oid = p.polrelid
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE pg_get_expr(p.polqual, p.polrelid) LIKE '%integin.tenant_id%'
       OR pg_get_expr(p.polqual, p.polrelid) LIKE '%integin.organization_id%'
       OR pg_get_expr(p.polwithcheck, p.polrelid) LIKE '%integin.tenant_id%'
       OR pg_get_expr(p.polwithcheck, p.polrelid) LIKE '%integin.organization_id%'
  LOOP
    using_expr := replace(replace(r.using_src, 'integin.tenant_id', 'integin.tenant_id'), 'integin.organization_id', 'integin.organization_id');
    check_expr := replace(replace(r.check_src, 'integin.tenant_id', 'integin.tenant_id'), 'integin.organization_id', 'integin.organization_id');

    SELECT COALESCE(string_agg(quote_ident(rolname), ', '), 'PUBLIC')
      INTO roles_sql
      FROM pg_roles WHERE oid = ANY (r.polroles);

    cmd_sql := CASE r.polcmd
      WHEN 'r' THEN 'SELECT'
      WHEN 'a' THEN 'INSERT'
      WHEN 'w' THEN 'UPDATE'
      WHEN 'd' THEN 'DELETE'
      ELSE 'ALL' END;
    kind_sql := CASE WHEN r.polpermissive THEN 'PERMISSIVE' ELSE 'RESTRICTIVE' END;

    EXECUTE format('DROP POLICY %I ON %I.%I', r.polname, r.schemaname, r.tablename);
    stmt := format('CREATE POLICY %I ON %I.%I AS %s FOR %s TO %s',
        r.polname, r.schemaname, r.tablename, kind_sql, cmd_sql, roles_sql);
    IF using_expr IS NOT NULL THEN
      stmt := stmt || format(' USING (%s)', using_expr);
    END IF;
    IF check_expr IS NOT NULL THEN
      stmt := stmt || format(' WITH CHECK (%s)', check_expr);
    END IF;
    EXECUTE stmt;
  END LOOP;
END $$;

COMMIT;
