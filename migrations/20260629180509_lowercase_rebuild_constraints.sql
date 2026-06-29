-- +goose Up
-- 1. Drop old constraints
ALTER TABLE brand_customers DROP CONSTRAINT IF EXISTS chk_brand_customers_joined_source;
ALTER TABLE loyalty_programs DROP CONSTRAINT IF EXISTS chk_loyalty_programs_rounding_mode;
ALTER TABLE loyalty_point_transactions DROP CONSTRAINT IF EXISTS chk_loyalty_point_transactions_type;
ALTER TABLE loyalty_point_transactions DROP CONSTRAINT IF EXISTS chk_loyalty_point_transactions_delta_direction;
ALTER TABLE brand_benefits DROP CONSTRAINT IF EXISTS chk_brand_benefits_benefit_type;
ALTER TABLE brand_benefits DROP CONSTRAINT IF EXISTS chk_brand_benefits_unlock_type;
ALTER TABLE benefit_redemptions DROP CONSTRAINT IF EXISTS chk_benefit_redemptions_status;
ALTER TABLE brand_conversations DROP CONSTRAINT IF EXISTS chk_brand_conversations_status;
ALTER TABLE brand_conversation_messages DROP CONSTRAINT IF EXISTS chk_brand_conversation_messages_sender_role;
ALTER TABLE brand_items DROP CONSTRAINT IF EXISTS chk_brand_items_type;
ALTER TABLE digital_sample_responses DROP CONSTRAINT IF EXISTS chk_digital_sample_responses_vote_type;
ALTER TABLE outfit_items DROP CONSTRAINT IF EXISTS chk_outfit_items_item_context;

-- 2. Update data to lowercase
UPDATE brand_customers SET joined_source = LOWER(joined_source);
UPDATE loyalty_programs SET rounding_mode = LOWER(rounding_mode);
UPDATE loyalty_point_transactions SET transaction_type = LOWER(transaction_type);
UPDATE brand_benefits SET benefit_type = LOWER(benefit_type), unlock_type = LOWER(unlock_type);
UPDATE benefit_redemptions SET status = LOWER(status);
UPDATE brand_conversations SET status = LOWER(status);
UPDATE brand_conversation_messages SET sender_role = LOWER(sender_role);
UPDATE brand_items SET item_type = LOWER(item_type);
UPDATE digital_sample_responses SET vote_type = LOWER(vote_type);
UPDATE outfit_items SET item_context = LOWER(item_context);

-- 3. Add new constraints (lowercase check)
ALTER TABLE brand_customers ADD CONSTRAINT chk_brand_customers_joined_source CHECK (joined_source IN ('self_join', 'offline_purchase', 'import'));
ALTER TABLE loyalty_programs ADD CONSTRAINT chk_loyalty_programs_rounding_mode CHECK (rounding_mode IN ('floor', 'round', 'ceil'));
ALTER TABLE loyalty_point_transactions ADD CONSTRAINT chk_loyalty_point_transactions_type CHECK (transaction_type IN ('earn', 'redeem', 'adjust', 'expire', 'refund'));
ALTER TABLE loyalty_point_transactions ADD CONSTRAINT chk_loyalty_point_transactions_delta_direction CHECK (
    (transaction_type = 'earn' AND points_delta > 0)
    OR (transaction_type IN ('redeem', 'expire') AND points_delta < 0)
    OR (transaction_type IN ('adjust', 'refund') AND points_delta <> 0)
);
ALTER TABLE brand_benefits ADD CONSTRAINT chk_brand_benefits_benefit_type CHECK (benefit_type IN ('voucher', 'discount', 'gift', 'free_shipping', 'early_access', 'feature_access'));
ALTER TABLE brand_benefits ADD CONSTRAINT chk_brand_benefits_unlock_type CHECK (unlock_type IN ('tier_privilege', 'point_redemption', 'manual_grant'));
ALTER TABLE benefit_redemptions ADD CONSTRAINT chk_benefit_redemptions_status CHECK (status IN ('pending', 'redeemed', 'used', 'cancelled', 'expired'));
ALTER TABLE brand_conversations ADD CONSTRAINT chk_brand_conversations_status CHECK (status IN ('open', 'closed'));
ALTER TABLE brand_conversation_messages ADD CONSTRAINT chk_brand_conversation_messages_sender_role CHECK (sender_role IN ('customer', 'brand_staff', 'system'));
ALTER TABLE brand_items ADD CONSTRAINT chk_brand_items_type CHECK (item_type IN ('product', 'sample'));
ALTER TABLE digital_sample_responses ADD CONSTRAINT chk_digital_sample_responses_vote_type CHECK (vote_type IN ('like', 'dislike', 'would_buy', 'not_interested'));
ALTER TABLE outfit_items ADD CONSTRAINT chk_outfit_items_item_context CHECK (item_context IN ('user_wardrobe', 'brand_item'));

