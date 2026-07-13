-- +goose Up
-- >>> 20260628072715_archive_legacy_community_resale_tables.sql UP <<<
-- Phase 02 B2B2C rebuild: archive legacy community/resale schema after routes and workers
-- have been removed from the MVP runtime.
DROP TABLE IF EXISTS likes;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS transfer_requests;
DROP TABLE IF EXISTS post_media;
DROP TABLE IF EXISTS post_score_snapshots;
DROP TABLE IF EXISTS post_items;
DROP TABLE IF EXISTS posts;

-- >>> 20260628073348_create_fashion_items_and_backfill.sql UP <<<
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

-- >>> 20260628123858_outfit_items_fashion_item_context.sql UP <<<
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

-- >>> 20260628130423_brand_core.sql UP <<<
CREATE TABLE IF NOT EXISTS brands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    logo_url VARCHAR(500),
    logo_public_id VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING_REVIEW',
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    approved_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    CONSTRAINT chk_brands_status CHECK (status IN ('PENDING_REVIEW', 'ACTIVE', 'SUSPENDED', 'ARCHIVED'))
);

CREATE TABLE IF NOT EXISTS brand_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    CONSTRAINT uq_brand_members_brand_user UNIQUE (brand_id, user_id),
    CONSTRAINT chk_brand_members_role CHECK (role IN ('OWNER', 'MANAGER', 'SUPPORT_STAFF', 'MARKETER')),
    CONSTRAINT chk_brand_members_status CHECK (status IN ('ACTIVE', 'INVITED', 'DISABLED'))
);

