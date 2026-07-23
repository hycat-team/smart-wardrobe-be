-- Seed Data: Feature 006 — Dashboard & Analytics
-- Chạy sau khi đã chạy seed_feature_006_cleanup.sql
-- Yêu cầu: init/03-seed.sql đã chạy (users, categories, subscription_plans)
-- Toàn bộ seed nằm trong 1 transaction — nếu lỗi sẽ ROLLBACK toàn bộ

BEGIN;

-- ============================================================
-- PHASE 1: Users & Brands
-- ============================================================

-- Thêm 2 users cho brand test (Owner + Staff)
INSERT INTO users (id, username, email, password_hash, role_slug, status, is_deleted, created_at, updated_at)
VALUES
    ('e7b20001-0000-0000-0000-000000000001', 'owner_branda', 'owner@branda.com',
     '$2a$11$kXWLREY8wu6wEQlONWcLveV2jeE/Tx9MS4vOlQqXmcQ9VASP0NMhu', 'user', 0, false, NOW(), NOW()),
    ('e7b20002-0000-0000-0000-000000000002', 'staff_branda', 'staff@branda.com',
     '$2a$11$kXWLREY8wu6wEQlONWcLveV2jeE/Tx9MS4vOlQqXmcQ9VASP0NMhu', 'user', 0, false, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Brands
INSERT INTO brands (id, slug, name, description, status, created_by_user_id, created_at, updated_at)
VALUES
    ('e7b10001-0000-0000-0000-000000000001', 'thoi-trang-abc', 'Thời Trang ABC', 'Thương hiệu thời trang hàng đầu Việt Nam', 'active',
     'e7b20001-0000-0000-0000-000000000001', NOW(), NOW()),
    ('e7b10002-0000-0000-0000-000000000002', 'fashion-xyz', 'Fashion XYZ', 'Thương hiệu thời trang trẻ trung', 'active',
     'ad11ad11-ad11-ad11-ad11-ad11ad11ad11', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Brand Members
INSERT INTO brand_members (id, brand_id, user_id, role, status, created_at, updated_at)
VALUES
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001', 'e7b20001-0000-0000-0000-000000000001', 'owner', 'active', NOW(), NOW()),
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001', 'e7b20002-0000-0000-0000-000000000002', 'staff', 'active', NOW(), NOW()),
    (gen_random_uuid(), 'e7b10002-0000-0000-0000-000000000002', 'ad11ad11-ad11-ad11-ad11-ad11ad11ad11', 'owner', 'active', NOW(), NOW());

-- Loyalty Programs
INSERT INTO loyalty_programs (id, brand_id, name, amount_per_point, point_expiry_days, is_active, created_at, updated_at)
VALUES
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001', 'Chương trình tích điểm ABC', 1000, 365, true, NOW(), NOW()),
    (gen_random_uuid(), 'e7b10002-0000-0000-0000-000000000002', 'XYZ Rewards', 1000, 180, true, NOW(), NOW());

-- Brand Customers (50 cho Brand A, 10 cho Brand B)
DO $$
DECLARE
    i INT;
BEGIN
    FOR i IN 1..50 LOOP
        INSERT INTO brand_customers (id, brand_id, user_id, customer_name, phone_e164, joined_source, status, created_at, updated_at)
        VALUES (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001', NULL,
                'Khách hàng ' || i, '8490000000' || LPAD(i::text, 2, '0'), 'direct', 'active', NOW(), NOW());
    END LOOP;
    FOR i IN 1..10 LOOP
        INSERT INTO brand_customers (id, brand_id, user_id, customer_name, phone_e164, joined_source, status, created_at, updated_at)
        VALUES (gen_random_uuid(), 'e7b10002-0000-0000-0000-000000000002', NULL,
                'XYZ Customer ' || i, '8491000000' || LPAD(i::text, 2, '0'), 'direct', 'active', NOW(), NOW());
    END LOOP;
END $$;

-- Brand Benefits (voucher-type)
INSERT INTO brand_benefits (id, brand_id, name, description, benefit_type, unlock_type, required_points, status, created_at, updated_at)
VALUES
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001', 'Giảm 10%', 'Giảm 10% cho đơn hàng tiếp theo', 'voucher', 'points', 500, 'active', NOW(), NOW()),
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001', 'Giảm 50K', 'Giảm 50.000đ cho đơn từ 500K', 'voucher', 'points', 1000, 'active', NOW(), NOW()),
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001', 'Miễn phí vận chuyển', 'Miễn phí vận chuyển toàn quốc', 'voucher', 'points', 300, 'active', NOW(), NOW()),
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001', 'Quà tặng sinh nhật', 'Món quà đặc biệt nhân dịp sinh nhật', 'gift', 'tier', 0, 'active', NOW(), NOW()),
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001', 'Giảm 20%', 'Giảm 20% cho đơn hàng mùa hè', 'voucher', 'points', 2000, 'active', NOW(), NOW());

-- Benefit Redemptions
INSERT INTO benefit_redemptions (id, benefit_id, brand_id, brand_customer_id, user_id, points_spent, status, redeemed_at, created_at, updated_at)
SELECT gen_random_uuid(), b.id, b.brand_id, bc.id, NULL, b.required_points, 'redeemed',
       NOW() - (random() * 30 || ' days')::interval, NOW(), NOW()
FROM brand_benefits b
CROSS JOIN LATERAL (SELECT id FROM brand_customers WHERE brand_id = b.brand_id ORDER BY random() LIMIT 2) bc
WHERE b.required_points > 0;

-- ============================================================
-- PHASE 2: Subscription Layer
-- ============================================================

-- User Subscription (user@closy.com → premium-monthly)
INSERT INTO user_subscriptions (user_id, subscription_plan_id, expires_at, is_auto_renew_enabled, created_at, updated_at,
                                current_plan_code, current_tier_rank, current_plan_kind, current_benefit_snapshot, started_at)
SELECT 'ad11ad11-ad11-ad11-ad11-ad11ad11ad12', id, NOW() + INTERVAL '30 days', true, NOW(), NOW(),
       slug, tier_rank, plan_kind, '{}'::jsonb, NOW()
FROM subscription_plans
WHERE slug = 'premium-monthly'
ON CONFLICT (user_id) DO UPDATE SET
    subscription_plan_id = EXCLUDED.subscription_plan_id,
    expires_at = EXCLUDED.expires_at,
    is_auto_renew_enabled = EXCLUDED.is_auto_renew_enabled;

-- User Daily Quota
INSERT INTO user_daily_quotas (user_id, outfit_recommend_count, ai_usage_count, last_reset_date, created_at, updated_at)
VALUES ('ad11ad11-ad11-ad11-ad11-ad11ad11ad12', 5, 8, CURRENT_DATE, NOW(), NOW())
ON CONFLICT (user_id) DO UPDATE SET
    outfit_recommend_count = 5, ai_usage_count = 8, last_reset_date = CURRENT_DATE;

-- User Wallet
INSERT INTO user_wallets (user_id, balance, currency, created_at, updated_at)
VALUES ('ad11ad11-ad11-ad11-ad11-ad11ad11ad12', 500000.00, 'VND', NOW(), NOW())
ON CONFLICT (user_id) DO NOTHING;

-- Deposit Transactions (10 dòng, direct_purchase, status=1 Success)
INSERT INTO deposit_transactions (id, user_id, amount, currency, status, transaction_type, order_code,
                                  provider, created_at, updated_at, expected_amount)
SELECT
    gen_random_uuid(), 'ad11ad11-ad11-ad11-ad11-ad11ad11ad12',
    (random() * 950000 + 50000)::numeric(12,2), 'VND', 1, 'direct_purchase',
    nextval('deposit_transactions_order_code_seq'),
    'PAYOS', NOW() - (n || ' days')::interval, NOW(), 0
FROM generate_series(0, 9) AS n;

-- ============================================================
-- PHASE 3: Brand Business (Points, Vouchers, Conversations)
-- ============================================================

-- Loyalty Accounts (1 account per customer, 10 customers)
DO $$
DECLARE
    cust RECORD;
    acc_id UUID;
    txn_ids UUID[];
BEGIN
    FOR cust IN (SELECT id FROM brand_customers WHERE brand_id = 'e7b10001-0000-0000-0000-000000000001' ORDER BY random() LIMIT 10) LOOP
        acc_id := gen_random_uuid();
        INSERT INTO loyalty_accounts (id, brand_id, brand_customer_id, current_points, lifetime_points, total_spend, created_at, updated_at)
        VALUES (acc_id, 'e7b10001-0000-0000-0000-000000000001', cust.id, 5000, 50000, 5000000.00, NOW(), NOW());

        -- 1st EARN transaction (capture ID for earn_transaction_id)
        WITH inserted AS (
            INSERT INTO loyalty_point_transactions (id, loyalty_account_id, brand_id, brand_customer_id, points_delta, balance_after, transaction_type, created_at)
            VALUES (gen_random_uuid(), acc_id, 'e7b10001-0000-0000-0000-000000000001', cust.id, 3000, 3000, 'EARN', NOW() - INTERVAL '30 days')
            RETURNING id
        ) SELECT array_agg(id) INTO txn_ids FROM inserted;

        -- 2nd EARN transaction
        INSERT INTO loyalty_point_transactions (id, loyalty_account_id, brand_id, brand_customer_id, points_delta, balance_after, transaction_type, created_at)
        VALUES (gen_random_uuid(), acc_id, 'e7b10001-0000-0000-0000-000000000001', cust.id, 2000, 5000, 'EARN', NOW() - INTERVAL '15 days');

        -- 1 REDEEM transaction per account (tài khoản chẵn)
        IF (cust.id::text) LIKE '%0' OR (cust.id::text) LIKE '%2' OR (cust.id::text) LIKE '%4' THEN
            INSERT INTO loyalty_point_transactions (id, loyalty_account_id, brand_id, brand_customer_id, points_delta, balance_after, transaction_type, created_at)
            VALUES (gen_random_uuid(), acc_id, 'e7b10001-0000-0000-0000-000000000001', cust.id, -500, 4500, 'REDEEM', NOW() - INTERVAL '7 days');
        END IF;

        -- Loyalty Point Lots (1 lot per account, reference real earn_transaction_id)
        INSERT INTO loyalty_point_lots (id, loyalty_account_id, brand_id, brand_customer_id, user_id, earn_transaction_id,
                                        earned_points, remaining_points, expires_at, status, created_at, updated_at)
        VALUES (
            gen_random_uuid(), acc_id, 'e7b10001-0000-0000-0000-000000000001', cust.id, NULL, txn_ids[1],
            5000, 3000,
            CASE WHEN random() < 0.3 THEN NOW() + (floor(random() * 25 + 5) || ' days')::interval
                 ELSE NOW() + (floor(random() * 300 + 30) || ' days')::interval END,
            'active', NOW(), NOW()
        );
    END LOOP;
END $$;

-- Customers for conversations (6 additional users to avoid UNIQUE(brand_id, user_id) conflict)
INSERT INTO users (id, username, email, password_hash, role_slug, status, is_deleted, created_at, updated_at)
SELECT ('e7b3000' || n || '-0000-0000-0000-000000000000')::uuid, 'conv_user_' || n, 'conv' || n || '@test.com',
       '$2a$11$kXWLREY8wu6wEQlONWcLveV2jeE/Tx9MS4vOlQqXmcQ9VASP0NMhu', 'user', 0, false, NOW(), NOW()
FROM generate_series(1, 6) AS n
ON CONFLICT (id) DO NOTHING;

-- Brand Conversations (10 conversations — mỗi user khác nhau do UNIQUE(brand_id, user_id))
INSERT INTO brand_conversations (id, brand_id, user_id, status, last_message_at, created_at, updated_at,
                                 user_last_read_at, staff_last_read_at)
VALUES
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001',
     'ad11ad11-ad11-ad11-ad11-ad11ad11ad11', 'closed',
     NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days', NOW(),
     NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days'),
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001',
     'e7b20001-0000-0000-0000-000000000001', 'closed',
     NOW() - INTERVAL '9 days', NOW() - INTERVAL '9 days', NOW(),
     NOW() - INTERVAL '9 days', NOW() - INTERVAL '9 days'),
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001',
     'e7b20002-0000-0000-0000-000000000002', 'open',
     NOW() - INTERVAL '8 days', NOW() - INTERVAL '8 days', NOW(),
     NOW() - INTERVAL '8 days', NOW() - INTERVAL '7 days'),
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001',
     'ad11ad11-ad11-ad11-ad11-ad11ad11ad12', 'open',
     NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days', NOW(),
     NOW() - INTERVAL '7 days', NOW() - INTERVAL '6 days'),
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001',
     'e7b30001-0000-0000-0000-000000000000', 'open',
     NOW() - INTERVAL '6 days', NOW() - INTERVAL '6 days', NOW(),
     NOW() - INTERVAL '6 days', NOW() - INTERVAL '5 days'),
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001',
     'e7b30002-0000-0000-0000-000000000000', 'open',
     NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days', NOW(),
     NOW() - INTERVAL '5 days', NOW() - INTERVAL '4 days'),
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001',
     'e7b30003-0000-0000-0000-000000000000', 'open',
     NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days', NOW(),
     NOW() - INTERVAL '4 days', NOW() - INTERVAL '3 days'),
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001',
     'e7b30004-0000-0000-0000-000000000000', 'open',
     NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days', NOW(),
     NOW() - INTERVAL '3 days', NOW() - INTERVAL '2 days'),
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001',
     'e7b30005-0000-0000-0000-000000000000', 'open',
     NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days', NOW(),
     NOW() - INTERVAL '2 days', NOW() - INTERVAL '1 days'),
    (gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001',
     'e7b30006-0000-0000-0000-000000000000', 'open',
     NOW() - INTERVAL '1 days', NOW() - INTERVAL '1 days', NOW(),
     NOW() - INTERVAL '1 days', NOW());

