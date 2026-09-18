-- +goose Up
-- +goose StatementBegin
-- ========================================================
-- Migration gộp (Feature 012 - Brand RBAC & Admin):
-- (a) Theo dõi phiên đăng nhập cho identity.refresh_tokens
-- (b) Bảng audit log quản trị shared.admin_audit_logs
-- ========================================================

-- (a) Thêm cột theo dõi phiên vào identity.refresh_tokens
ALTER TABLE identity.refresh_tokens
    ADD COLUMN IF NOT EXISTS device_name varchar(255),
    ADD COLUMN IF NOT EXISTS device_ip varchar(45),
    ADD COLUMN IF NOT EXISTS user_agent varchar(512),
    ADD COLUMN IF NOT EXISTS last_active_at timestamptz;

-- (b) Tạo bảng audit log quản trị
CREATE TABLE IF NOT EXISTS shared.admin_audit_logs (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id uuid NOT NULL REFERENCES identity.users(id),
    action        varchar(100) NOT NULL,
    target_type   varchar(50)  NOT NULL,
    target_id     uuid,
    payload       jsonb NOT NULL DEFAULT '{}'::jsonb,
    request_ip    varchar(45),
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_actor
    ON shared.admin_audit_logs (actor_user_id);
CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_target
    ON shared.admin_audit_logs (target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_created_at
    ON shared.admin_audit_logs (created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS shared.admin_audit_logs;

ALTER TABLE identity.refresh_tokens
    DROP COLUMN IF EXISTS device_name,
    DROP COLUMN IF EXISTS device_ip,
    DROP COLUMN IF EXISTS user_agent,
    DROP COLUMN IF EXISTS last_active_at;
-- +goose StatementEnd
