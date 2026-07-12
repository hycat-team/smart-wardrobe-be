-- Demo seed for quick frontend testing of the brand portal flow.
-- All demo accounts use password: 123456
--
-- Suggested login usernames:
--   seed_admin
--   seed_brand_owner
--   seed_brand_staff
--   seed_loyalty_customer
--   seed_loyalty_gold_customer
--   seed_loyalty_vip_customer

BEGIN;

-- Password hash for "123456", reused from the project's default seed.
-- This seed is idempotent and updates known demo rows when re-run.

INSERT INTO users (
    id,
    username,
    email,
    password_hash,
    first_name,
    last_name,
    role_slug,
    status,
    is_deleted,
    created_at,
    updated_at
)
VALUES
    (
        '10000000-0000-0000-0000-000000000001',
        'seed_admin',
        'seed.admin@closy.local',
        '$2a$11$kXWLREY8wu6wEQlONWcLveV2jeE/Tx9MS4vOlQqXmcQ9VASP0NMhu',
        'Seed',
        'Admin',
        'admin',
        0,
        false,
        now(),
        now()
    ),
    (
        '10000000-0000-0000-0000-000000000011',
        'seed_brand_owner',
        'seed.brand.owner@closy.local',
        '$2a$11$kXWLREY8wu6wEQlONWcLveV2jeE/Tx9MS4vOlQqXmcQ9VASP0NMhu',
        'Seed',
        'Brand Owner',
        'user',
        0,
        false,
        now(),
        now()
    ),
    (
        '10000000-0000-0000-0000-000000000012',
        'seed_brand_staff',
        'seed.brand.staff@closy.local',
        '$2a$11$kXWLREY8wu6wEQlONWcLveV2jeE/Tx9MS4vOlQqXmcQ9VASP0NMhu',
        'Seed',
        'Brand Staff',
        'user',
        0,
        false,
        now(),
        now()
    ),
    (
        '10000000-0000-0000-0000-000000000021',
        'seed_loyalty_customer',
        'seed.loyalty.customer@closy.local',
        '$2a$11$kXWLREY8wu6wEQlONWcLveV2jeE/Tx9MS4vOlQqXmcQ9VASP0NMhu',
        'Seed',
        'Loyalty Customer',
        'user',
        0,
        false,
        now(),
        now()
    ),
    (
        '10000000-0000-0000-0000-000000000022',
        'seed_loyalty_vip_customer',
        'seed.loyalty.vip@closy.local',
        '$2a$11$kXWLREY8wu6wEQlONWcLveV2jeE/Tx9MS4vOlQqXmcQ9VASP0NMhu',
        'Seed',
        'VIP Customer',
        'user',
        0,
        false,
        now(),
        now()
    ),
    (
        '10000000-0000-0000-0000-000000000023',
        'seed_loyalty_gold_customer',
        'seed.loyalty.gold@closy.local',
        '$2a$11$kXWLREY8wu6wEQlONWcLveV2jeE/Tx9MS4vOlQqXmcQ9VASP0NMhu',
        'Seed',
        'Gold Customer',
        'user',
        0,
        false,
        now(),
        now()
    )
ON CONFLICT (id) DO UPDATE SET
    username = EXCLUDED.username,
    email = EXCLUDED.email,
    password_hash = EXCLUDED.password_hash,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    role_slug = EXCLUDED.role_slug,
    status = EXCLUDED.status,
    is_deleted = EXCLUDED.is_deleted,
    updated_at = now();

INSERT INTO brands (
    id,
    slug,
    name,
    description,
    logo_url,
    logo_public_id,
    status,
    created_by_user_id,
    approved_by_user_id,
    approved_at,
    created_at,
    updated_at
)
VALUES (
    '20000000-0000-0000-0000-000000000001',
    'seed-active-brand',
    'Seed Active Brand',
    'Ready-to-use active brand for frontend brand portal testing.',
    'https://placehold.co/256x256/111827/ffffff?text=SEED',
    'seed/brands/active-brand-logo',
    'active',
    '10000000-0000-0000-0000-000000000011',
    '10000000-0000-0000-0000-000000000001',
    now(),
    now(),
    now()
)
ON CONFLICT (id) DO UPDATE SET
    slug = EXCLUDED.slug,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    logo_url = EXCLUDED.logo_url,
    logo_public_id = EXCLUDED.logo_public_id,
    status = EXCLUDED.status,
    created_by_user_id = EXCLUDED.created_by_user_id,
    approved_by_user_id = EXCLUDED.approved_by_user_id,
    approved_at = EXCLUDED.approved_at,
    updated_at = now();

