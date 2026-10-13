-- Remove HMAC columns from short_links table
ALTER TABLE short_links
    DROP COLUMN IF EXISTS hmac_secret_ref,
    DROP COLUMN IF EXISTS hmac_algorithm,
    DROP COLUMN IF EXISTS hmac_signature;

-- Drop HMAC secrets table
DROP TABLE IF EXISTS short_link_hmac_secrets;