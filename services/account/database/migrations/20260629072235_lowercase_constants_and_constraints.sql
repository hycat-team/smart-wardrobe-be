-- +goose Up
-- 030 DDL ownership handover (US1): ai_* lines removed (ai_cost_policies,
-- user_ai_policy_grants, ai_usage_events are owned by the AI service baseline).
-- brand_* lines below are unchanged.

-- 1. Drop old constraints
ALTER TABLE brands DROP CONSTRAINT IF EXISTS chk_brands_status;
ALTER TABLE brand_members DROP CONSTRAINT IF EXISTS chk_brand_members_status;
ALTER TABLE brand_customers DROP CONSTRAINT IF EXISTS chk_brand_customers_status;
ALTER TABLE loyalty_point_lots DROP CONSTRAINT IF EXISTS chk_loyalty_point_lots_status;
ALTER TABLE brand_benefits DROP CONSTRAINT IF EXISTS chk_brand_benefits_status;
ALTER TABLE brand_items DROP CONSTRAINT IF EXISTS chk_brand_items_status;

-- 2. Update data to lowercase
UPDATE brands SET status = LOWER(status);
UPDATE brand_members SET status = LOWER(status);
UPDATE brand_customers SET status = LOWER(status);
UPDATE loyalty_point_lots SET status = LOWER(status);
UPDATE brand_benefits SET status = LOWER(status);
UPDATE brand_items SET status = LOWER(status);

-- 3. Add new constraints (lowercase check)
ALTER TABLE brands ADD CONSTRAINT chk_brands_status CHECK (status IN ('pending_review', 'active', 'suspended', 'archived'));
ALTER TABLE brand_members ADD CONSTRAINT chk_brand_members_status CHECK (status IN ('active', 'invited', 'disabled'));
ALTER TABLE brand_customers ADD CONSTRAINT chk_brand_customers_status CHECK (status IN ('active', 'blocked', 'left'));
ALTER TABLE loyalty_point_lots ADD CONSTRAINT chk_loyalty_point_lots_status CHECK (status IN ('active', 'consumed', 'expired'));
ALTER TABLE brand_benefits ADD CONSTRAINT chk_brand_benefits_status CHECK (status IN ('active', 'inactive', 'archived'));
ALTER TABLE brand_items ADD CONSTRAINT chk_brand_items_status CHECK (status IN ('draft', 'active', 'archived'));


-- +goose Down
-- 030 handover Down: ai_* lines removed, brand_* lines unchanged.

-- 1. Drop new constraints
ALTER TABLE brands DROP CONSTRAINT IF EXISTS chk_brands_status;
ALTER TABLE brand_members DROP CONSTRAINT IF EXISTS chk_brand_members_status;
ALTER TABLE brand_customers DROP CONSTRAINT IF EXISTS chk_brand_customers_status;
ALTER TABLE loyalty_point_lots DROP CONSTRAINT IF EXISTS chk_loyalty_point_lots_status;
ALTER TABLE brand_benefits DROP CONSTRAINT IF EXISTS chk_brand_benefits_status;
ALTER TABLE brand_items DROP CONSTRAINT IF EXISTS chk_brand_items_status;

-- 2. Update data to uppercase
UPDATE brands SET status = UPPER(status);
UPDATE brand_members SET status = UPPER(status);
UPDATE brand_customers SET status = UPPER(status);
UPDATE loyalty_point_lots SET status = UPPER(status);
UPDATE brand_benefits SET status = UPPER(status);
UPDATE brand_items SET status = UPPER(status);

-- 3. Add old constraints
ALTER TABLE brands ADD CONSTRAINT chk_brands_status CHECK (status IN ('PENDING_REVIEW', 'ACTIVE', 'SUSPENDED', 'ARCHIVED'));
ALTER TABLE brand_members ADD CONSTRAINT chk_brand_members_status CHECK (status IN ('ACTIVE', 'INVITED', 'DISABLED'));
ALTER TABLE brand_customers ADD CONSTRAINT chk_brand_customers_status CHECK (status IN ('ACTIVE', 'BLOCKED', 'LEFT'));
ALTER TABLE loyalty_point_lots ADD CONSTRAINT chk_loyalty_point_lots_status CHECK (status IN ('ACTIVE', 'CONSUMED', 'EXPIRED'));
ALTER TABLE brand_benefits ADD CONSTRAINT chk_brand_benefits_status CHECK (status IN ('ACTIVE', 'INACTIVE', 'ARCHIVED'));
ALTER TABLE brand_items ADD CONSTRAINT chk_brand_items_status CHECK (status IN ('DRAFT', 'ACTIVE', 'ARCHIVED'));