INSERT INTO brand_members (
    id,
    brand_id,
    user_id,
    role,
    status,
    created_at,
    updated_at
)
VALUES
    (
        '30000000-0000-0000-0000-000000000011',
        '20000000-0000-0000-0000-000000000001',
        '10000000-0000-0000-0000-000000000011',
        'owner',
        'active',
        now(),
        now()
    ),
    (
        '30000000-0000-0000-0000-000000000012',
        '20000000-0000-0000-0000-000000000001',
        '10000000-0000-0000-0000-000000000012',
        'staff',
        'active',
        now(),
        now()
    )
ON CONFLICT (id) DO UPDATE SET
    brand_id = EXCLUDED.brand_id,
    user_id = EXCLUDED.user_id,
    role = EXCLUDED.role,
    status = EXCLUDED.status,
    updated_at = now();

INSERT INTO brand_customers (
    id,
    brand_id,
    user_id,
    customer_name,
    phone_e164,
    phone_hash,
    external_customer_code,
    joined_source,
    status,
    joined_at,
    claimed_at,
    created_by_member_id,
    created_at,
    updated_at
)
VALUES
    (
        '40000000-0000-0000-0000-000000000021',
        '20000000-0000-0000-0000-000000000001',
        '10000000-0000-0000-0000-000000000021',
        'Seed Loyalty Customer',
        '+84900000021',
        'seed-phone-hash-loyalty-customer',
        'SEED-CUSTOMER-001',
        'self_join',
        'active',
        now() - interval '20 days',
        now() - interval '20 days',
        '30000000-0000-0000-0000-000000000011',
        now(),
        now()
    ),
    (
        '40000000-0000-0000-0000-000000000022',
        '20000000-0000-0000-0000-000000000001',
        '10000000-0000-0000-0000-000000000022',
        'Seed VIP Customer',
        '+84900000022',
        'seed-phone-hash-vip-customer',
        'SEED-CUSTOMER-002',
        'offline_purchase',
        'active',
        now() - interval '60 days',
        now() - interval '55 days',
        '30000000-0000-0000-0000-000000000012',
        now(),
        now()
    ),
    (
        '40000000-0000-0000-0000-000000000023',
        '20000000-0000-0000-0000-000000000001',
        '10000000-0000-0000-0000-000000000023',
        'Seed Gold Customer',
        '+84900000023',
        'seed-phone-hash-gold-customer',
        'SEED-CUSTOMER-003',
        'offline_purchase',
        'active',
        now() - interval '5 days',
        now() - interval '4 days',
        '30000000-0000-0000-0000-000000000012',
        now(),
        now()
    )
ON CONFLICT (id) DO UPDATE SET
    brand_id = EXCLUDED.brand_id,
    user_id = EXCLUDED.user_id,
    customer_name = EXCLUDED.customer_name,
    phone_e164 = EXCLUDED.phone_e164,
    phone_hash = EXCLUDED.phone_hash,
    external_customer_code = EXCLUDED.external_customer_code,
    joined_source = EXCLUDED.joined_source,
    status = EXCLUDED.status,
    joined_at = EXCLUDED.joined_at,
    claimed_at = EXCLUDED.claimed_at,
    created_by_member_id = EXCLUDED.created_by_member_id,
    updated_at = now();

INSERT INTO loyalty_programs (
    id,
    brand_id,
    name,
    amount_per_point,
    point_expiry_days,
    rounding_mode,
    is_active,
    created_at,
    updated_at
)
VALUES (
    '50000000-0000-0000-0000-000000000001',
    '20000000-0000-0000-0000-000000000001',
    'Seed Brand Loyalty',
    10000.00,
    365,
    'floor',
    true,
    now(),
    now()
)
ON CONFLICT (id) DO UPDATE SET
    brand_id = EXCLUDED.brand_id,
    name = EXCLUDED.name,
    amount_per_point = EXCLUDED.amount_per_point,
    point_expiry_days = EXCLUDED.point_expiry_days,
    rounding_mode = EXCLUDED.rounding_mode,
    is_active = EXCLUDED.is_active,
    updated_at = now();

INSERT INTO loyalty_tiers (
    id,
    brand_id,
    name,
    rank,
    min_total_spend,
    description,
    created_at,
    updated_at
)
VALUES
    (
        '51000000-0000-0000-0000-000000000001',
        '20000000-0000-0000-0000-000000000001',
        'Member',
        1,
        0.00,
        'Default tier for newly joined customers.',
        now(),
        now()
    ),
    (
        '51000000-0000-0000-0000-000000000002',
        '20000000-0000-0000-0000-000000000001',
        'Gold',
        2,
        2000000.00,
        'Gold tier for customers with at least 2,000,000 VND total spend.',
        now(),
        now()
    ),
    (
        '51000000-0000-0000-0000-000000000003',
        '20000000-0000-0000-0000-000000000001',
        'VIP',
        3,
        5000000.00,
        'VIP tier for customers with at least 5,000,000 VND total spend.',
        now(),
        now()
    )
