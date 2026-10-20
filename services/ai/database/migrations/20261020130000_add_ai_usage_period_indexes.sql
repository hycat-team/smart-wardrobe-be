-- +goose Up
-- +goose StatementBegin
-- 029 D-01: indexes for the period-bounded AI usage queries.
--
-- The admin dashboard aggregates usage over a half-open period and groups released
-- errors by status. `ai_usage_events` carried two indexes that cannot serve either
-- shape: `(user_id, created_at, status)` needs a user, and the partial
-- `ix_ai_usage_events_unknown` covers only `reserved`/`unknown_usage` with a
-- different sort key.
--
-- Measured on a synthetic database (500k events, 30-day window), EXPLAIN
-- (ANALYZE, BUFFERS):
--   period aggregate        59.0 ms -> 18.5 ms  (Parallel Seq Scan -> Bitmap Index Scan)
--   error breakdown         36.9 ms ->  2.9 ms  (Parallel Seq Scan -> Index Only Scan)
--
-- The composite `(status, created_at)` is kept because dropping it puts the error
-- breakdown back to 15.5 ms and forces heap access; the single-column index cannot
-- answer a status filter and range together.
CREATE INDEX IF NOT EXISTS idx_ai_usage_events_created_at
    ON aiexecutor.ai_usage_events (created_at);

CREATE INDEX IF NOT EXISTS idx_ai_usage_events_status_created_at
    ON aiexecutor.ai_usage_events (status, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS aiexecutor.idx_ai_usage_events_created_at;
DROP INDEX IF EXISTS aiexecutor.idx_ai_usage_events_status_created_at;
-- +goose StatementEnd
