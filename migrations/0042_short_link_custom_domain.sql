-- Migration 0042: Short Link Custom Domain

BEGIN;

ALTER TABLE short_links ADD COLUMN IF NOT EXISTS custom_domain TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_short_links_custom_domain ON short_links (custom_domain) WHERE custom_domain IS NOT NULL;

GRANT UPDATE (custom_domain) ON short_links TO integin_runtime;

COMMIT;
