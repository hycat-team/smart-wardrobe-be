-- +goose Up
-- +goose StatementBegin
-- 030 DDL ownership handover (US1): all four dashboard tables (including
-- dashboard_brand_item_daily_snapshots, later dropped by 20260724112217)
-- moved to the Commerce baseline. No-op preserves the version row (FR-008).
SELECT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Exact reverse of the new Up: nothing to drop.
SELECT 1;
-- +goose StatementEnd