-- ============================================================
-- PHASE 4: Fashion Items + Brand Items + Digital Sample
-- ============================================================

-- Fashion Items (6 items, processing_version=1, embedding=NULL)
INSERT INTO fashion_items (id, category_id, image_url, image_public_id, color, color_hex,
                           color_hue, color_saturation, color_lightness,
                           style, material, pattern, fit, seasonality, description,
                           embedding, processing_retry_count, processing_version,
                           created_at, updated_at)
VALUES
    ('f1000001-0000-0000-0000-000000000001', '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a01',
     'https://placehold.co/400x500/FF6B6B/333?text=Ao+Thun', '', 'Đỏ', '#FF6B6B',
     0, 100, 50, 'Casual', 'Cotton', 'Trơn', 'Regular', 'Mùa hè',
     'Áo thun đỏ basic dễ phối đồ', NULL, 0, 1, NOW(), NOW()),

    ('f1000002-0000-0000-0000-000000000002', '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a02',
     'https://placehold.co/400x500/4A90D9/333?text=Quan+Jean', '', 'Xanh dương', '#4A90D9',
     210, 80, 50, 'Casual', 'Denim', 'Trơn', 'Regular', 'Bốn mùa',
     'Quần jeans xanh dương cổ điển', NULL, 0, 1, NOW(), NOW()),

    ('f1000003-0000-0000-0000-000000000003', '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a07',
     'https://placehold.co/400x500/333333/FFF?text=Ao+Khoac', '', 'Đen', '#333333',
     0, 0, 20, 'Formal', 'Wool', 'Trơn', 'Regular', 'Mùa đông',
     'Áo khoác len đen cao cấp', NULL, 0, 1, NOW(), NOW()),

    ('f1000004-0000-0000-0000-000000000004', '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a04',
     'https://placehold.co/400x500/FFFFFF/333?text=Giay+TT', '', 'Trắng', '#FFFFFF',
     0, 0, 100, 'Sporty', 'Canvas', 'Trơn', 'Regular', 'Mùa hè',
     'Giày sneaker trắng thể thao', NULL, 0, 1, NOW(), NOW()),

    ('f1000005-0000-0000-0000-000000000005', '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a06',
     'https://placehold.co/400x500/FFB6C1/333?text=Dam+Dep', '', 'Hồng', '#FFB6C1',
     350, 60, 70, 'Elegant', 'Lụa', 'Hoa', 'Regular', 'Mùa hè',
     'Đầm hồng thanh lịch dạo phố', NULL, 0, 1, NOW(), NOW()),

    ('f1000006-0000-0000-0000-000000000006', '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a09',
     'https://placehold.co/400x500/98D8C8/333?text=Chan+Vay', '', 'Xanh mint', '#98D8C8',
     160, 60, 70, 'Casual', 'Cotton', 'Trơn', 'Regular', 'Mùa hè',
     'Chân váy xanh mint trẻ trung', NULL, 0, 1, NOW(), NOW());

