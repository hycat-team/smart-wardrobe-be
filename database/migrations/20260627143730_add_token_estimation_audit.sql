-- +goose Up
-- +goose StatementBegin
ALTER TABLE ai_usage_events
    ADD COLUMN estimated_prompt_tokens BIGINT
        CHECK (estimated_prompt_tokens IS NULL OR estimated_prompt_tokens >= 0),
    ADD COLUMN token_estimation_method VARCHAR(30)
        CHECK (token_estimation_method IS NULL OR token_estimation_method IN ('LOCAL', 'PROVIDER_COUNT')),
    ADD COLUMN token_count_latency_ms BIGINT
        CHECK (token_count_latency_ms IS NULL OR token_count_latency_ms >= 0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ai_usage_events
    DROP COLUMN IF EXISTS estimated_prompt_tokens,
    DROP COLUMN IF EXISTS token_estimation_method,
    DROP COLUMN IF EXISTS token_count_latency_ms;
-- +goose StatementEnd
