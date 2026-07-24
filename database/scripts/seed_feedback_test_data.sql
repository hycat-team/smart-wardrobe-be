-- ================================================================
-- Seed: Feedback test data for user 'user' trên brand Thời Trang ABC
-- Yêu cầu: database/init/03-seed.sql + seed_feature_006_dashboard.sql đã chạy
-- Password tất cả account: 123456
--
-- Account test:
--   user         — gọi AI outfit + gửi feedback
--   owner_branda — brand owner (brand portal)
--   staff_branda — brand staff (xem feedback)
-- ================================================================

BEGIN;

-- ============================================================
-- 1. BRAND CUSTOMER (link user → Thời Trang ABC)
-- ============================================================
INSERT INTO brand_customers (id, brand_id, user_id, customer_name, phone_e164, phone_hash, joined_source, status, joined_at, claimed_at, created_at, updated_at)
VALUES (
    'feed0003-0000-0000-0000-000000000001',
    'e7b10001-0000-0000-0000-000000000001', -- Thời Trang ABC
    'ad11ad11-ad11-ad11-ad11-ad11ad11ad12', -- user
    'Feedback Test User',
    '+84901111111',
    'feedback-test-user-phone-hash',
    'self_join',
    'active',
    NOW(),
    NOW(),
    NOW(),
    NOW()
)
ON CONFLICT (id) DO UPDATE SET
    brand_id       = EXCLUDED.brand_id,
    user_id        = EXCLUDED.user_id,
    customer_name  = EXCLUDED.customer_name,
    phone_e164     = EXCLUDED.phone_e164,
    phone_hash     = EXCLUDED.phone_hash,
    joined_source  = EXCLUDED.joined_source,
    status         = EXCLUDED.status,
    joined_at      = EXCLUDED.joined_at,
    claimed_at     = EXCLUDED.claimed_at,
    updated_at     = NOW();

