-- +goose Up
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

-- +goose Down
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
