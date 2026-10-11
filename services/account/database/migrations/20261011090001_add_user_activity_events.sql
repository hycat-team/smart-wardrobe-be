-- +goose Up
-- +goose StatementBegin
-- Admin dashboard DAU: an append-only log of real user activity.
--
-- Only login and token-refresh are recorded. A client refreshes its access
-- token roughly every access-token expiry while the app is in use, so the pair
-- (login, refresh) captures genuine app usage without writing a row on every
-- request. The previous DAU proxy counted users.updated_at in the day, which
-- moves on any profile/admin write and says nothing about activity.
CREATE TABLE IF NOT EXISTS identity.user_activity_events (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    activity_type varchar(20) NOT NULL,
    occurred_at   timestamptz NOT NULL DEFAULT now(),
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_activity_events_occurred_at
    ON identity.user_activity_events (occurred_at);

CREATE INDEX IF NOT EXISTS idx_user_activity_events_user_occurred_at
    ON identity.user_activity_events (user_id, occurred_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS identity.user_activity_events;
-- +goose StatementEnd