ON CONFLICT (id) DO UPDATE SET
    brand_id = EXCLUDED.brand_id,
    name = EXCLUDED.name,
    rank = EXCLUDED.rank,
    min_total_spend = EXCLUDED.min_total_spend,
    description = EXCLUDED.description,
    updated_at = now();

INSERT INTO loyalty_accounts (
    id,
    brand_id,
    brand_customer_id,
    user_id,
    current_points,
    lifetime_points,
    total_spend,
    current_tier_id,
    created_at,
    updated_at
)
VALUES
    (
        '52000000-0000-0000-0000-000000000021',
        '20000000-0000-0000-0000-000000000001',
        '40000000-0000-0000-0000-000000000021',
        '10000000-0000-0000-0000-000000000021',
        120,
        120,
        1200000.00,
        '51000000-0000-0000-0000-000000000001',
        now(),
        now()
    ),
    (
        '52000000-0000-0000-0000-000000000022',
        '20000000-0000-0000-0000-000000000001',
        '40000000-0000-0000-0000-000000000022',
        '10000000-0000-0000-0000-000000000022',
        750,
        750,
        7500000.00,
        '51000000-0000-0000-0000-000000000003',
        now(),
        now()
    ),
    (
        '52000000-0000-0000-0000-000000000023',
        '20000000-0000-0000-0000-000000000001',
        '40000000-0000-0000-0000-000000000023',
        '10000000-0000-0000-0000-000000000023',
        250,
        250,
        2500000.00,
        '51000000-0000-0000-0000-000000000002',
        now(),
        now()
    )
ON CONFLICT (id) DO UPDATE SET
    brand_id = EXCLUDED.brand_id,
    brand_customer_id = EXCLUDED.brand_customer_id,
    user_id = EXCLUDED.user_id,
    current_points = EXCLUDED.current_points,
    lifetime_points = EXCLUDED.lifetime_points,
    total_spend = EXCLUDED.total_spend,
    current_tier_id = EXCLUDED.current_tier_id,
    updated_at = now();

INSERT INTO loyalty_point_transactions (
    id,
    loyalty_account_id,
    brand_id,
    brand_customer_id,
    user_id,
    points_delta,
    balance_after,
    transaction_type,
    reason,
    spend_amount,
    reference_type,
    reference_id,
    expires_at,
    idempotency_key,
    created_by_user_id,
    created_at
)
VALUES
    (
        '53000000-0000-0000-0000-000000000021',
        '52000000-0000-0000-0000-000000000021',
        '20000000-0000-0000-0000-000000000001',
        '40000000-0000-0000-0000-000000000021',
        '10000000-0000-0000-0000-000000000021',
        120,
        120,
        'earn',
        'Initial seed purchase for standard customer.',
        1200000.00,
        'seed_demo',
        NULL,
        now() + interval '365 days',
        'seed-brand-standard-customer-earn-120',
        '10000000-0000-0000-0000-000000000012',
        now() - interval '20 days'
    ),
    (
        '53000000-0000-0000-0000-000000000022',
        '52000000-0000-0000-0000-000000000022',
        '20000000-0000-0000-0000-000000000001',
        '40000000-0000-0000-0000-000000000022',
        '10000000-0000-0000-0000-000000000022',
        750,
        750,
        'earn',
        'Initial seed purchase for VIP customer.',
        7500000.00,
        'seed_demo',
        NULL,
        now() + interval '365 days',
        'seed-brand-vip-customer-earn-750',
        '10000000-0000-0000-0000-000000000012',
        now() - interval '55 days'
    ),
    (
        '53000000-0000-0000-0000-000000000023',
        '52000000-0000-0000-0000-000000000023',
        '20000000-0000-0000-0000-000000000001',
        '40000000-0000-0000-0000-000000000023',
        '10000000-0000-0000-0000-000000000023',
        250,
        250,
        'earn',
        'Initial seed offline purchase for Gold tier.',
        2500000.00,
        'seed_demo',
        NULL,
        now() + interval '365 days',
        'seed-brand-offline-customer-earn-250',
        '10000000-0000-0000-0000-000000000012',
        now() - interval '5 days'
    )
