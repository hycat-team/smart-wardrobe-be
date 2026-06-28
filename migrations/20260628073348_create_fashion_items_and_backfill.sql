-- +goose Up
CREATE TABLE IF NOT EXISTS fashion_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID REFERENCES categories(id) ON DELETE RESTRICT,
    image_url VARCHAR(500) NOT NULL,
    image_public_id VARCHAR(255) NOT NULL,
    color VARCHAR(50),
    color_hex VARCHAR(7),
    color_hue DOUBLE PRECISION,
    color_saturation DOUBLE PRECISION,
    color_lightness DOUBLE PRECISION,
    style VARCHAR(100),
    material VARCHAR(100),
    pattern VARCHAR(100),
    fit VARCHAR(50),
    seasonality VARCHAR(100),
    description TEXT,
    embedding VECTOR(768),
    processing_retry_count INT NOT NULL DEFAULT 0,
    processing_version INT NOT NULL DEFAULT 0,
    processing_started_at TIMESTAMP WITH TIME ZONE,
    last_processing_attempt_at TIMESTAMP WITH TIME ZONE,
    processing_error_reason TEXT,
    review_reason VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

INSERT INTO fashion_items (
    id,
    category_id,
    image_url,
    image_public_id,
    color,
    color_hex,
    color_hue,
    color_saturation,
    color_lightness,
    style,
    material,
    pattern,
    fit,
    seasonality,
    description,
    embedding,
    processing_retry_count,
    processing_version,
    processing_started_at,
    last_processing_attempt_at,
    processing_error_reason,
    review_reason,
    created_at,
    updated_at
)
SELECT
    id,
    category_id,
    image_url,
    image_public_id,
    color,
    color_hex,
    color_hue,
    color_saturation,
    color_lightness,
    style,
    material,
    pattern,
    fit,
    seasonality,
    description,
    embedding,
    processing_retry_count,
    processing_version,
    processing_started_at,
    last_processing_attempt_at,
    processing_error_reason,
    review_reason,
    created_at,
    updated_at
FROM wardrobe_items
ON CONFLICT (id) DO NOTHING;

ALTER TABLE wardrobe_items
    ADD COLUMN IF NOT EXISTS fashion_item_id UUID;

UPDATE wardrobe_items
SET fashion_item_id = id
WHERE fashion_item_id IS NULL;

ALTER TABLE wardrobe_items
    ALTER COLUMN fashion_item_id SET NOT NULL;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_wardrobe_items_fashion_item'
    ) THEN
        ALTER TABLE wardrobe_items
            ADD CONSTRAINT fk_wardrobe_items_fashion_item
            FOREIGN KEY (fashion_item_id) REFERENCES fashion_items(id)
            ON DELETE RESTRICT;
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE wardrobe_items
    RENAME COLUMN price TO purchase_price;

CREATE INDEX IF NOT EXISTS idx_wardrobe_items_user_fashion_item
ON wardrobe_items(user_id, fashion_item_id);

CREATE INDEX IF NOT EXISTS idx_fashion_items_category_id
ON fashion_items(category_id);

CREATE INDEX IF NOT EXISTS fitems_embedding_cosine_idx
ON fashion_items
USING hnsw (embedding vector_cosine_ops);

CREATE INDEX IF NOT EXISTS idx_fashion_items_lexical_search
ON fashion_items
USING gin (
  to_tsvector('simple', immutable_unaccent(lower(
    coalesce(color, '') || ' ' ||
    coalesce(style, '') || ' ' ||
    coalesce(material, '') || ' ' ||
    coalesce(pattern, '') || ' ' ||
    coalesce(fit, '') || ' ' ||
    coalesce(seasonality, '') || ' ' ||
    coalesce(description, '')
  )))
);

ALTER TABLE wardrobe_items
    DROP COLUMN IF EXISTS category_id,
    DROP COLUMN IF EXISTS image_url,
    DROP COLUMN IF EXISTS image_public_id,
    DROP COLUMN IF EXISTS color,
    DROP COLUMN IF EXISTS color_hex,
    DROP COLUMN IF EXISTS color_hue,
    DROP COLUMN IF EXISTS color_saturation,
    DROP COLUMN IF EXISTS color_lightness,
    DROP COLUMN IF EXISTS style,
    DROP COLUMN IF EXISTS material,
    DROP COLUMN IF EXISTS pattern,
    DROP COLUMN IF EXISTS fit,
    DROP COLUMN IF EXISTS seasonality,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS embedding,
    DROP COLUMN IF EXISTS processing_retry_count,
    DROP COLUMN IF EXISTS processing_version,
    DROP COLUMN IF EXISTS processing_started_at,
    DROP COLUMN IF EXISTS last_processing_attempt_at,
    DROP COLUMN IF EXISTS processing_error_reason,
    DROP COLUMN IF EXISTS review_reason;

-- +goose Down
ALTER TABLE wardrobe_items
    ADD COLUMN IF NOT EXISTS category_id UUID,
    ADD COLUMN IF NOT EXISTS image_url VARCHAR(500),
    ADD COLUMN IF NOT EXISTS image_public_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS color VARCHAR(50),
    ADD COLUMN IF NOT EXISTS color_hex VARCHAR(7),
    ADD COLUMN IF NOT EXISTS color_hue DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS color_saturation DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS color_lightness DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS style VARCHAR(100),
    ADD COLUMN IF NOT EXISTS material VARCHAR(100),
    ADD COLUMN IF NOT EXISTS pattern VARCHAR(100),
    ADD COLUMN IF NOT EXISTS fit VARCHAR(50),
    ADD COLUMN IF NOT EXISTS seasonality VARCHAR(100),
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS embedding VECTOR(768),
    ADD COLUMN IF NOT EXISTS processing_retry_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS processing_version INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS processing_started_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS last_processing_attempt_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS processing_error_reason TEXT,
    ADD COLUMN IF NOT EXISTS review_reason VARCHAR(100);

UPDATE wardrobe_items wi
SET
    category_id = fi.category_id,
    image_url = fi.image_url,
    image_public_id = fi.image_public_id,
    color = fi.color,
    color_hex = fi.color_hex,
    color_hue = fi.color_hue,
    color_saturation = fi.color_saturation,
    color_lightness = fi.color_lightness,
    style = fi.style,
    material = fi.material,
    pattern = fi.pattern,
    fit = fi.fit,
    seasonality = fi.seasonality,
    description = fi.description,
    embedding = fi.embedding,
    processing_retry_count = fi.processing_retry_count,
    processing_version = fi.processing_version,
    processing_started_at = fi.processing_started_at,
    last_processing_attempt_at = fi.last_processing_attempt_at,
    processing_error_reason = fi.processing_error_reason,
    review_reason = fi.review_reason
FROM fashion_items fi
WHERE fi.id = wi.fashion_item_id;

ALTER TABLE wardrobe_items
    ALTER COLUMN image_url SET NOT NULL,
    ALTER COLUMN image_public_id SET NOT NULL;

DROP INDEX IF EXISTS idx_fashion_items_lexical_search;
DROP INDEX IF EXISTS fitems_embedding_cosine_idx;
DROP INDEX IF EXISTS idx_fashion_items_category_id;
DROP INDEX IF EXISTS idx_wardrobe_items_user_fashion_item;

ALTER TABLE wardrobe_items
    RENAME COLUMN purchase_price TO price;

ALTER TABLE wardrobe_items
    DROP CONSTRAINT IF EXISTS fk_wardrobe_items_fashion_item;

ALTER TABLE wardrobe_items
    DROP COLUMN IF EXISTS fashion_item_id;

DROP TABLE IF EXISTS fashion_items;
