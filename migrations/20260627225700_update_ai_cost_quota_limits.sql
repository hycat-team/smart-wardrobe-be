-- +goose Up
-- Migration to update AI Cost Policy quota limits, token constraints, and synchronize subscription plans quotas.

-- 1. Update Free default AI Cost Policy limits (aa000000-0000-0000-0000-000000000001)
-- Free users get 5 total daily outfit quota, but only 3 of those can be Paid API routes.
UPDATE ai_cost_policy_operations 
SET 
    normal_max_input_tokens = 5000,
    normal_max_output_tokens = 300,
    reduced_max_input_tokens = 4000,
    reduced_max_output_tokens = 250,
    max_paid_attempts_per_day = 3
WHERE policy_id = 'aa000000-0000-0000-0000-000000000001' AND operation = 'outfit';

-- Free users get 3 total daily chat quota, but only 1 of those can be Paid API routes.
UPDATE ai_cost_policy_operations 
SET 
    normal_max_input_tokens = 3000,
    normal_max_output_tokens = 500,
    reduced_max_input_tokens = 2500,
    reduced_max_output_tokens = 400,
    max_paid_attempts_per_day = 3
WHERE policy_id = 'aa000000-0000-0000-0000-000000000001' AND operation = 'chat';


-- 2. Update Premium default AI Cost Policy limits (aa000000-0000-0000-0000-000000000002)
-- Premium users get 30 total daily outfit quota, all 30 can be Paid API routes.
UPDATE ai_cost_policy_operations 
SET 
    normal_max_input_tokens = 6000,
    normal_max_output_tokens = 400,
    reduced_max_input_tokens = 5000,
    reduced_max_output_tokens = 350,
    max_paid_attempts_per_day = 30
WHERE policy_id = 'aa000000-0000-0000-0000-000000000002' AND operation = 'outfit';

-- Premium users get 10 total daily chat quota, all 10 can be Paid API routes.
UPDATE ai_cost_policy_operations 
SET 
    normal_max_input_tokens = 4000,
    normal_max_output_tokens = 600,
    reduced_max_input_tokens = 3500,
    reduced_max_output_tokens = 500,
    max_paid_attempts_per_day = 10
WHERE policy_id = 'aa000000-0000-0000-0000-000000000002' AND operation = 'chat';


-- 3. Synchronize product quotas in subscription_plans table
-- Synchronize Free Plan (slug: 'free') -> 5 outfits, 3 chats daily.
UPDATE subscription_plans
SET 
    ai_outfit_daily_quota = 5,
    ai_chat_daily_quota = 3,
    updated_at = NOW()
WHERE slug = 'free';

-- Synchronize Premium Plan (slug: 'premium-monthly') -> 30 outfits, 10 chats daily.
UPDATE subscription_plans
SET 
    ai_outfit_daily_quota = 30,
    ai_chat_daily_quota = 10,
    updated_at = NOW()
WHERE slug = 'premium-monthly';


-- +goose Down
-- Rollback AI Cost Policy quota limits and subscription plans to original values.

-- 1. Restore Free default AI Cost Policy limits
UPDATE ai_cost_policy_operations 
SET 
    normal_max_input_tokens = 7000,
    normal_max_output_tokens = 400,
    reduced_max_input_tokens = 5000,
    reduced_max_output_tokens = 350,
    max_paid_attempts_per_day = 5
WHERE policy_id = 'aa000000-0000-0000-0000-000000000001' AND operation = 'outfit';

UPDATE ai_cost_policy_operations 
SET 
    normal_max_input_tokens = 3000,
    normal_max_output_tokens = 1000,
    reduced_max_input_tokens = 2500,
    reduced_max_output_tokens = 800,
    max_paid_attempts_per_day = 5
WHERE policy_id = 'aa000000-0000-0000-0000-000000000001' AND operation = 'chat';


-- 2. Restore Premium default AI Cost Policy limits
UPDATE ai_cost_policy_operations 
SET 
    normal_max_input_tokens = 7000,
    normal_max_output_tokens = 400,
    reduced_max_input_tokens = 5000,
    reduced_max_output_tokens = 400,
    max_paid_attempts_per_day = 15
WHERE policy_id = 'aa000000-0000-0000-0000-000000000002' AND operation = 'outfit';

UPDATE ai_cost_policy_operations 
SET 
    normal_max_input_tokens = 4000,
    normal_max_output_tokens = 1000,
    reduced_max_input_tokens = 4000,
    reduced_max_output_tokens = 1000,
    max_paid_attempts_per_day = 20
WHERE policy_id = 'aa000000-0000-0000-0000-000000000002' AND operation = 'chat';


-- 3. Restore product quotas in subscription_plans table
UPDATE subscription_plans
SET 
    ai_outfit_daily_quota = 3,
    ai_chat_daily_quota = 3,
    updated_at = NOW()
WHERE slug = 'free';

UPDATE subscription_plans
SET 
    ai_outfit_daily_quota = 15,
    ai_chat_daily_quota = 20,
    updated_at = NOW()
WHERE slug = 'premium-monthly';
