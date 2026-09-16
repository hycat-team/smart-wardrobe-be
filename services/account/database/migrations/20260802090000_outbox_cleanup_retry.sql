-- +goose Up
-- +goose StatementBegin
-- Outbox cleanup & failed retry:
--   - thêm cột updated_at (mốc backoff retry = lần fail/cập nhật cuối)
--   - thay 2 index đơn (status), (created_at) bằng 2 composite phục vụ đúng query:
--       (status, created_at)     -> ClaimToPublish (filter status + ORDER BY created_at)
--       (status, published_at)   -> DeletePublishedBefore (cleanup published cũ)
-- Bảng outbox_records thuộc schema shared (tách schema Sprint 2).
-- Lưu ý: goose MIGRATION_DSN không có search_path -> phải schema-qualified cho mọi thao tác
-- (index/table đều ở shared, không resolve qua public).

ALTER TABLE shared.outbox_records ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

DROP INDEX IF EXISTS shared.idx_outbox_records_status;
DROP INDEX IF EXISTS shared.idx_outbox_records_created_at;
CREATE INDEX IF NOT EXISTS idx_outbox_records_status_created_at ON shared.outbox_records(status, created_at);
CREATE INDEX IF NOT EXISTS idx_outbox_records_status_published_at ON shared.outbox_records(status, published_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS shared.idx_outbox_records_status_published_at;
DROP INDEX IF EXISTS shared.idx_outbox_records_status_created_at;
CREATE INDEX IF NOT EXISTS idx_outbox_records_status ON shared.outbox_records(status);
CREATE INDEX IF NOT EXISTS idx_outbox_records_created_at ON shared.outbox_records(created_at);

ALTER TABLE shared.outbox_records DROP COLUMN IF EXISTS updated_at;
-- +goose StatementEnd
