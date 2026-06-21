-- +goose Up
-- Remove standard case-sensitive unique constraint on email
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;

-- Create case-insensitive unique indexes on LOWER(email) and LOWER(username)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower ON users (LOWER(email));
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_lower ON users (LOWER(username));

-- +goose Down
-- Drop case-insensitive unique indexes
DROP INDEX IF EXISTS idx_users_email_lower;
DROP INDEX IF EXISTS idx_users_username_lower;

-- Restore standard case-sensitive unique constraint on email
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);

