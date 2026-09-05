-- Drop custom_domain column
DROP INDEX IF EXISTS idx_short_links_custom_domain;
ALTER TABLE short_links DROP COLUMN IF EXISTS custom_domain;