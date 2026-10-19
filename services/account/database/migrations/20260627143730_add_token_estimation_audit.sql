-- +goose Up
-- +goose StatementBegin
-- 030 DDL ownership handover (US1): the three ai_usage_events audit columns
-- (estimated_prompt_tokens, token_estimation_method, token_count_latency_ms)
-- are created by the AI service baseline in their final state, so this
-- migration is a no-op. No-op preserves the version row (FR-008); R1 keeps
-- content edits invisible to already-applied databases.
SELECT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- 030 handover: Up is now a no-op (DDL lives in the AI baseline), so Down is
-- the exact reverse — nothing to undo.
SELECT 1;
-- +goose StatementEnd
