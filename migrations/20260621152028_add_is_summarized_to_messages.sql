-- +goose Up
-- +goose StatementBegin
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_summarized BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_messages_context_unsummarized ON messages (context_id) WHERE is_summarized = FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_messages_context_unsummarized;
ALTER TABLE messages DROP COLUMN IF EXISTS is_summarized;
-- +goose StatementEnd
