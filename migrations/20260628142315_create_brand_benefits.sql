-- +goose Up
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

-- +goose Down
DROP INDEX IF EXISTS idx_benefit_redemptions_customer_status;
DROP INDEX IF EXISTS idx_benefit_redemptions_user_status;
DROP INDEX IF EXISTS idx_brand_benefits_brand_status;
DROP TABLE IF EXISTS benefit_redemptions;
DROP TABLE IF EXISTS brand_benefits;
