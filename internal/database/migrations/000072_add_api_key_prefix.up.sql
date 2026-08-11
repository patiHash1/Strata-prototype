ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS key_prefix VARCHAR(16);
CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys (key_prefix) WHERE key_prefix IS NOT NULL;