-- +goose Down
-- 1. Drop new constraints
ALTER TABLE brand_customers DROP CONSTRAINT IF EXISTS chk_brand_customers_joined_source;
ALTER TABLE loyalty_programs DROP CONSTRAINT IF EXISTS chk_loyalty_programs_rounding_mode;
ALTER TABLE loyalty_point_transactions DROP CONSTRAINT IF EXISTS chk_loyalty_point_transactions_type;
ALTER TABLE loyalty_point_transactions DROP CONSTRAINT IF EXISTS chk_loyalty_point_transactions_delta_direction;
ALTER TABLE brand_benefits DROP CONSTRAINT IF EXISTS chk_brand_benefits_benefit_type;
ALTER TABLE brand_benefits DROP CONSTRAINT IF EXISTS chk_brand_benefits_unlock_type;
ALTER TABLE benefit_redemptions DROP CONSTRAINT IF EXISTS chk_benefit_redemptions_status;
ALTER TABLE brand_conversations DROP CONSTRAINT IF EXISTS chk_brand_conversations_status;
ALTER TABLE brand_conversation_messages DROP CONSTRAINT IF EXISTS chk_brand_conversation_messages_sender_role;
ALTER TABLE brand_items DROP CONSTRAINT IF EXISTS chk_brand_items_type;
ALTER TABLE digital_sample_responses DROP CONSTRAINT IF EXISTS chk_digital_sample_responses_vote_type;
ALTER TABLE outfit_items DROP CONSTRAINT IF EXISTS chk_outfit_items_item_context;

-- 2. Update data to uppercase
UPDATE brand_customers SET joined_source = UPPER(joined_source);
UPDATE loyalty_programs SET rounding_mode = UPPER(rounding_mode);
UPDATE loyalty_point_transactions SET transaction_type = UPPER(transaction_type);
UPDATE brand_benefits SET benefit_type = UPPER(benefit_type), unlock_type = UPPER(unlock_type);
UPDATE benefit_redemptions SET status = UPPER(status);
UPDATE brand_conversations SET status = UPPER(status);
UPDATE brand_conversation_messages SET sender_role = UPPER(sender_role);
UPDATE brand_items SET item_type = UPPER(item_type);
UPDATE digital_sample_responses SET vote_type = UPPER(vote_type);
UPDATE outfit_items SET item_context = UPPER(item_context);

-- 3. Add old constraints (uppercase check)
ALTER TABLE brand_customers ADD CONSTRAINT chk_brand_customers_joined_source CHECK (joined_source IN ('SELF_JOIN', 'OFFLINE_PURCHASE', 'IMPORT'));
ALTER TABLE loyalty_programs ADD CONSTRAINT chk_loyalty_programs_rounding_mode CHECK (rounding_mode IN ('FLOOR', 'ROUND', 'CEIL'));
ALTER TABLE loyalty_point_transactions ADD CONSTRAINT chk_loyalty_point_transactions_type CHECK (transaction_type IN ('EARN', 'REDEEM', 'ADJUST', 'EXPIRE', 'REFUND'));
ALTER TABLE loyalty_point_transactions ADD CONSTRAINT chk_loyalty_point_transactions_delta_direction CHECK (
    (transaction_type = 'EARN' AND points_delta > 0)
    OR (transaction_type IN ('REDEEM', 'EXPIRE') AND points_delta < 0)
    OR (transaction_type IN ('ADJUST', 'REFUND') AND points_delta <> 0)
);
ALTER TABLE brand_benefits ADD CONSTRAINT chk_brand_benefits_benefit_type CHECK (benefit_type IN ('VOUCHER', 'DISCOUNT', 'GIFT', 'FREE_SHIPPING', 'EARLY_ACCESS', 'FEATURE_ACCESS'));
ALTER TABLE brand_benefits ADD CONSTRAINT chk_brand_benefits_unlock_type CHECK (unlock_type IN ('TIER_PRIVILEGE', 'POINT_REDEMPTION', 'MANUAL_GRANT'));
ALTER TABLE benefit_redemptions ADD CONSTRAINT chk_benefit_redemptions_status CHECK (status IN ('PENDING', 'REDEEMED', 'USED', 'CANCELLED', 'EXPIRED'));
ALTER TABLE brand_conversations ADD CONSTRAINT chk_brand_conversations_status CHECK (status IN ('OPEN', 'CLOSED'));
ALTER TABLE brand_conversation_messages ADD CONSTRAINT chk_brand_conversation_messages_sender_role CHECK (sender_role IN ('CUSTOMER', 'BRAND_STAFF', 'SYSTEM'));
ALTER TABLE brand_items ADD CONSTRAINT chk_brand_items_type CHECK (item_type IN ('PRODUCT', 'SAMPLE'));
ALTER TABLE digital_sample_responses ADD CONSTRAINT chk_digital_sample_responses_vote_type CHECK (vote_type IN ('LIKE', 'DISLIKE', 'WOULD_BUY', 'NOT_INTERESTED'));
ALTER TABLE outfit_items ADD CONSTRAINT chk_outfit_items_item_context CHECK (item_context IN ('USER_WARDROBE', 'BRAND_ITEM'));
