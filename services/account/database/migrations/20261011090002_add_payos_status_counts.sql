-- +goose Up
-- +goose StatementBegin
-- Admin dashboard PayOS reconciliation (fix false admin metrics): the daily
-- snapshot gains a real status breakdown. The old report derived "failed" as
-- total − success, which buckets pending/cancelled/expired/refunded
-- transactions as failures.
ALTER TABLE dashboard.dashboard_admin_daily_snapshots
    ADD COLUMN IF NOT EXISTS payos_pending_count int NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS payos_failed_count  int NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS payos_voided_count  int NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE dashboard.dashboard_admin_daily_snapshots
    DROP COLUMN IF EXISTS payos_pending_count,
    DROP COLUMN IF EXISTS payos_failed_count,
    DROP COLUMN IF EXISTS payos_voided_count;
-- +goose StatementEnd