-- Brand Items (4 items, status=ACTIVE, gắn fashion_items)
INSERT INTO brand_items (id, brand_id, fashion_item_id, name, description, price, item_type, status, created_at, updated_at)
VALUES
    ('b1000001-0000-0000-0000-000000000001', 'e7b10001-0000-0000-0000-000000000001',
     'f1000001-0000-0000-0000-000000000001', 'Áo Thun ABC Đỏ', 'Áo thun đỏ signature của Thời Trang ABC', 299000, 'sample', 'active', NOW(), NOW()),
    ('b1000002-0000-0000-0000-000000000002', 'e7b10001-0000-0000-0000-000000000001',
     'f1000002-0000-0000-0000-000000000002', 'Quần Jean ABC Xanh', 'Quần jeans xanh cao cấp', 599000, 'sample', 'active', NOW(), NOW()),
    ('b1000003-0000-0000-0000-000000000003', 'e7b10001-0000-0000-0000-000000000001',
     'f1000003-0000-0000-0000-000000000003', 'Áo Khoác ABC Đen', 'Áo khoác len mùa đông', 899000, 'sample', 'active', NOW(), NOW()),
    ('b1000004-0000-0000-0000-000000000004', 'e7b10002-0000-0000-0000-000000000002',
     'f1000004-0000-0000-0000-000000000004', 'Sneaker XYZ Trắng', 'Giày sneaker trắng phong cách', 450000, 'sample', 'active', NOW(), NOW());