ON CONFLICT (id) DO UPDATE SET
    loyalty_account_id = EXCLUDED.loyalty_account_id,
    brand_id = EXCLUDED.brand_id,
    brand_customer_id = EXCLUDED.brand_customer_id,
    user_id = EXCLUDED.user_id,
    points_delta = EXCLUDED.points_delta,
    balance_after = EXCLUDED.balance_after,
    transaction_type = EXCLUDED.transaction_type,
    reason = EXCLUDED.reason,
    spend_amount = EXCLUDED.spend_amount,
    reference_type = EXCLUDED.reference_type,
    reference_id = EXCLUDED.reference_id,
    expires_at = EXCLUDED.expires_at,
    idempotency_key = EXCLUDED.idempotency_key,
    created_by_user_id = EXCLUDED.created_by_user_id;

INSERT INTO loyalty_point_lots (
    id,
    loyalty_account_id,
    brand_id,
    brand_customer_id,
    user_id,
    earn_transaction_id,
    earned_points,
    remaining_points,
    expires_at,
    status,
    created_at,
    updated_at
)
VALUES
    (
        '54000000-0000-0000-0000-000000000021',
        '52000000-0000-0000-0000-000000000021',
        '20000000-0000-0000-0000-000000000001',
        '40000000-0000-0000-0000-000000000021',
        '10000000-0000-0000-0000-000000000021',
        '53000000-0000-0000-0000-000000000021',
        120,
        120,
        now() + interval '365 days',
        'active',
        now(),
        now()
    ),
    (
        '54000000-0000-0000-0000-000000000022',
        '52000000-0000-0000-0000-000000000022',
        '20000000-0000-0000-0000-000000000001',
        '40000000-0000-0000-0000-000000000022',
        '10000000-0000-0000-0000-000000000022',
        '53000000-0000-0000-0000-000000000022',
        750,
        750,
        now() + interval '365 days',
        'active',
        now(),
        now()
    ),
    (
        '54000000-0000-0000-0000-000000000023',
        '52000000-0000-0000-0000-000000000023',
        '20000000-0000-0000-0000-000000000001',
        '40000000-0000-0000-0000-000000000023',
        '10000000-0000-0000-0000-000000000023',
        '53000000-0000-0000-0000-000000000023',
        250,
        250,
        now() + interval '365 days',
        'active',
        now(),
        now()
    )
ON CONFLICT (id) DO UPDATE SET
    loyalty_account_id = EXCLUDED.loyalty_account_id,
    brand_id = EXCLUDED.brand_id,
    brand_customer_id = EXCLUDED.brand_customer_id,
    user_id = EXCLUDED.user_id,
    earn_transaction_id = EXCLUDED.earn_transaction_id,
    earned_points = EXCLUDED.earned_points,
    remaining_points = EXCLUDED.remaining_points,
    expires_at = EXCLUDED.expires_at,
    status = EXCLUDED.status,
    updated_at = now();

INSERT INTO brand_benefits (
    id,
    brand_id,
    name,
    description,
    benefit_type,
    unlock_type,
    required_points,
    required_tier_id,
    feature_code,
    feature_config,
    status,
    created_at,
    updated_at
)
VALUES
    (
        '60000000-0000-0000-0000-000000000001',
        '20000000-0000-0000-0000-000000000001',
        'VIP sample mix access',
        'Unlocks digital sample outfit mixing for VIP customers.',
        'feature_access',
        'tier_privilege',
        NULL,
        '51000000-0000-0000-0000-000000000003',
        'sample_mix_access',
        '{"source":"seed_demo"}',
        'active',
        now(),
        now()
    ),
    (
        '60000000-0000-0000-0000-000000000002',
        '20000000-0000-0000-0000-000000000001',
        '100 point voucher',
        'Point redemption benefit for quick frontend redeem testing.',
        'voucher',
        'point_redemption',
        100,
        NULL,
        NULL,
        '{"voucher_code":"SEED100"}',
        'active',
        now(),
        now()
    )
ON CONFLICT (id) DO UPDATE SET
    brand_id = EXCLUDED.brand_id,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    benefit_type = EXCLUDED.benefit_type,
    unlock_type = EXCLUDED.unlock_type,
    required_points = EXCLUDED.required_points,
    required_tier_id = EXCLUDED.required_tier_id,
    feature_code = EXCLUDED.feature_code,
    feature_config = EXCLUDED.feature_config,
    status = EXCLUDED.status,
    updated_at = now();

