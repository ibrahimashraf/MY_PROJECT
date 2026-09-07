BEGIN;

DROP POLICY IF EXISTS river_poison_quarantine_tenant_isolation ON river_poison_quarantine;
ALTER TABLE river_poison_quarantine DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS river_poison_quarantine;

COMMIT;