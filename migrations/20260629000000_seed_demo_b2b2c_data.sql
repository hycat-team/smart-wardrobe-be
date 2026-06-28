-- +goose Up

-- 1. Users
INSERT INTO users (id, username, email, password_hash, first_name, last_name, role_slug, status, created_at, updated_at)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'brandowner', 'owner@brand.com', '$2a$11$kXWLREY8wu6wEQlONWcLveV2jeE/Tx9MS4vOlQqXmcQ9VASP0NMhu', 'Brand', 'Owner', 'user', 0, now(), now()),
    ('11111111-1111-1111-1111-111111111112', 'brandmanager', 'manager@brand.com', '$2a$11$kXWLREY8wu6wEQlONWcLveV2jeE/Tx9MS4vOlQqXmcQ9VASP0NMhu', 'Brand', 'Manager', 'user', 0, now(), now()),
    ('22222222-2222-2222-2222-222222222221', 'bronzeuser', 'bronze@closy.com', '$2a$11$kXWLREY8wu6wEQlONWcLveV2jeE/Tx9MS4vOlQqXmcQ9VASP0NMhu', 'Bronze', 'User', 'user', 0, now(), now()),
    ('22222222-2222-2222-2222-222222222222', 'golduser', 'gold@closy.com', '$2a$11$kXWLREY8wu6wEQlONWcLveV2jeE/Tx9MS4vOlQqXmcQ9VASP0NMhu', 'Gold', 'User', 'user', 0, now(), now())
ON CONFLICT (id) DO NOTHING;

-- 2. Brands
INSERT INTO brands (id, slug, name, description, logo_url, status, created_by_user_id, approved_by_user_id, approved_at, created_at, updated_at)
VALUES
    ('33333333-3333-3333-3333-333333333333', 'closy-brand', 'Closy Brand', 'Closy Fashion Brand Demo B2B2C', 'https://logo.com/closy', 'ACTIVE', '11111111-1111-1111-1111-111111111111', 'ad11ad11-ad11-ad11-ad11-ad11ad11ad11', now(), now(), now())
ON CONFLICT (id) DO NOTHING;

-- 3. Brand Members
INSERT INTO brand_members (id, brand_id, user_id, role, status, created_at, updated_at)
VALUES
    ('44444444-4444-4444-4444-444444444441', '33333333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', 'OWNER', 'ACTIVE', now(), now()),
    ('44444444-4444-4444-4444-444444444442', '33333333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111112', 'MANAGER', 'ACTIVE', now(), now())
ON CONFLICT (id) DO NOTHING;

-- 4. Brand Customers
INSERT INTO brand_customers (id, brand_id, user_id, customer_name, phone_e164, phone_hash, joined_source, status, joined_at, created_at, updated_at)
VALUES
    ('55555555-5555-5555-5555-555555555551', '33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222221', 'Bronze User', '+84911111111', '89c96804cf9c870c3523f1de76654f74dc7b552e311e99abbc25b4e7e4038923', 'SELF_JOIN', 'ACTIVE', now(), now(), now()),
    ('55555555-5555-5555-5555-555555555552', '33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222222', 'Gold User', '+84922222222', '279b348bff07a69e8b7fec1fae9c2dfe33fa48aa382c74406d0310782cea0457', 'SELF_JOIN', 'ACTIVE', now(), now(), now()),
    ('55555555-5555-5555-5555-555555555553', '33333333-3333-3333-3333-333333333333', NULL, 'Offline Client', '+84999999999', 'c0670defeccfaa974f32e596cfa63c8316731ee0e198b64a1b4e52063195ff13', 'OFFLINE_PURCHASE', 'ACTIVE', now(), now(), now())
ON CONFLICT (id) DO NOTHING;

