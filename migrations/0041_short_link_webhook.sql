-- Migration 0041: Short Link Webhook

BEGIN;

ALTER TABLE short_links ADD COLUMN IF NOT EXISTS webhook_url TEXT;

GRANT UPDATE (webhook_url) ON short_links TO integin_test_runtime;

COMMIT;