-- ============================================================
-- 2. LOYALTY TIERS
-- ============================================================
INSERT INTO loyalty_tiers (id, brand_id, name, rank, min_total_spend, description, created_at, updated_at)
VALUES
    (
        'fbf10001-0000-0000-0000-000000000001',
        'e7b10001-0000-0000-0000-000000000001', -- Thời Trang ABC
        'Member',
        1,
        0.00,
        'Hạng cơ bản cho mọi khách hàng.',
        NOW(),
        NOW()
    ),
    (
        'fbf10001-0000-0000-0000-000000000002',
        'e7b10001-0000-0000-0000-000000000001',
        'Gold',
        2,
        2000000.00,
        'Hạng Gold cho khách hàng chi tiêu từ 2 triệu.',
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO UPDATE SET
    brand_id        = EXCLUDED.brand_id,
    name            = EXCLUDED.name,
    rank            = EXCLUDED.rank,
    min_total_spend = EXCLUDED.min_total_spend,
    description     = EXCLUDED.description,
    updated_at      = NOW();

-- ============================================================
-- 3. LOYALTY ACCOUNT (user có Gold tier, rank 2)
-- ============================================================
INSERT INTO loyalty_accounts (id, brand_id, brand_customer_id, user_id, current_points, lifetime_points, total_spend, current_tier_id, created_at, updated_at)
VALUES (
    'feed0006-0000-0000-0000-000000000001',
    'e7b10001-0000-0000-0000-000000000001', -- Thời Trang ABC
    'feed0003-0000-0000-0000-000000000001',
    'ad11ad11-ad11-ad11-ad11-ad11ad11ad12',
    500,
    500,
    2500000.00,
    'fbf10001-0000-0000-0000-000000000002', -- Gold tier
    NOW(),
    NOW()
)
ON CONFLICT (id) DO UPDATE SET
    brand_id          = EXCLUDED.brand_id,
    brand_customer_id = EXCLUDED.brand_customer_id,
    user_id           = EXCLUDED.user_id,
    current_points    = EXCLUDED.current_points,
    lifetime_points   = EXCLUDED.lifetime_points,
    total_spend       = EXCLUDED.total_spend,
    current_tier_id   = EXCLUDED.current_tier_id,
    updated_at        = NOW();

-- ============================================================
-- 4. LOYALTY POINT TRANSACTION
-- ============================================================
INSERT INTO loyalty_point_transactions (id, loyalty_account_id, brand_id, brand_customer_id, user_id, points_delta, balance_after, transaction_type, reason, spend_amount, reference_type, expires_at, idempotency_key, created_by_user_id, created_at)
VALUES (
    'feed0007-0000-0000-0000-000000000001',
    'feed0006-0000-0000-0000-000000000001',
    'e7b10001-0000-0000-0000-000000000001',
    'feed0003-0000-0000-0000-000000000001',
    'ad11ad11-ad11-ad11-ad11-ad11ad11ad12',
    500,
    500,
    'earn',
    'Initial seed purchase for feedback test user.',
    2500000.00,
    'seed_demo',
    NOW() + INTERVAL '365 days',
    'feedback-test-user-earn-500',
    'e7b20001-0000-0000-0000-000000000001', -- owner_branda
    NOW()
)
ON CONFLICT (id) DO UPDATE SET
    loyalty_account_id = EXCLUDED.loyalty_account_id,
    brand_id           = EXCLUDED.brand_id,
    brand_customer_id  = EXCLUDED.brand_customer_id,
    user_id            = EXCLUDED.user_id,
    points_delta       = EXCLUDED.points_delta,
    balance_after      = EXCLUDED.balance_after,
    transaction_type   = EXCLUDED.transaction_type,
    reason             = EXCLUDED.reason,
    spend_amount       = EXCLUDED.spend_amount,
    reference_type     = EXCLUDED.reference_type,
    expires_at         = EXCLUDED.expires_at,
    idempotency_key    = EXCLUDED.idempotency_key,
    created_by_user_id = EXCLUDED.created_by_user_id;

-- ============================================================
-- 5. LOYALTY POINT LOT
-- ============================================================
INSERT INTO loyalty_point_lots (id, loyalty_account_id, brand_id, brand_customer_id, user_id, earn_transaction_id, earned_points, remaining_points, expires_at, status, created_at, updated_at)
VALUES (
    'feed0008-0000-0000-0000-000000000001',
    'feed0006-0000-0000-0000-000000000001',
    'e7b10001-0000-0000-0000-000000000001',
    'feed0003-0000-0000-0000-000000000001',
    'ad11ad11-ad11-ad11-ad11-ad11ad11ad12',
    'feed0007-0000-0000-0000-000000000001',
    500,
    500,
    NOW() + INTERVAL '365 days',
    'active',
    NOW(),
    NOW()
)
ON CONFLICT (id) DO UPDATE SET
    loyalty_account_id  = EXCLUDED.loyalty_account_id,
    brand_id            = EXCLUDED.brand_id,
    brand_customer_id   = EXCLUDED.brand_customer_id,
    user_id             = EXCLUDED.user_id,
    earn_transaction_id = EXCLUDED.earn_transaction_id,
    earned_points       = EXCLUDED.earned_points,
    remaining_points    = EXCLUDED.remaining_points,
    expires_at          = EXCLUDED.expires_at,
    status              = EXCLUDED.status,
    updated_at          = NOW();

-- ============================================================
-- 6. BRAND BENEFIT (sample_mix_access)
--    tier_privilege yêu cầu Member (rank 1)
--    User có Gold (rank 2) → 2 >= 1 → access granted
-- ============================================================
INSERT INTO brand_benefits (id, brand_id, name, description, benefit_type, unlock_type, required_tier_id, feature_code, feature_config, status, created_at, updated_at)
VALUES (
    'feed0009-0000-0000-0000-000000000001',
    'e7b10001-0000-0000-0000-000000000001', -- Thời Trang ABC
    'Sample Mix Access',
    'Allows user to try digital samples in AI outfit recommendations.',
    'feature_access',
    'tier_privilege',
    'fbf10001-0000-0000-0000-000000000001', -- Member tier (rank 1)
    'sample_mix_access',
    '{"source":"feedback_seed"}',
    'active',
    NOW(),
    NOW()
)
ON CONFLICT (id) DO UPDATE SET
    brand_id         = EXCLUDED.brand_id,
    name             = EXCLUDED.name,
    description      = EXCLUDED.description,
    benefit_type     = EXCLUDED.benefit_type,
    unlock_type      = EXCLUDED.unlock_type,
    required_tier_id = EXCLUDED.required_tier_id,
    feature_code     = EXCLUDED.feature_code,
    feature_config   = EXCLUDED.feature_config,
    status           = EXCLUDED.status,
    updated_at       = NOW();

-- ============================================================
-- 7. FASHION ITEMS + BRAND ITEMS (product type)
--    Product không cần sample_mix_access, luôn eligible
-- ============================================================
INSERT INTO fashion_items (id, category_id, image_url, image_public_id, color, color_hex,
    color_hue, color_saturation, color_lightness,
    style, material, pattern, fit, seasonality, description,
    embedding, processing_retry_count, processing_version, created_at, updated_at)
VALUES
    (
        'fdb20003-0000-0000-0000-000000000003',
        '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a04', -- Giày
        'https://placehold.co/400x500/FFFFFF/333?text=ABC+Sneaker', '',
        'Trắng', '#FFFFFF', 0, 0, 100,
        'Casual', 'Canvas', 'Solid', 'Regular', 'All season',
        'Giày sneaker trắng ABC, phù hợp mọi phong cách.', NULL, 0, 1, NOW(), NOW()
    ),
    (
        'fdb20004-0000-0000-0000-000000000004',
        '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a06', -- Đầm
        'https://placehold.co/400x500/FFB6C1/333?text=ABC+Dress', '',
        'Pink', '#FFB6C1', 350, 60, 70,
        'Elegant', 'Silk', 'Floral', 'Regular', 'Summer',
        'Đầm hồng ABC thanh lịch, chất lụa cao cấp.', NULL, 0, 1, NOW(), NOW()
    ),
    (
        'fdb20005-0000-0000-0000-000000000005',
        '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a09', -- Chân váy
        'https://placehold.co/400x500/98D8C8/333?text=ABC+Skirt', '',
        'Mint', '#98D8C8', 160, 60, 70,
        'Casual', 'Cotton', 'Solid', 'Regular', 'Summer',
        'Chân váy mint ABC trẻ trung, cotton thoáng mát.', NULL, 0, 1, NOW(), NOW()
    )
ON CONFLICT (id) DO UPDATE SET
    category_id      = EXCLUDED.category_id,
    image_url        = EXCLUDED.image_url,
    color            = EXCLUDED.color,
    color_hex        = EXCLUDED.color_hex,
    color_hue        = EXCLUDED.color_hue,
    color_saturation = EXCLUDED.color_saturation,
    color_lightness  = EXCLUDED.color_lightness,
    style            = EXCLUDED.style,
    material         = EXCLUDED.material,
    pattern          = EXCLUDED.pattern,
    fit              = EXCLUDED.fit,
    seasonality      = EXCLUDED.seasonality,
    description      = EXCLUDED.description,
    updated_at       = NOW();

INSERT INTO brand_items (id, brand_id, fashion_item_id, product_code, name, description, price, item_type, status, created_at, updated_at)
VALUES
    (
        '00000000-0000-0000-0000-0000fdb20003',
        'e7b10001-0000-0000-0000-000000000001',
        'fdb20003-0000-0000-0000-000000000003',
        'ABC-SNEAKER-WHITE',
        'Giày Sneaker Trắng ABC',
        'Giày sneaker trắng thể thao, phù hợp mọi phong cách.',
        450000.00,
        'product',
        'active',
        NOW(),
        NOW()
    ),
    (
        '00000000-0000-0000-0000-0000fdb20004',
        'e7b10001-0000-0000-0000-000000000001',
        'fdb20004-0000-0000-0000-000000000004',
        'ABC-DRESS-PINK',
        'Đầm Hồng ABC',
        'Đầm hồng thanh lịch, chất liệu lụa cao cấp.',
        690000.00,
        'product',
        'active',
        NOW(),
        NOW()
    ),
    (
        '00000000-0000-0000-0000-0000fdb20005',
        'e7b10001-0000-0000-0000-000000000001',
        'fdb20005-0000-0000-0000-000000000005',
        'ABC-SKIRT-MINT',
        'Chân Váy Xanh Mint ABC',
        'Chân váy xanh mint trẻ trung, chất cotton thoáng mát.',
        350000.00,
        'product',
        'active',
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO UPDATE SET
    brand_id        = EXCLUDED.brand_id,
    fashion_item_id = EXCLUDED.fashion_item_id,
    product_code    = EXCLUDED.product_code,
    name            = EXCLUDED.name,
    description     = EXCLUDED.description,
    price           = EXCLUDED.price,
    item_type       = EXCLUDED.item_type,
    status          = EXCLUDED.status,
    updated_at      = NOW();

COMMIT;
