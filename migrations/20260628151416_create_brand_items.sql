-- +goose Up
CREATE TABLE IF NOT EXISTS brand_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    fashion_item_id UUID NOT NULL REFERENCES fashion_items(id) ON DELETE CASCADE,
    product_code VARCHAR(100) NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NULL,
    price DECIMAL(12,2) NULL,
    item_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    CONSTRAINT uq_brand_items_fashion_item UNIQUE(fashion_item_id),
    CONSTRAINT uq_brand_items_brand_product_code UNIQUE(brand_id, product_code),
    CONSTRAINT chk_brand_items_type CHECK (item_type IN ('PRODUCT', 'SAMPLE')),
    CONSTRAINT chk_brand_items_status CHECK (status IN ('DRAFT', 'ACTIVE', 'ARCHIVED'))
);

CREATE TABLE IF NOT EXISTS digital_sample_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_item_id UUID NOT NULL REFERENCES brand_items(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    outfit_id UUID REFERENCES outfits(id) ON DELETE SET NULL,
    vote_type VARCHAR(50) NULL,
    rating INT NULL,
    feedback_text TEXT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    CONSTRAINT chk_digital_sample_responses_vote_type CHECK (vote_type IN ('LIKE', 'DISLIKE', 'WOULD_BUY', 'NOT_INTERESTED'))
);

CREATE INDEX IF NOT EXISTS idx_brand_items_brand_status ON brand_items(brand_id, status);
CREATE INDEX IF NOT EXISTS idx_digital_sample_responses_item_user ON digital_sample_responses(brand_item_id, user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_digital_sample_responses_item_user;
DROP INDEX IF EXISTS idx_brand_items_brand_status;
DROP TABLE IF EXISTS digital_sample_responses;
DROP TABLE IF EXISTS brand_items;
