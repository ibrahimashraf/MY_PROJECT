-- 0027_client_portal_domains_acls.down.candidate.sql
-- Revert: Client portal custom domains + granular ACLs + QuickLink + client audit trail

DROP INDEX IF EXISTS client_audit_entry_client_time_idx;
DROP INDEX IF EXISTS quick_link_expiry_idx;
DROP INDEX IF EXISTS quick_link_token_idx;
DROP INDEX IF EXISTS client_portal_acl_client_lookup_idx;
DROP INDEX IF EXISTS client_portal_domain_status_idx;

DROP TABLE IF EXISTS client_audit_entry;
DROP TABLE IF EXISTS quick_link;
DROP TABLE IF EXISTS client_portal_acl;
DROP TABLE IF EXISTS client_portal_domain;