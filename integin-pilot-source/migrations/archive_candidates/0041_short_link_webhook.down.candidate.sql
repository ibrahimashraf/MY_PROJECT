-- Remove webhook_url column from short_links table
ALTER TABLE short_links DROP COLUMN IF EXISTS webhook_url;