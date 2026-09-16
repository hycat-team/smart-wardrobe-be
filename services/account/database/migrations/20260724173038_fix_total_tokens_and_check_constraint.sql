-- +goose Up
-- 1. Xoá CHECK constraint cũ (uppercase) trước khi update data
ALTER TABLE ai_usage_events
DROP CONSTRAINT IF EXISTS ai_usage_events_token_estimation_method_check;

-- 2. Chuyển token_estimation_method về lowercase
UPDATE ai_usage_events
SET token_estimation_method = LOWER(token_estimation_method)
WHERE token_estimation_method IS NOT NULL;

-- 3. Thêm CHECK constraint mới (lowercase)
ALTER TABLE ai_usage_events
ADD CONSTRAINT ai_usage_events_token_estimation_method_check
    CHECK (token_estimation_method IS NULL
        OR token_estimation_method IN ('local', 'provider_count'));

-- 4. Xóa ai_cost_policies_check (uppercase 'STRICT')
ALTER TABLE ai_cost_policies
DROP CONSTRAINT IF EXISTS ai_cost_policies_check;

-- 5. Chuẩn hoá enforcement_mode về lowercase
UPDATE ai_cost_policies
SET enforcement_mode = LOWER(enforcement_mode)
WHERE enforcement_mode IS NOT NULL;

-- 6. Thêm ai_cost_policies_check mới (lowercase 'strict')
ALTER TABLE ai_cost_policies
ADD CONSTRAINT ai_cost_policies_check
    CHECK ((enforcement_mode::text <> 'strict'::text) OR (hard_cost_micro_vnd IS NOT NULL));

-- +goose Down
-- 1. Xóa ai_cost_policies_check (lowercase 'strict')
ALTER TABLE ai_cost_policies
DROP CONSTRAINT IF EXISTS ai_cost_policies_check;

-- 2. Chuẩn hoá enforcement_mode về uppercase
UPDATE ai_cost_policies
SET enforcement_mode = UPPER(enforcement_mode)
WHERE enforcement_mode IS NOT NULL;

-- 3. Thêm ai_cost_policies_check cũ (uppercase 'STRICT')
ALTER TABLE ai_cost_policies
ADD CONSTRAINT ai_cost_policies_check
    CHECK ((enforcement_mode::text <> 'STRICT'::text) OR (hard_cost_micro_vnd IS NOT NULL));

-- 4. Xóa CHECK constraint (lowercase)
ALTER TABLE ai_usage_events
DROP CONSTRAINT IF EXISTS ai_usage_events_token_estimation_method_check;

-- 5. Chuyển token_estimation_method về uppercase
UPDATE ai_usage_events
SET token_estimation_method = UPPER(token_estimation_method)
WHERE token_estimation_method IS NOT NULL;

-- 6. Thêm CHECK constraint cũ (uppercase)
ALTER TABLE ai_usage_events
ADD CONSTRAINT ai_usage_events_token_estimation_method_check
    CHECK (token_estimation_method IS NULL
        OR token_estimation_method IN ('LOCAL', 'PROVIDER_COUNT'));
