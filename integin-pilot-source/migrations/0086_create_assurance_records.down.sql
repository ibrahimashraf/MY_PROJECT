SET lock_timeout = '2s';

DROP POLICY IF EXISTS tenant_isolation_current ON current_assurance_states;
DROP POLICY IF EXISTS tenant_isolation_records ON assurance_records;

DROP TABLE IF EXISTS current_assurance_states;
DROP TABLE IF EXISTS assurance_records;
