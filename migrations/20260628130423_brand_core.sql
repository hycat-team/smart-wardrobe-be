-- +goose Up
CREATE TABLE IF NOT EXISTS brands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    logo_url VARCHAR(500),
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

-- +goose Down
DROP INDEX IF EXISTS idx_brand_customers_brand_status;
DROP INDEX IF EXISTS idx_brand_members_user_status;
DROP INDEX IF EXISTS uq_brand_customers_brand_phone_hash;
DROP INDEX IF EXISTS uq_brand_customers_brand_user;
DROP TABLE IF EXISTS brand_customers;
DROP TABLE IF EXISTS brand_members;
DROP TABLE IF EXISTS brands;
