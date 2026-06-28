-- +goose Up
ALTER TABLE outfit_items
    ADD COLUMN IF NOT EXISTS fashion_item_id UUID,
    ADD COLUMN IF NOT EXISTS item_context VARCHAR(50);

UPDATE outfit_items oi
SET
    fashion_item_id = wi.fashion_item_id,
    item_context = 'USER_WARDROBE'
FROM wardrobe_items wi
WHERE wi.id = oi.item_id
  AND oi.fashion_item_id IS NULL;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM outfit_items WHERE fashion_item_id IS NULL) THEN
        RAISE EXCEPTION 'outfit_items.fashion_item_id backfill has NULL values';
    END IF;
    IF EXISTS (SELECT 1 FROM outfit_items WHERE item_context IS NULL) THEN
        RAISE EXCEPTION 'outfit_items.item_context backfill has NULL values';
    END IF;
    IF EXISTS (
        SELECT 1
        FROM outfit_items oi
        LEFT JOIN fashion_items fi ON fi.id = oi.fashion_item_id
        WHERE fi.id IS NULL
    ) THEN
        RAISE EXCEPTION 'outfit_items.fashion_item_id contains values missing from fashion_items';
    END IF;
    IF EXISTS (
        SELECT 1
        FROM outfit_items
        GROUP BY outfit_id, fashion_item_id, item_context
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'outfit_items contains duplicate outfit_id/fashion_item_id/item_context rows';
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE outfit_items
    ALTER COLUMN fashion_item_id SET NOT NULL,
    ALTER COLUMN item_context SET NOT NULL;

-- +goose StatementBegin
DO $$
DECLARE
    constraint_record RECORD;
BEGIN
    FOR constraint_record IN
        SELECT c.conname
        FROM pg_constraint c
        JOIN pg_class t ON t.oid = c.conrelid
        JOIN pg_namespace n ON n.oid = t.relnamespace
        WHERE n.nspname = current_schema()
          AND t.relname = 'outfit_items'
          AND c.contype IN ('p', 'f', 'u')
          AND (
              pg_get_constraintdef(c.oid) ILIKE '%item_id%'
              OR c.contype = 'p'
          )
    LOOP
        EXECUTE format('ALTER TABLE outfit_items DROP CONSTRAINT IF EXISTS %I', constraint_record.conname);
    END LOOP;
END $$;
-- +goose StatementEnd

ALTER TABLE outfit_items
    ADD CONSTRAINT fk_outfit_items_fashion_item
    FOREIGN KEY (fashion_item_id) REFERENCES fashion_items(id)
    ON DELETE RESTRICT;

ALTER TABLE outfit_items
    ADD CONSTRAINT chk_outfit_items_item_context
    CHECK (item_context IN ('USER_WARDROBE', 'BRAND_ITEM'));

ALTER TABLE outfit_items
    ADD PRIMARY KEY (outfit_id, fashion_item_id, item_context);

ALTER TABLE outfit_items
    DROP COLUMN IF EXISTS item_id;

CREATE INDEX IF NOT EXISTS idx_outfit_items_fashion_item_context
ON outfit_items(fashion_item_id, item_context);

-- +goose Down
ALTER TABLE outfit_items
    ADD COLUMN IF NOT EXISTS item_id UUID;

UPDATE outfit_items oi
SET item_id = wi.id
FROM wardrobe_items wi
WHERE wi.fashion_item_id = oi.fashion_item_id
  AND wi.user_id = (
      SELECT o.user_id
      FROM outfits o
      WHERE o.id = oi.outfit_id
  )
  AND oi.item_context = 'USER_WARDROBE'
  AND oi.item_id IS NULL;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM outfit_items WHERE item_id IS NULL) THEN
        RAISE EXCEPTION 'outfit_items.item_id rollback backfill has NULL values';
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE outfit_items
    ALTER COLUMN item_id SET NOT NULL;

DROP INDEX IF EXISTS idx_outfit_items_fashion_item_context;

ALTER TABLE outfit_items
    DROP CONSTRAINT IF EXISTS chk_outfit_items_item_context,
    DROP CONSTRAINT IF EXISTS fk_outfit_items_fashion_item,
    DROP CONSTRAINT IF EXISTS outfit_items_pkey;

ALTER TABLE outfit_items
    ADD CONSTRAINT outfit_items_pkey PRIMARY KEY (outfit_id, item_id);

ALTER TABLE outfit_items
    ADD CONSTRAINT fk_outfit_items_wardrobe_item
    FOREIGN KEY (item_id) REFERENCES wardrobe_items(id)
    ON DELETE CASCADE;

ALTER TABLE outfit_items
    DROP COLUMN IF EXISTS fashion_item_id,
    DROP COLUMN IF EXISTS item_context;