INSERT INTO fashion_items (
    id,
    category_id,
    image_url,
    image_public_id,
    color,
    color_hex,
    style,
    material,
    pattern,
    fit,
    seasonality,
    description,
    created_at,
    updated_at
)
VALUES
    (
        '70000000-0000-0000-0000-000000000001',
        '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a01',
        'https://placehold.co/800x1000/e11d48/ffffff?text=Seed+Tee',
        'seed/fashion/brand-red-tee',
        'Red',
        '#e11d48',
        'Casual',
        'Cotton',
        'Solid',
        'Regular',
        'All season',
        'Seed brand red tee for product listing tests.',
        now(),
        now()
    ),
    (
        '70000000-0000-0000-0000-000000000002',
        '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a07',
        'https://placehold.co/800x1000/0f766e/ffffff?text=Seed+Sample',
        'seed/fashion/brand-green-jacket-sample',
        'Green',
        '#0f766e',
        'Streetwear',
        'Twill',
        'Solid',
        'Oversized',
        'Autumn',
        'Seed digital sample jacket for feedback tests.',
        now(),
        now()
    )
ON CONFLICT (id) DO UPDATE SET
    category_id = EXCLUDED.category_id,
    image_url = EXCLUDED.image_url,
    image_public_id = EXCLUDED.image_public_id,
    color = EXCLUDED.color,
    color_hex = EXCLUDED.color_hex,
    style = EXCLUDED.style,
    material = EXCLUDED.material,
    pattern = EXCLUDED.pattern,
    fit = EXCLUDED.fit,
    seasonality = EXCLUDED.seasonality,
    description = EXCLUDED.description,
    updated_at = now();

INSERT INTO brand_items (
    id,
    brand_id,
    fashion_item_id,
    product_code,
    name,
    description,
    price,
    item_type,
    status,
    created_at,
    updated_at
)
VALUES
    (
        '71000000-0000-0000-0000-000000000001',
        '20000000-0000-0000-0000-000000000001',
        '70000000-0000-0000-0000-000000000001',
        'SEED-TEE-RED',
        'Seed Red Tee',
        'Active seed product for brand catalog testing.',
        249000.00,
        'product',
        'active',
        now(),
        now()
    ),
    (
        '71000000-0000-0000-0000-000000000002',
        '20000000-0000-0000-0000-000000000001',
        '70000000-0000-0000-0000-000000000002',
        'SEED-SAMPLE-JACKET',
        'Seed Digital Sample Jacket',
        'Active seed sample for digital sample feedback testing.',
        NULL,
        'sample',
        'active',
        now(),
        now()
    )
ON CONFLICT (id) DO UPDATE SET
    brand_id = EXCLUDED.brand_id,
    fashion_item_id = EXCLUDED.fashion_item_id,
    product_code = EXCLUDED.product_code,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    price = EXCLUDED.price,
    item_type = EXCLUDED.item_type,
    status = EXCLUDED.status,
    updated_at = now();

INSERT INTO brand_conversations (
    id,
    brand_id,
    user_id,
    status,
    last_message_at,
    user_last_read_at,
    staff_last_read_at,
    created_at,
    updated_at
)
VALUES (
    '80000000-0000-0000-0000-000000000001',
    '20000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000022',
    'open',
    now() - interval '10 minutes',
    now() - interval '30 minutes',
    now() - interval '10 minutes',
    now() - interval '1 hour',
    now()
)
ON CONFLICT (id) DO UPDATE SET
    brand_id = EXCLUDED.brand_id,
    user_id = EXCLUDED.user_id,
    status = EXCLUDED.status,
    last_message_at = EXCLUDED.last_message_at,
    user_last_read_at = EXCLUDED.user_last_read_at,
    staff_last_read_at = EXCLUDED.staff_last_read_at,
    updated_at = now();

INSERT INTO brand_conversation_messages (
    id,
    conversation_id,
    sender_user_id,
    sender_role,
    message,
    created_at
)
VALUES
    (
        '81000000-0000-0000-0000-000000000001',
        '80000000-0000-0000-0000-000000000001',
        '10000000-0000-0000-0000-000000000022',
        'customer',
        'Can you recommend an outfit with the new sample jacket?',
        now() - interval '30 minutes'
    ),
    (
        '81000000-0000-0000-0000-000000000002',
        '80000000-0000-0000-0000-000000000001',
        '10000000-0000-0000-0000-000000000012',
        'brand_staff',
        'Yes, the sample jacket works well with neutral trousers and white sneakers.',
        now() - interval '10 minutes'
    )
ON CONFLICT (id) DO UPDATE SET
    conversation_id = EXCLUDED.conversation_id,
    sender_user_id = EXCLUDED.sender_user_id,
    sender_role = EXCLUDED.sender_role,
    message = EXCLUDED.message,
    created_at = EXCLUDED.created_at;

COMMIT;
