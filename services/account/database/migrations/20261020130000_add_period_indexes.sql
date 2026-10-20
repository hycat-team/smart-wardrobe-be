-- +goose Up
-- +goose StatementBegin
-- 029 D-01: indexes for the period-bounded admin queries.
--
-- Every 029 read filters a half-open VN period (`created_at >= ? AND created_at < ?`)
-- and the lists order by `created_at DESC`. Before this migration the only usable
-- index on `deposit_transactions` was the reconciliation partial index, and
-- `subscription_renewal_attempts` had nothing but its primary key.
--
-- Measured on a synthetic database (200k deposits, 100k renewal attempts, 30-day
-- window), EXPLAIN (ANALYZE, BUFFERS):
--   deposits list (no status)   17.5 ms -> 0.10 ms  (Parallel Seq Scan -> Index Scan)
--   deposits list (status IN)   24.1 ms -> 0.20 ms  (Bitmap -> Index Scan)
--   deposits revenue by period  12.9 ms -> 4.2 ms   (Seq Scan -> Bitmap Index Scan)
--   renewal stats by period     16.1 ms -> 2.6 ms   (Seq Scan -> Bitmap Index Scan)
--
-- Deliberately NOT added: `deposit_transactions(status, created_at)`. It moved the
-- revenue aggregate from 4.2 ms to 2.6 ms and nothing else, so it is not worth the
-- write cost on the hottest write path in the service.
CREATE INDEX IF NOT EXISTS idx_deposit_transactions_created_at
    ON subscription.deposit_transactions (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_subscription_renewal_attempts_created_at
    ON subscription.subscription_renewal_attempts (created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS subscription.idx_deposit_transactions_created_at;
DROP INDEX IF EXISTS subscription.idx_subscription_renewal_attempts_created_at;
-- +goose StatementEnd
