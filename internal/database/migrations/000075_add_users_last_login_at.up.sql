-- Add last_login_at to users for tracking active user sessions.
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;
