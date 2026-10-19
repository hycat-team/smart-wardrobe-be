-- +goose Up
-- 030 DDL ownership handover (US1): ai_cost_policy_operations quota UPDATEs removed (sections 1-2);
-- final caps are seeded by the AI service baseline. Only the subscription_plans
-- product-quota sync below stays (Account-owned).

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
-- 030 handover Down: ai_cost_policy_operations restores removed (sections 1-2);
-- only the subscription_plans restore below stays (Account-owned).

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
