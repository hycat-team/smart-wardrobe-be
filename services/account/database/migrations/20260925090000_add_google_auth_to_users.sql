-- +goose Up
-- +goose StatementBegin
-- ========================================================
-- Feature 021 - Google Login:
-- (a) Them cot nha cung cap danh tinh cho identity.users
-- (b) Cho phep password_hash NULL (tai khoan Google-only)
-- (c) Backfill email_verified=true cho nguoi dung hien huu
-- ========================================================

ALTER TABLE identity.users
    ADD COLUMN IF NOT EXISTS provider       varchar(50)  NOT NULL DEFAULT 'local',
    ADD COLUMN IF NOT EXISTS provider_id    varchar(255),
    ADD COLUMN IF NOT EXISTS email_verified boolean      NOT NULL DEFAULT false;

ALTER TABLE identity.users
    ALTER COLUMN password_hash DROP NOT NULL;

-- Mot dinh danh nha cung cap chi thuoc mot tai khoan; bo qua local (provider_id NULL).
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_provider_provider_id
    ON identity.users (provider, provider_id)
    WHERE provider_id IS NOT NULL;

UPDATE identity.users
   SET email_verified = true
 WHERE provider = 'local';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS identity.idx_users_provider_provider_id;

ALTER TABLE identity.users
    ALTER COLUMN password_hash SET NOT NULL;

ALTER TABLE identity.users
    DROP COLUMN IF EXISTS provider,
    DROP COLUMN IF EXISTS provider_id,
    DROP COLUMN IF EXISTS email_verified;
-- +goose StatementEnd
