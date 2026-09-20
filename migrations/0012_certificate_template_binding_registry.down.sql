-- Rollback for 0012_certificate_template_binding_registry.
DROP INDEX IF EXISTS certificate_template_lookup_idx;
DROP TABLE IF EXISTS certificate_template_cell;
DROP TABLE IF EXISTS certificate_template;
