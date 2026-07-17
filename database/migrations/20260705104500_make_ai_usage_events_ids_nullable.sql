-- +goose Up
-- +goose StatementBegin
ALTER TABLE ai_usage_events ALTER COLUMN ledger_id DROP NOT NULL;
ALTER TABLE ai_usage_events ALTER COLUMN user_id DROP NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ai_usage_events ALTER COLUMN ledger_id SET NOT NULL;
ALTER TABLE ai_usage_events ALTER COLUMN user_id SET NOT NULL;
-- +goose StatementEnd
