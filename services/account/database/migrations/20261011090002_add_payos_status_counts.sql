-- +goose Up
-- +goose StatementBegin
-- 030 DDL ownership handover (US1): the three payos columns moved to the
-- Commerce dashboard baseline, which creates them from the start. No-op
-- preserves the version row (FR-008). Original rationale kept for history.
SELECT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Exact reverse of the new Up: nothing to drop.
SELECT 1;
-- +goose StatementEnd