-- Digital Sample Responses (15 responses cho brand items của Brand A)
INSERT INTO digital_sample_responses (id, brand_item_id, user_id, rating, feedback_text, created_at)
SELECT
    gen_random_uuid(),
    b.id,
    'ad11ad11-ad11-ad11-ad11-ad11ad11ad12',
    (random() * 4 + 1)::int,
    (ARRAY['Chất liệu tốt, mặc thoải mái', 'Màu sắc đẹp như hình', 'Không giống như quảng cáo lắm', 'Rất hài lòng với sản phẩm', 'Sẽ mua thêm', 'Sản phẩm ổn, giá hợp lý', 'Mong muốn có thêm màu khác', 'Chất vải hơi mỏng', 'Phù hợp với thời tiết hiện tại', 'Đã mua và rất ưng'])[floor(random() * 10 + 1)],
    NOW() - (random() * 14 || ' days')::interval
FROM brand_items b
CROSS JOIN generate_series(1, 3)
WHERE b.brand_id = 'e7b10001-0000-0000-0000-000000000001'
LIMIT 15;

-- ============================================================
-- PHASE 5: AI Layer (Drill-down data)
-- ============================================================

-- AI Policy Grants (5 grants for the end user, referencing the premium-default policy + premium plan)
INSERT INTO ai_policy_grants (id, user_id, policy_id, plan_id, plan_code, tier_rank, policy_snapshot,
                              effective_from, effective_to, status, created_at, updated_at)
