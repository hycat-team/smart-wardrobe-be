-- +goose Up
ALTER TABLE wardrobe_items ALTER COLUMN fashion_item_id DROP NOT NULL;
ALTER TABLE brand_items ALTER COLUMN fashion_item_id DROP NOT NULL;

ALTER TABLE brand_items DROP CONSTRAINT IF EXISTS uq_brand_items_fashion_item;
CREATE UNIQUE INDEX uq_brand_items_fashion_item ON brand_items(fashion_item_id) WHERE fashion_item_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS uq_brand_items_fashion_item;
UPDATE wardrobe_items SET fashion_item_id = id WHERE fashion_item_id IS NULL;
UPDATE brand_items SET fashion_item_id = gen_random_uuid() WHERE fashion_item_id IS NULL;
ALTER TABLE wardrobe_items ALTER COLUMN fashion_item_id SET NOT NULL;
ALTER TABLE brand_items ALTER COLUMN fashion_item_id SET NOT NULL;
ALTER TABLE brand_items ADD CONSTRAINT uq_brand_items_fashion_item UNIQUE(fashion_item_id);
