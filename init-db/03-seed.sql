-- ========================================================
-- DỮ LIỆU KHỞI TẠO MẶC ĐỊNH (SEED DATA)
-- ========================================================

-- Chèn các gói cước mặc định nếu chưa tồn tại
INSERT INTO subscription_plans (id, slug, name, price, max_wardrobe_items, max_outfits, ai_outfit_daily_quota, ai_chat_daily_quota, duration_days, plan_kind, tier_rank, pricing_version, is_active)
VALUES 
    ('ea78546b-f458-47df-bc53-ea849fb75630', 'free', 'Free Plan', 0.00, 100, 100, 3, 3, NULL, 0, 0, 1, TRUE),
    ('cb591a26-9f4a-4e86-b489-8d195c80521e', 'premium-monthly', 'Premium Plan', 5000.00, 300, 300, 15, 20, 30, 1, 1, 1, TRUE)
ON CONFLICT (id) DO NOTHING;

-- Chèn các danh mục trang phục mặc định
INSERT INTO categories (id, name, slug)
VALUES
    ('8b7eb3de-2661-46ab-ae7d-b57bfd2d2a01', 'Áo', 'ao'),
    ('8b7eb3de-2661-46ab-ae7d-b57bfd2d2a02', 'Quần', 'quan'),
    ('8b7eb3de-2661-46ab-ae7d-b57bfd2d2a03', 'Mũ', 'mu'),
    ('8b7eb3de-2661-46ab-ae7d-b57bfd2d2a04', 'Giày', 'giay'),
    ('8b7eb3de-2661-46ab-ae7d-b57bfd2d2a05', 'Phụ kiện', 'phu-kien'),
    ('8b7eb3de-2661-46ab-ae7d-b57bfd2d2a06', 'Váy', 'vay'),
    ('8b7eb3de-2661-46ab-ae7d-b57bfd2d2a07', 'Áo khoác', 'ao-khoac'),
    ('8b7eb3de-2661-46ab-ae7d-b57bfd2d2a08', 'Khác', 'other')
ON CONFLICT (id) DO NOTHING;

-- Chèn tài khoản Admin hệ thống mặc định (password: admin123)
INSERT INTO users (id, username, email, password_hash, first_name, last_name, role_slug, status)
VALUES
    ('ad11ad11-ad11-ad11-ad11-ad11ad11ad11', 'admin', 'admin@smartwardrobe.com', '$2a$11$kXWLREY8wu6wEQlONWcLveV2jeE/Tx9MS4vOlQqXmcQ9VASP0NMhu', 'System', 'Admin', 'admin', 0),
    ('ad11ad11-ad11-ad11-ad11-ad11ad11ad12', 'user', 'user@smartwardrobe.com', '$2a$11$kXWLREY8wu6wEQlONWcLveV2jeE/Tx9MS4vOlQqXmcQ9VASP0NMhu', 'System', 'User', 'user', 0)
ON CONFLICT (id) DO NOTHING;

-- Chèn các trang phục mẫu trong Global Fashion Catalog (item_type = 1)
INSERT INTO wardrobe_items (id, user_id, category_id, image_url, image_public_id, color, style, material, pattern, fit, seasonality, description, status, item_type)
VALUES
    ('ca7ca7ca-ca7c-ca7c-ca7c-ca7ca7ca7c01', 'ad11ad11-ad11-ad11-ad11-ad11ad11ad11', '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a01', '', '', 'Trắng', 'Casual', 'Cotton', 'Trơn', 'Regular', 'Mùa hè', 'Áo thun trắng basic mẫu mực dễ phối đồ', 0, 1),
    ('ca7ca7ca-ca7c-ca7c-ca7c-ca7ca7ca7c02', 'ad11ad11-ad11-ad11-ad11-ad11ad11ad11', '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a02', '', '', 'Xanh dương', 'Casual', 'Denim', 'Trơn', 'Regular', 'Bốn mùa', 'Quần jeans xanh trơn bền bỉ cổ điển', 0, 1),
    ('ca7ca7ca-ca7c-ca7c-ca7c-ca7ca7ca7c03', 'ad11ad11-ad11-ad11-ad11-ad11ad11ad11', '8b7eb3de-2661-46ab-ae7d-b57bfd2d2a04', '', '', 'Trắng', 'Sporty', 'Da nhân tạo', 'Trơn', 'Regular', 'Bốn mùa', 'Giày sneaker trắng thể thao năng động', 0, 1)
ON CONFLICT (id) DO NOTHING;