SELECT
    gen_random_uuid(), 'ad11ad11-ad11-ad11-ad11-ad11ad11ad12',
    'aa000000-0000-0000-0000-000000000002', 'cb591a26-9f4a-4e86-b489-8d195c80521e',
    'premium-monthly', 1, '{}'::jsonb,
    NOW() - (5 - n || ' days')::interval, NOW() + INTERVAL '30 days', 'active', NOW(), NOW()
FROM generate_series(0, 4) AS n;

-- AI Usage Period Ledgers (5 ledgers, reference real grant_id)
INSERT INTO ai_usage_period_ledgers (id, grant_id, user_id, period_index, period_start, period_end,
                                     paid_input_tokens, paid_output_tokens, free_input_tokens, free_output_tokens,
                                     actual_cost_micro_vnd, reserved_cost_micro_vnd, created_at, updated_at)
SELECT
    gen_random_uuid(), g.id, 'ad11ad11-ad11-ad11-ad11-ad11ad11ad12', n,
    NOW() - (5 - n || ' days')::interval, NOW() - (5 - n - 1 || ' days')::interval,
    (random() * 100000 + 10000)::int, (random() * 20000 + 5000)::int,
    (random() * 50000 + 5000)::int, (random() * 10000 + 1000)::int,
    (random() * 10000000000 + 500000000)::bigint, (random() * 1000000000 + 100000000)::bigint,
    NOW(), NOW()