-- 5. Loyalty Program & Tiers
INSERT INTO loyalty_programs (id, brand_id, name, amount_per_point, point_expiry_days, rounding_mode, is_active, created_at, updated_at)
VALUES
    ('66666666-6666-6666-6666-666666666666', '33333333-3333-3333-3333-333333333333', 'Closy Loyalty Program', 10000.00, NULL, 'FLOOR', true, now(), now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO loyalty_tiers (id, brand_id, name, rank, min_total_spend, description, created_at, updated_at)
VALUES
    ('77777777-7777-7777-7777-777777777771', '33333333-3333-3333-3333-333333333333', 'Bronze', 1, 0.00, 'Hạng đồng mặc định', now(), now()),
    ('77777777-7777-7777-7777-777777777772', '33333333-3333-3333-3333-333333333333', 'Silver', 2, 1000000.00, 'Hạng bạc tích lũy từ 1 triệu đồng', now(), now()),
    ('77777777-7777-7777-7777-777777777773', '33333333-3333-3333-3333-333333333333', 'Gold', 3, 5000000.00, 'Hạng vàng tích lũy từ 5 triệu đồng', now(), now())
ON CONFLICT (id) DO NOTHING;

-- 6. Loyalty Accounts
INSERT INTO loyalty_accounts (id, brand_id, brand_customer_id, user_id, current_points, lifetime_points, total_spend, current_tier_id, created_at, updated_at)
VALUES
    ('88888888-8888-8888-8888-888888888881', '33333333-3333-3333-3333-333333333333', '55555555-5555-5555-5555-555555555551', '22222222-2222-2222-2222-222222222221', 0, 0, 0.00, '77777777-7777-7777-7777-777777777771', now(), now()),
    ('88888888-8888-8888-8888-888888888882', '33333333-3333-3333-3333-333333333333', '55555555-5555-5555-5555-555555555552', '22222222-2222-2222-2222-222222222222', 500, 500, 5000000.00, '77777777-7777-7777-7777-777777777773', now(), now()),
    ('88888888-8888-8888-8888-888888888883', '33333333-3333-3333-3333-333333333333', '55555555-5555-5555-5555-555555555553', NULL, 0, 0, 0.00, '77777777-7777-7777-7777-777777777771', now(), now())
ON CONFLICT (id) DO NOTHING;

-- 7. Benefits
INSERT INTO brand_benefits (id, brand_id, name, description, benefit_type, unlock_type, required_points, required_tier_id, feature_code, feature_config, status, created_at, updated_at)
VALUES
    ('99999999-9999-9999-9999-999999999991', '33333333-3333-3333-3333-333333333333', 'Phối mẫu thử Gold', 'Quyền lợi phối đồ mẫu thử của brand chỉ dành riêng cho hạng Gold', 'FEATURE_ACCESS', 'TIER_PRIVILEGE', NULL, '77777777-7777-7777-7777-777777777773', 'SAMPLE_MIX_ACCESS', '{}', 'ACTIVE', now(), now())
ON CONFLICT (id) DO NOTHING;

-- 8. Fashion Items & Brand Items
INSERT INTO fashion_items (id, category_id, image_url, image_public_id, color, style, material, pattern, fit, seasonality, description, status, created_at, updated_at)
VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a01', 'https://res.cloudinary.com/demo/image/upload/v1/red-shirt.jpg', 'red-shirt', 'Đỏ', 'Casual', 'Cotton', 'Trơn', 'Regular', 'Bốn mùa', 'Áo thun đỏ Closy chính hãng mịn mát', 1, now(), now()),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaab', '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a01', 'https://res.cloudinary.com/demo/image/upload/v1/yellow-shirt.jpg', 'yellow-shirt', 'Vàng', 'Casual', 'Cotton', 'Trơn', 'Regular', 'Mùa hè', 'Mẫu thử kỹ thuật số Áo thun vàng Closy', 1, now(), now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO brand_items (id, brand_id, fashion_item_id, product_code, name, description, price, item_type, status, created_at, updated_at)
VALUES
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb1', '33333333-3333-3333-3333-333333333333', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'CLSY-RED-TEE', 'Áo thun đỏ Closy', 'Chất liệu thun cotton thoáng mát co giãn tốt', 200000.00, 'PRODUCT', 'ACTIVE', now(), now()),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb2', '33333333-3333-3333-3333-333333333333', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaab', 'CLSY-YELLOW-SAMPLE', 'Mẫu thử Áo thun vàng Closy', 'Thiết kế mẫu thử vàng nổi bật mùa hè', 0.00, 'SAMPLE', 'ACTIVE', now(), now())
ON CONFLICT (id) DO NOTHING;

-- 9. Wardrobe Items (Bao gồm quần jeans xanh trơn có sẵn để phối)
INSERT INTO wardrobe_items (id, user_id, category_id, fashion_item_id, purchase_price, status, item_type, created_at, updated_at)
VALUES
    ('cccccccc-cccc-cccc-cccc-ccccccccccc1', '22222222-2222-2222-2222-222222222221', '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a02', 'ca7ca7ca-ca7c-ca7c-ca7c-ca7ca7ca7c02', 450000.00, 0, 0, now(), now()),
    ('cccccccc-cccc-cccc-cccc-ccccccccccc2', '22222222-2222-2222-2222-222222222222', '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a02', 'ca7ca7ca-ca7c-ca7c-ca7c-ca7ca7ca7c02', 450000.00, 0, 0, now(), now())
ON CONFLICT (id) DO NOTHING;

-- 10. Chat
INSERT INTO brand_conversations (id, brand_id, user_id, status, last_message_at, created_at, updated_at)
VALUES
    ('dddddddd-dddd-dddd-dddd-dddddddddddd', '33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222222', 'OPEN', now(), now(), now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO brand_conversation_messages (id, conversation_id, sender_user_id, sender_role, message, created_at)
VALUES
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee1', 'dddddddd-dddd-dddd-dddd-dddddddddddd', '22222222-2222-2222-2222-222222222222', 'USER', 'Tôi muốn tư vấn phối đồ với sản phẩm của brand', now() - interval '1 hour'),
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee2', 'dddddddd-dddd-dddd-dddd-dddddddddddd', '11111111-1111-1111-1111-111111111112', 'STAFF', 'Xin chào, chúng tôi có các mẫu thử mới rất phù hợp với bạn.', now() - interval '30 minutes')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM brand_conversation_messages WHERE id IN ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee1', 'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee2');
DELETE FROM brand_conversations WHERE id = 'dddddddd-dddd-dddd-dddd-dddddddddddd';
DELETE FROM wardrobe_items WHERE id IN ('cccccccc-cccc-cccc-cccc-ccccccccccc1', 'cccccccc-cccc-cccc-cccc-ccccccccccc2');
DELETE FROM brand_items WHERE id IN ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb1', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb2');
DELETE FROM fashion_items WHERE id IN ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaab');
DELETE FROM brand_benefits WHERE id = '99999999-9999-9999-9999-999999999991';
DELETE FROM loyalty_accounts WHERE id IN ('88888888-8888-8888-8888-888888888881', '88888888-8888-8888-8888-888888888882', '88888888-8888-8888-8888-888888888883');
DELETE FROM loyalty_tiers WHERE id IN ('77777777-7777-7777-7777-777777777771', '77777777-7777-7777-7777-777777777772', '77777777-7777-7777-7777-777777777773');
DELETE FROM loyalty_programs WHERE id = '66666666-6666-6666-6666-666666666666';
DELETE FROM brand_customers WHERE id IN ('55555555-5555-5555-5555-555555555551', '55555555-5555-5555-5555-555555555552', '55555555-5555-5555-5555-555555555553');
DELETE FROM brand_members WHERE id IN ('44444444-4444-4444-4444-444444444441', '44444444-4444-4444-4444-444444444442');
DELETE FROM brands WHERE id = '33333333-3333-3333-3333-333333333333';
DELETE FROM users WHERE id IN ('11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111112', '22222222-2222-2222-2222-222222222221', '22222222-2222-2222-2222-222222222222');
