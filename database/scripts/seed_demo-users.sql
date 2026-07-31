-- ========================================================
-- 50 demo users (demo1 -> demo50), all using the Free plan
-- ========================================================

BEGIN;

INSERT INTO users (
    username,
    email,
    password_hash,
    first_name,
    last_name,
    role_slug,
    status
)
SELECT
    'demo' || sequence_number,
    'demo' || sequence_number || '@hycat.online',
    '$2a$12$PAyZT3QycAqmrD/oEENRpeL0ZRS77va0OJK9rFMucydDZLYFOR.qe',
    'Demo ' || sequence_number,
    'Pre',
    'user',
    0
FROM generate_series(51, 100) AS sequence_number
WHERE NOT EXISTS (
    SELECT 1
    FROM users AS existing_user
    WHERE existing_user.email =
        'demo' || sequence_number || '@hycat.online'
);

INSERT INTO user_subscriptions (
    user_id,
    subscription_plan_id,
    current_plan_code,
    current_tier_rank,
    current_plan_kind,
    current_benefit_snapshot,
    started_at
)
SELECT
    user_account.id,
    free_plan.id,
    free_plan.slug,
    free_plan.tier_rank,
    free_plan.plan_kind,
    jsonb_build_object(
        'maxWardrobeItems', free_plan.max_wardrobe_items,
        'maxOutfits', free_plan.max_outfits,
        'aiOutfitDailyQuota', free_plan.ai_outfit_daily_quota,
        'aiChatDailyQuota', free_plan.ai_chat_daily_quota
    ),
    NOW()
FROM users AS user_account
CROSS JOIN subscription_plans AS free_plan
WHERE free_plan.slug = 'premium-monthly'
  AND free_plan.is_active = TRUE
  AND user_account.username ~ '^demo([1-9]|[1-4][0-9]|50)$'
  AND user_account.email = user_account.username || '@hycat.online'
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO user_daily_quotas (user_id)
SELECT id
FROM users
WHERE username ~ '^demo([1-9]|[1-4][0-9]|50)$'
  AND email = username || '@hycat.online'
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO user_wallets (user_id)
SELECT id
FROM users
WHERE username ~ '^demo([1-9]|[1-4][0-9]|50)$'
  AND email = username || '@hycat.online'
ON CONFLICT (user_id) DO NOTHING;

COMMIT;