FROM generate_series(0, 4) AS n
JOIN LATERAL (SELECT id FROM ai_policy_grants WHERE user_id = 'ad11ad11-ad11-ad11-ad11-ad11ad11ad12' ORDER BY created_at LIMIT 1 OFFSET n) g ON true;

-- AI Usage Events (50 events: 45 confirmed + 5 released)
DO $$
DECLARE
    ledgers UUID[] := ARRAY(SELECT id FROM ai_usage_period_ledgers);
    providers TEXT[] := ARRAY['openai', 'anthropic', 'google'];
    models TEXT[] := ARRAY['gpt-4', 'gpt-4o', 'claude-3', 'claude-3.5', 'gemini-2.0-flash'];
    v_status TEXT;
    provider TEXT;
    model TEXT;
    lid UUID;
BEGIN
    FOR i IN 1..50 LOOP
        IF i <= 45 THEN
            v_status := 'confirmed';
        ELSE
            v_status := 'released';
        END IF;
        provider := providers[floor(random() * array_length(providers, 1) + 1)];
        model := models[floor(random() * array_length(models, 1) + 1)];
        lid := ledgers[floor(random() * array_length(ledgers, 1) + 1)];

        INSERT INTO ai_usage_events (id, request_id, ledger_id, user_id, operation, logical_route, provider, model,
                                     input_usd_per_million, output_usd_per_million, usd_to_vnd,
                                     prompt_tokens, output_tokens, thinking_tokens,
                                     reserved_cost_micro_vnd, actual_cost_micro_vnd, estimated_max_cost_micro_vnd,
                                     status, finish_reason, error_code, sent_at, completed_at, created_at, updated_at)
        VALUES (
            gen_random_uuid(), gen_random_uuid(), lid, 'ad11ad11-ad11-ad11-ad11-ad11ad11ad12',
            'chat', 'paid', provider, model,
            2.50, 10.00, 27500,
            (random() * 2000 + 100)::int, (random() * 800 + 50)::int, 0,
            (random() * 100000 + 5000)::int, (random() * 100000 + 5000)::int, (random() * 200000 + 10000)::int,
            v_status,
            CASE WHEN v_status = 'confirmed' THEN 'stop' ELSE NULL END,
            CASE WHEN v_status = 'released' THEN 'rate_limit_exceeded' ELSE NULL END,
            NOW() - (random() * 5 || ' hours')::interval,
            CASE WHEN v_status = 'confirmed' THEN NOW() - (random() * 4.5 || ' hours')::interval ELSE NULL END,
            NOW() - (random() * 5 || ' hours')::interval, NOW()
        );
    END LOOP;
END $$;

-- ============================================================
-- PHASE 6: Snapshot Tables (API response ngay lập tức)
-- ============================================================

-- Admin Daily Snapshots (7 ngày)
INSERT INTO dashboard_admin_daily_snapshots (id, snapshot_date, total_users, new_users, active_users_dau,
    total_brands, pending_review_brands, ai_requests_count, ai_error_count, ai_paid_tokens,
    ai_actual_cost_micro_vnd, subscription_revenue_vnd, net_ai_margin_vnd,
    payos_transaction_count, payos_success_count, created_at, updated_at)
SELECT
    gen_random_uuid(), CURRENT_DATE - n,
    1500 + n * 5, 30 + n, 650 + n * 10,
    50 - n, 3 + (n % 2),     12000 + n * 100, 45 + n, 14000000::bigint + n * 100000,
    35000000000::bigint + n * 500000000::bigint, 120000000.00 + n * 5000000, 80000000.00 + n * 5000000,
    115 + n, 110 + n, NOW(), NOW()
