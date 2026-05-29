-- ========================================================
-- DỮ LIỆU KHỞI TẠO MẶC ĐỊNH (SEED DATA)
-- ========================================================

-- Chèn các gói cước mặc định nếu chưa tồn tại
INSERT INTO subscription_plans (id, slug, name, price, max_wardrobe_items, max_outfits, ai_outfit_daily_quota, ai_chat_daily_quota, duration_days, is_active)
VALUES 
    ('ea78546b-f458-47df-bc53-ea849fb75630', 'free', 'Free Plan', 0.00, 50, 50, 3, 3, NULL, TRUE),
    ('cb591a26-9f4a-4e86-b489-8d195c80521e', 'premium-monthly', 'Premium Plan', 5000.00, 150, 150, 28, 30, 30, TRUE)
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
    ('8b7eb3de-2661-46ab-ae7d-b57bfd2d2a07', 'Áo khoác', 'ao-khoac')
ON CONFLICT (id) DO NOTHING;
