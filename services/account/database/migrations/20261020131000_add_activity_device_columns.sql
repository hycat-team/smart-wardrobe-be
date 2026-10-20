-- +goose Up
-- +goose StatementBegin
-- 029 D-07: the activity log recorded WHO and WHEN but not FROM WHERE, so an admin
-- investigating "which device/IP did this login come from" had to join the session
-- table by timestamp and guess. These three columns are copied from the session the
-- event belongs to (login input, refresh token on refresh).
--
-- They are nullable on purpose: rows written before this migration have no source to
-- backfill from. Same types as identity.refresh_tokens (migration 20260817090000).
ALTER TABLE identity.user_activity_events
    ADD COLUMN IF NOT EXISTS device_name varchar(255),
    ADD COLUMN IF NOT EXISTS device_ip   varchar(45),
    ADD COLUMN IF NOT EXISTS user_agent  varchar(512);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE identity.user_activity_events
    DROP COLUMN IF EXISTS user_agent,
    DROP COLUMN IF EXISTS device_ip,
    DROP COLUMN IF EXISTS device_name;
-- +goose StatementEnd