FROM generate_series(0, 6) AS n;

-- Brand Daily Snapshots (7 ngày × 2 brands)
INSERT INTO dashboard_brand_daily_snapshots (id, brand_id, snapshot_date, total_members, points_issued, points_redeemed,
    points_expiring_30d, points_liability_value_vnd, active_vouchers, vouchers_redeemed,
    digital_sample_votes_count, digital_sample_avg_rating, open_cs_tickets, cs_first_response_avg_seconds,
    created_at, updated_at)
SELECT
    gen_random_uuid(),
    CASE WHEN n % 2 = 0 THEN 'e7b10001-0000-0000-0000-000000000001'::uuid ELSE 'e7b10002-0000-0000-0000-000000000002'::uuid END,
    CURRENT_DATE - (n / 2),
    500 + n * 3, 14000 + n * 150, 4000 + n * 50,
    5500 + n * 20, 5500000.00 + n * 20000, 18 - (n % 2), 6 + n,
    220 + n * 2, 4.2 + (n % 3) * 0.1, 10 + (n % 3), 140 + n * 2,
    NOW(), NOW()
FROM generate_series(0, 13) AS n;

-- Brand Item Daily Snapshots (7 ngày × 2 items, cho drill-down)
INSERT INTO dashboard_brand_item_daily_snapshots (id, brand_id, item_id, snapshot_date, votes_count, avg_rating,
    feedback_sentiment, top_paired_categories, created_at, updated_at)
SELECT
    gen_random_uuid(), 'e7b10001-0000-0000-0000-000000000001'::uuid,
    CASE WHEN n % 2 = 0 THEN 'b1000001-0000-0000-0000-000000000001'::uuid ELSE 'b1000002-0000-0000-0000-000000000002'::uuid END,
    CURRENT_DATE - (n / 2),
    30 + n * 2, 4.0 + (n % 3) * 0.1,
    'positive', '[{"categoryName":"Quần Jean","pairCount":25},{"categoryName":"Giày Sneaker","pairCount":18}]'::jsonb,
    NOW(), NOW()
FROM generate_series(0, 13) AS n;

-- User Wardrobe Summary (cho user@closy.com)
INSERT INTO dashboard_user_wardrobe_summaries (user_id, total_items, total_wardrobe_value_vnd, average_cost_per_wear_vnd,
    underutilized_items_count, underutilized_items, ai_quota_remaining, expiring_vouchers_count, updated_at)
VALUES (
    'ad11ad11-ad11-ad11-ad11-ad11ad11ad12',
    42, 12500000.00, 297619.05,
    3, '[{"itemId":"11111111-1111-1111-1111-111111111111","name":"Áo khoác Blazer đen","categoryName":"Áo Khoác","purchasePriceVnd":1200000,"lastWornDaysAgo":120,"imageUrl":"https://placehold.co/400x500/333333/FFF?text=Blazer"},{"itemId":"22222222-2222-2222-2222-222222222222","name":"Đầm dạ hội đỏ","categoryName":"Đầm","purchasePriceVnd":2500000,"lastWornDaysAgo":90,"imageUrl":"https://placehold.co/400x500/FF0000/FFF?text=Dam+Do"},{"itemId":"33333333-3333-3333-3333-333333333333","name":"Giày cao gót đen","categoryName":"Giày","purchasePriceVnd":1800000,"lastWornDaysAgo":60,"imageUrl":"https://placehold.co/400x500/333333/FFF?text=Giay+Cao+Got"}]'::jsonb,
    8, 3, NOW()
)
ON CONFLICT (user_id) DO UPDATE SET
    total_items = EXCLUDED.total_items,
    total_wardrobe_value_vnd = EXCLUDED.total_wardrobe_value_vnd,
    underutilized_items_count = EXCLUDED.underutilized_items_count,
    underutilized_items = EXCLUDED.underutilized_items,
    ai_quota_remaining = EXCLUDED.ai_quota_remaining,
    expiring_vouchers_count = EXCLUDED.expiring_vouchers_count,
    updated_at = NOW();

COMMIT;
