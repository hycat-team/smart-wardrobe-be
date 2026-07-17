-- +goose Up
-- 1. Drop old constraints
ALTER TABLE brands DROP CONSTRAINT IF EXISTS chk_brands_status;
ALTER TABLE brand_members DROP CONSTRAINT IF EXISTS chk_brand_members_status;
ALTER TABLE brand_customers DROP CONSTRAINT IF EXISTS chk_brand_customers_status;
ALTER TABLE loyalty_point_lots DROP CONSTRAINT IF EXISTS chk_loyalty_point_lots_status;
ALTER TABLE brand_benefits DROP CONSTRAINT IF EXISTS chk_brand_benefits_status;
ALTER TABLE brand_items DROP CONSTRAINT IF EXISTS chk_brand_items_status;
ALTER TABLE ai_cost_policies DROP CONSTRAINT IF EXISTS ai_cost_policies_enforcement_mode_check;
ALTER TABLE user_ai_policy_grants DROP CONSTRAINT IF EXISTS user_ai_policy_grants_status_check;
ALTER TABLE ai_usage_events DROP CONSTRAINT IF EXISTS ai_usage_events_status_check;

-- 2. Update data to lowercase
UPDATE brands SET status = LOWER(status);
UPDATE brand_members SET status = LOWER(status);
UPDATE brand_customers SET status = LOWER(status);
UPDATE loyalty_point_lots SET status = LOWER(status);
UPDATE brand_benefits SET status = LOWER(status);
UPDATE brand_items SET status = LOWER(status);
UPDATE ai_cost_policies SET enforcement_mode = LOWER(enforcement_mode);
UPDATE user_ai_policy_grants SET status = LOWER(status);
UPDATE ai_usage_events SET status = LOWER(status);

-- 3. Add new constraints (lowercase check)
ALTER TABLE brands ADD CONSTRAINT chk_brands_status CHECK (status IN ('pending_review', 'active', 'suspended', 'archived'));
ALTER TABLE brand_members ADD CONSTRAINT chk_brand_members_status CHECK (status IN ('active', 'invited', 'disabled'));
ALTER TABLE brand_customers ADD CONSTRAINT chk_brand_customers_status CHECK (status IN ('active', 'blocked', 'left'));
ALTER TABLE loyalty_point_lots ADD CONSTRAINT chk_loyalty_point_lots_status CHECK (status IN ('active', 'consumed', 'expired'));
ALTER TABLE brand_benefits ADD CONSTRAINT chk_brand_benefits_status CHECK (status IN ('active', 'inactive', 'archived'));
ALTER TABLE brand_items ADD CONSTRAINT chk_brand_items_status CHECK (status IN ('draft', 'active', 'archived'));
ALTER TABLE ai_cost_policies ADD CONSTRAINT ai_cost_policies_enforcement_mode_check CHECK (enforcement_mode IN ('strict', 'observe_only', 'free_only'));
ALTER TABLE user_ai_policy_grants ADD CONSTRAINT user_ai_policy_grants_status_check CHECK (status IN ('active', 'future', 'closed'));
ALTER TABLE ai_usage_events ADD CONSTRAINT ai_usage_events_status_check CHECK (status IN ('reserved', 'in_flight', 'confirmed', 'released', 'unknown_usage', 'expired_unverified'));

-- 4. Recreate partial index with lowercase filter
DROP INDEX IF EXISTS ix_ai_usage_events_unknown;
CREATE INDEX ix_ai_usage_events_unknown ON ai_usage_events(status, unknown_expires_at) WHERE status IN ('reserved', 'unknown_usage');

-- +goose Down
-- 1. Drop new constraints
ALTER TABLE brands DROP CONSTRAINT IF EXISTS chk_brands_status;
ALTER TABLE brand_members DROP CONSTRAINT IF EXISTS chk_brand_members_status;
ALTER TABLE brand_customers DROP CONSTRAINT IF EXISTS chk_brand_customers_status;
ALTER TABLE loyalty_point_lots DROP CONSTRAINT IF EXISTS chk_loyalty_point_lots_status;
ALTER TABLE brand_benefits DROP CONSTRAINT IF EXISTS chk_brand_benefits_status;
ALTER TABLE brand_items DROP CONSTRAINT IF EXISTS chk_brand_items_status;
ALTER TABLE ai_cost_policies DROP CONSTRAINT IF EXISTS ai_cost_policies_enforcement_mode_check;
ALTER TABLE user_ai_policy_grants DROP CONSTRAINT IF EXISTS user_ai_policy_grants_status_check;
ALTER TABLE ai_usage_events DROP CONSTRAINT IF EXISTS ai_usage_events_status_check;

-- 2. Update data to uppercase
UPDATE brands SET status = UPPER(status);
UPDATE brand_members SET status = UPPER(status);
UPDATE brand_customers SET status = UPPER(status);
UPDATE loyalty_point_lots SET status = UPPER(status);
UPDATE brand_benefits SET status = UPPER(status);
UPDATE brand_items SET status = UPPER(status);
UPDATE ai_cost_policies SET enforcement_mode = UPPER(enforcement_mode);
UPDATE user_ai_policy_grants SET status = UPPER(status);
UPDATE ai_usage_events SET status = UPPER(status);

-- 3. Add old constraints
ALTER TABLE brands ADD CONSTRAINT chk_brands_status CHECK (status IN ('PENDING_REVIEW', 'ACTIVE', 'SUSPENDED', 'ARCHIVED'));
ALTER TABLE brand_members ADD CONSTRAINT chk_brand_members_status CHECK (status IN ('ACTIVE', 'INVITED', 'DISABLED'));
ALTER TABLE brand_customers ADD CONSTRAINT chk_brand_customers_status CHECK (status IN ('ACTIVE', 'BLOCKED', 'LEFT'));
ALTER TABLE loyalty_point_lots ADD CONSTRAINT chk_loyalty_point_lots_status CHECK (status IN ('ACTIVE', 'CONSUMED', 'EXPIRED'));
ALTER TABLE brand_benefits ADD CONSTRAINT chk_brand_benefits_status CHECK (status IN ('ACTIVE', 'INACTIVE', 'ARCHIVED'));
ALTER TABLE brand_items ADD CONSTRAINT chk_brand_items_status CHECK (status IN ('DRAFT', 'ACTIVE', 'ARCHIVED'));
ALTER TABLE ai_cost_policies ADD CONSTRAINT ai_cost_policies_enforcement_mode_check CHECK (enforcement_mode IN ('STRICT', 'OBSERVE_ONLY', 'FREE_ONLY'));
ALTER TABLE user_ai_policy_grants ADD CONSTRAINT user_ai_policy_grants_status_check CHECK (status IN ('ACTIVE', 'FUTURE', 'CLOSED'));
ALTER TABLE ai_usage_events ADD CONSTRAINT ai_usage_events_status_check CHECK (status IN ('RESERVED', 'IN_FLIGHT', 'CONFIRMED', 'RELEASED', 'UNKNOWN_USAGE', 'EXPIRED_UNVERIFIED'));

DROP INDEX IF EXISTS ix_ai_usage_events_unknown;
CREATE INDEX ix_ai_usage_events_unknown ON ai_usage_events(status, unknown_expires_at) WHERE status IN ('RESERVED', 'UNKNOWN_USAGE');
