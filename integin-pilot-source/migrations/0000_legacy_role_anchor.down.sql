-- Down migration for 0000: intentionally a no-op.
-- Dropping the anchor role would orphan the GRANT targets recorded in the
-- frozen pre-rename migrations (0001-0084). The role is NOLOGIN and holds no
-- password, so retaining it changes nothing about who can access the database.

DO $$
BEGIN
  RAISE NOTICE '0000 down: legacy role anchor retained by design (see 0000_legacy_role_anchor.sql)';
END $$;