CREATE TABLE IF NOT EXISTS brand_customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    customer_name VARCHAR(255),
    phone_e164 VARCHAR(50),
    phone_hash VARCHAR(255),
    external_customer_code VARCHAR(100),
    joined_source VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    claimed_at TIMESTAMP WITH TIME ZONE,
    created_by_member_id UUID REFERENCES brand_members(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    CONSTRAINT chk_brand_customers_joined_source CHECK (joined_source IN ('SELF_JOIN', 'OFFLINE_PURCHASE', 'IMPORT')),
    CONSTRAINT chk_brand_customers_status CHECK (status IN ('ACTIVE', 'BLOCKED', 'LEFT'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_brand_customers_brand_user
ON brand_customers(brand_id, user_id)
WHERE user_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_brand_customers_brand_phone_hash
ON brand_customers(brand_id, phone_hash)
WHERE phone_hash IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_brand_members_user_status
ON brand_members(user_id, status);

CREATE INDEX IF NOT EXISTS idx_brand_customers_brand_status
ON brand_customers(brand_id, status);

-- >>> 20260628132100_loyalty_schema.sql UP <<<
CREATE TABLE IF NOT EXISTS loyalty_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    amount_per_point DECIMAL(12,2) NOT NULL,
    point_expiry_days INT,
    rounding_mode VARCHAR(50) NOT NULL DEFAULT 'FLOOR',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    CONSTRAINT chk_loyalty_programs_amount_per_point CHECK (amount_per_point > 0),
    CONSTRAINT chk_loyalty_programs_point_expiry_days CHECK (point_expiry_days IS NULL OR point_expiry_days > 0),
    CONSTRAINT chk_loyalty_programs_rounding_mode CHECK (rounding_mode IN ('FLOOR', 'ROUND', 'CEIL'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_loyalty_programs_active_brand
ON loyalty_programs(brand_id)
WHERE is_active = true;

CREATE TABLE IF NOT EXISTS loyalty_tiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    rank INT NOT NULL,
    min_total_spend DECIMAL(12,2) NOT NULL DEFAULT 0,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    CONSTRAINT uq_loyalty_tiers_brand_rank UNIQUE (brand_id, rank),
    CONSTRAINT uq_loyalty_tiers_brand_name UNIQUE (brand_id, name),
    CONSTRAINT chk_loyalty_tiers_min_total_spend CHECK (min_total_spend >= 0)
);

CREATE TABLE IF NOT EXISTS loyalty_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    brand_customer_id UUID NOT NULL REFERENCES brand_customers(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    current_points INT NOT NULL DEFAULT 0,
    lifetime_points INT NOT NULL DEFAULT 0,
    total_spend DECIMAL(12,2) NOT NULL DEFAULT 0,
    current_tier_id UUID REFERENCES loyalty_tiers(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    CONSTRAINT uq_loyalty_accounts_brand_customer UNIQUE (brand_customer_id),
    CONSTRAINT chk_loyalty_accounts_current_points CHECK (current_points >= 0),
    CONSTRAINT chk_loyalty_accounts_lifetime_points CHECK (lifetime_points >= 0),
    CONSTRAINT chk_loyalty_accounts_total_spend CHECK (total_spend >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_loyalty_accounts_brand_user
ON loyalty_accounts(brand_id, user_id)
WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_loyalty_accounts_brand_tier
ON loyalty_accounts(brand_id, current_tier_id);

CREATE TABLE IF NOT EXISTS loyalty_point_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loyalty_account_id UUID NOT NULL REFERENCES loyalty_accounts(id) ON DELETE CASCADE,
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    brand_customer_id UUID NOT NULL REFERENCES brand_customers(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    points_delta INT NOT NULL,
    balance_after INT NOT NULL,
    transaction_type VARCHAR(50) NOT NULL,
    reason VARCHAR(255),
    spend_amount DECIMAL(12,2),
    reference_type VARCHAR(100),
    reference_id UUID,
    expires_at TIMESTAMP WITH TIME ZONE,
    idempotency_key VARCHAR(100),
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    CONSTRAINT chk_loyalty_point_transactions_type CHECK (transaction_type IN ('EARN', 'REDEEM', 'ADJUST', 'EXPIRE', 'REFUND')),
    CONSTRAINT chk_loyalty_point_transactions_delta_direction CHECK (
        (transaction_type = 'EARN' AND points_delta > 0)
        OR (transaction_type IN ('REDEEM', 'EXPIRE') AND points_delta < 0)
        OR (transaction_type IN ('ADJUST', 'REFUND') AND points_delta <> 0)
    ),
    CONSTRAINT chk_loyalty_point_transactions_balance_after CHECK (balance_after >= 0),
    CONSTRAINT chk_loyalty_point_transactions_spend_amount CHECK (spend_amount IS NULL OR spend_amount >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_loyalty_point_transactions_brand_idempotency
ON loyalty_point_transactions(brand_id, idempotency_key)
WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_loyalty_point_transactions_account_created
ON loyalty_point_transactions(loyalty_account_id, created_at);

CREATE INDEX IF NOT EXISTS idx_loyalty_point_transactions_customer_created
ON loyalty_point_transactions(brand_customer_id, created_at);

CREATE INDEX IF NOT EXISTS idx_loyalty_point_transactions_brand_user
ON loyalty_point_transactions(brand_id, user_id);

CREATE INDEX IF NOT EXISTS idx_loyalty_point_transactions_expires_at
ON loyalty_point_transactions(expires_at)
WHERE expires_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS brand_customer_claims (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_customer_id UUID NOT NULL REFERENCES brand_customers(id) ON DELETE CASCADE,
    claim_token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    consumed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_brand_customer_claims_token_hash
ON brand_customer_claims(claim_token_hash);

CREATE INDEX IF NOT EXISTS idx_brand_customer_claims_customer
ON brand_customer_claims(brand_customer_id);

-- >>> 20260628141000_create_loyalty_point_lots.sql UP <<<
CREATE TABLE IF NOT EXISTS loyalty_point_lots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    loyalty_account_id UUID NOT NULL REFERENCES loyalty_accounts(id) ON DELETE CASCADE,
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    brand_customer_id UUID NOT NULL REFERENCES brand_customers(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,

    earn_transaction_id UUID NOT NULL REFERENCES loyalty_point_transactions(id) ON DELETE RESTRICT,

    earned_points INT NOT NULL,
    remaining_points INT NOT NULL,

    expires_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_loyalty_point_lots_earned_positive
        CHECK (earned_points > 0),

    CONSTRAINT chk_loyalty_point_lots_remaining_non_negative
        CHECK (remaining_points >= 0),

    CONSTRAINT chk_loyalty_point_lots_remaining_lte_earned
        CHECK (remaining_points <= earned_points),

    CONSTRAINT chk_loyalty_point_lots_status
        CHECK (status IN ('ACTIVE', 'CONSUMED', 'EXPIRED'))
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_loyalty_point_lots_earn_transaction
ON loyalty_point_lots(earn_transaction_id);

CREATE INDEX IF NOT EXISTS idx_loyalty_point_lots_account_active_expiry
ON loyalty_point_lots(loyalty_account_id, expires_at)
WHERE status = 'ACTIVE' AND remaining_points > 0;

CREATE INDEX IF NOT EXISTS idx_loyalty_point_lots_expiry_worker
ON loyalty_point_lots(expires_at)
WHERE status = 'ACTIVE' AND remaining_points > 0 AND expires_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_loyalty_point_lots_brand_customer
ON loyalty_point_lots(brand_customer_id);

INSERT INTO loyalty_point_lots (
    loyalty_account_id,
    brand_id,
    brand_customer_id,
    user_id,
    earn_transaction_id,
    earned_points,
    remaining_points,
    expires_at,
    status,
    created_at,
    updated_at
)
SELECT
    t.loyalty_account_id,
    t.brand_id,
    t.brand_customer_id,
    t.user_id,
    t.id,
    t.points_delta,
    t.points_delta,
    t.expires_at,
    CASE
        WHEN t.expires_at IS NOT NULL AND t.expires_at <= NOW() THEN 'EXPIRED'
        ELSE 'ACTIVE'
    END,
    t.created_at,
    NOW()
FROM loyalty_point_transactions t
WHERE t.transaction_type = 'EARN'
  AND t.points_delta > 0
  AND NOT EXISTS (
      SELECT 1
      FROM loyalty_point_lots l
      WHERE l.earn_transaction_id = t.id
  );

-- >>> 20260628142315_create_brand_benefits.sql UP <<<
CREATE TABLE IF NOT EXISTS brand_benefits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    benefit_type VARCHAR(50) NOT NULL,
    unlock_type VARCHAR(50) NOT NULL,
    required_points INT,
    required_tier_id UUID REFERENCES loyalty_tiers(id) ON DELETE SET NULL,
    feature_code VARCHAR(100),
    feature_config JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    CONSTRAINT chk_brand_benefits_benefit_type CHECK (benefit_type IN ('VOUCHER', 'DISCOUNT', 'GIFT', 'FREE_SHIPPING', 'EARLY_ACCESS', 'FEATURE_ACCESS')),
    CONSTRAINT chk_brand_benefits_unlock_type CHECK (unlock_type IN ('TIER_PRIVILEGE', 'POINT_REDEMPTION', 'MANUAL_GRANT')),
    CONSTRAINT chk_brand_benefits_status CHECK (status IN ('ACTIVE', 'INACTIVE', 'ARCHIVED'))
);

CREATE TABLE IF NOT EXISTS benefit_redemptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    benefit_id UUID NOT NULL REFERENCES brand_benefits(id) ON DELETE CASCADE,
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    brand_customer_id UUID NOT NULL REFERENCES brand_customers(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    points_spent INT NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'REDEEMED',
    redeemed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    CONSTRAINT chk_benefit_redemptions_status CHECK (status IN ('PENDING', 'REDEEMED', 'USED', 'CANCELLED', 'EXPIRED'))
);

CREATE INDEX IF NOT EXISTS idx_brand_benefits_brand_status ON brand_benefits(brand_id, status);
CREATE INDEX IF NOT EXISTS idx_benefit_redemptions_user_status ON benefit_redemptions(user_id, status);
CREATE INDEX IF NOT EXISTS idx_benefit_redemptions_customer_status ON benefit_redemptions(brand_customer_id, status);

-- >>> 20260628144413_create_brand_chat.sql UP <<<
CREATE TABLE IF NOT EXISTS brand_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'OPEN',
    last_message_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    CONSTRAINT uq_brand_conversations_brand_user UNIQUE (brand_id, user_id),
    CONSTRAINT chk_brand_conversations_status CHECK (status IN ('OPEN', 'CLOSED'))
);

CREATE TABLE IF NOT EXISTS brand_conversation_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES brand_conversations(id) ON DELETE CASCADE,
    sender_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    sender_role VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    CONSTRAINT chk_brand_conversation_messages_sender_role CHECK (sender_role IN ('CUSTOMER', 'BRAND_STAFF', 'SYSTEM'))
);

CREATE INDEX IF NOT EXISTS idx_brand_conversations_brand_status ON brand_conversations(brand_id, status);
CREATE INDEX IF NOT EXISTS idx_brand_conversation_messages_conv_created ON brand_conversation_messages(conversation_id, created_at);

-- >>> 20260628151416_create_brand_items.sql UP <<<
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

-- >>> 20260628151416_create_brand_items.sql DOWN <<<
DROP INDEX IF EXISTS idx_digital_sample_responses_item_user;
DROP INDEX IF EXISTS idx_brand_items_brand_status;
DROP TABLE IF EXISTS digital_sample_responses;
DROP TABLE IF EXISTS brand_items;

-- >>> 20260628144413_create_brand_chat.sql DOWN <<<
DROP INDEX IF EXISTS idx_brand_conversation_messages_conv_created;
DROP INDEX IF EXISTS idx_brand_conversations_brand_status;
DROP TABLE IF EXISTS brand_conversation_messages;
DROP TABLE IF EXISTS brand_conversations;

-- >>> 20260628142315_create_brand_benefits.sql DOWN <<<
DROP INDEX IF EXISTS idx_benefit_redemptions_customer_status;
DROP INDEX IF EXISTS idx_benefit_redemptions_user_status;
DROP INDEX IF EXISTS idx_brand_benefits_brand_status;
DROP TABLE IF EXISTS benefit_redemptions;
DROP TABLE IF EXISTS brand_benefits;

-- >>> 20260628141000_create_loyalty_point_lots.sql DOWN <<<
DROP INDEX IF EXISTS idx_loyalty_point_lots_brand_customer;
DROP INDEX IF EXISTS idx_loyalty_point_lots_expiry_worker;
DROP INDEX IF EXISTS idx_loyalty_point_lots_account_active_expiry;
DROP INDEX IF EXISTS ux_loyalty_point_lots_earn_transaction;

DROP TABLE IF EXISTS loyalty_point_lots;

-- >>> 20260628132100_loyalty_schema.sql DOWN <<<
DROP INDEX IF EXISTS idx_brand_customer_claims_customer;
DROP INDEX IF EXISTS uq_brand_customer_claims_token_hash;
DROP TABLE IF EXISTS brand_customer_claims;

DROP INDEX IF EXISTS idx_loyalty_point_transactions_expires_at;
DROP INDEX IF EXISTS idx_loyalty_point_transactions_brand_user;
DROP INDEX IF EXISTS idx_loyalty_point_transactions_customer_created;
DROP INDEX IF EXISTS idx_loyalty_point_transactions_account_created;
DROP INDEX IF EXISTS uq_loyalty_point_transactions_brand_idempotency;
DROP TABLE IF EXISTS loyalty_point_transactions;

DROP INDEX IF EXISTS idx_loyalty_accounts_brand_tier;
DROP INDEX IF EXISTS uq_loyalty_accounts_brand_user;
DROP TABLE IF EXISTS loyalty_accounts;

DROP TABLE IF EXISTS loyalty_tiers;

DROP INDEX IF EXISTS uq_loyalty_programs_active_brand;
DROP TABLE IF EXISTS loyalty_programs;

-- >>> 20260628130423_brand_core.sql DOWN <<<
DROP INDEX IF EXISTS idx_brand_customers_brand_status;
DROP INDEX IF EXISTS idx_brand_members_user_status;
DROP INDEX IF EXISTS uq_brand_customers_brand_phone_hash;
DROP INDEX IF EXISTS uq_brand_customers_brand_user;
DROP TABLE IF EXISTS brand_customers;
DROP TABLE IF EXISTS brand_members;
DROP TABLE IF EXISTS brands;

-- >>> 20260628123858_outfit_items_fashion_item_context.sql DOWN <<<
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

-- >>> 20260628073348_create_fashion_items_and_backfill.sql DOWN <<<
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

-- >>> 20260628072715_archive_legacy_community_resale_tables.sql DOWN <<<
CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_id VARCHAR(32) NOT NULL UNIQUE,
    post_type VARCHAR(50) NOT NULL,
    title VARCHAR(255),
    content TEXT NOT NULL,
    contact_info VARCHAR(255),
    total_price DECIMAL(12, 2) DEFAULT 0.00,
    like_count INT DEFAULT 0,
    comment_count INT DEFAULT 0,
    hotness_dirty_at TIMESTAMP WITH TIME ZONE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS post_score_snapshots (
    post_id UUID PRIMARY KEY REFERENCES posts(id) ON DELETE CASCADE,
    global_hotness_score DOUBLE PRECISION DEFAULT 0.0,
    last_calculated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS post_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES wardrobe_items(id) ON DELETE CASCADE,
    price DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    item_condition SMALLINT NOT NULL DEFAULT 1,
    status SMALLINT NOT NULL DEFAULT 1,
    buyer_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    transfer_state SMALLINT NOT NULL DEFAULT 0,
    sold_at TIMESTAMP WITH TIME ZONE,
    declined_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_post_item UNIQUE (post_id, item_id)
);

CREATE TABLE IF NOT EXISTS post_media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    media_type VARCHAR(20) NOT NULL,
    media_url VARCHAR(500) NOT NULL,
    public_id VARCHAR(255),
    sort_order SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transfer_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_item_id UUID NOT NULL REFERENCES post_items(id) ON DELETE CASCADE,
    buyer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_pending_request UNIQUE (post_item_id, buyer_id)
);

CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS likes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_like_target CHECK (
        (post_id IS NOT NULL AND comment_id IS NULL) OR
        (post_id IS NULL AND comment_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_snapshots_global_hotness_score
ON post_score_snapshots (global_hotness_score DESC);

CREATE INDEX IF NOT EXISTS idx_posts_hotness_dirty_at
ON posts (hotness_dirty_at ASC)
WHERE hotness_dirty_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_posts_created_at
ON posts (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_post_items_status
ON post_items (status);

CREATE INDEX IF NOT EXISTS idx_posts_user_id
ON posts (user_id);

CREATE INDEX IF NOT EXISTS idx_post_items_item_id
ON post_items (item_id);

CREATE INDEX IF NOT EXISTS idx_comments_post_id
ON comments (post_id);

CREATE INDEX IF NOT EXISTS idx_likes_post_id
ON likes (post_id)
WHERE post_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uidx_user_liked_post
ON likes (user_id, post_id)
WHERE post_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uidx_user_liked_comment
ON likes (user_id, comment_id)
WHERE comment_id IS NOT NULL;
