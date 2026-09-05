-- 0026_custom_docx_templates.down.candidate.sql
-- Revert: Custom DOCX certificate templates + PDF pack per job + batch email

DROP INDEX IF EXISTS certificate_pack_work_order_status_idx;
DROP INDEX IF EXISTS certificate_template_docx_lookup_idx;

DROP TABLE IF EXISTS certificate_pack_item;
DROP TABLE IF EXISTS certificate_pack;
DROP TABLE IF EXISTS certificate_template_docx;