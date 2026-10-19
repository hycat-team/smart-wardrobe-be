-- +goose Up
-- +goose StatementBegin
-- 030 DDL ownership handover (US1): all ai_* DDL/data below (ai_usage_events
-- token_estimation_method lowercase + CHECK, ai_cost_policies
-- enforcement_mode lowercase + CHECK) are owned by the AI service baseline,
-- which creates the final lowercase state directly. No-op preserves the
-- version row (FR-008); R1 keeps content edits invisible to applied DBs.
SELECT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- 030 handover: Up is now a no-op (DDL lives in the AI baseline), so Down is
-- the exact reverse — nothing to undo.
SELECT 1;
-- +goose StatementEnd
