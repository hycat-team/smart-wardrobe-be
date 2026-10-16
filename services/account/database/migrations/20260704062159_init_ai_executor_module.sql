-- +goose Up
-- +goose StatementBegin
-- 030 DDL ownership handover (US1): ai_model_configs, the grant table rename,
-- the ai_usage_events alters and the model seed moved to the AI service
-- baseline. This file keeps only the wardrobe/brand task_id columns.
ALTER TABLE wardrobe_items ADD COLUMN task_id UUID;
CREATE INDEX idx_wardrobe_items_task_id ON wardrobe_items(task_id);

ALTER TABLE brand_items ADD COLUMN task_id UUID;
CREATE INDEX idx_brand_items_task_id ON brand_items(task_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_brand_items_task_id;
ALTER TABLE brand_items DROP COLUMN IF EXISTS task_id;

DROP INDEX IF EXISTS idx_wardrobe_items_task_id;
ALTER TABLE wardrobe_items DROP COLUMN IF EXISTS task_id;
-- +goose StatementEnd
