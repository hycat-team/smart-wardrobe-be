-- ========================================================
-- DỮ LIỆU KHỞI TẠO MẶC ĐỊNH (SEED DATA)
-- ========================================================

-- Chèn các gói cước mặc định nếu chưa tồn tại
INSERT INTO subscription_plans (id, name, price, max_wardrobe_items, ai_outfit_daily_quota, ai_chat_daily_quota, duration_days, is_active)
VALUES 
    ('ea78546b-f458-47df-bc53-ea849fb75630', 'Free Plan', 0.00, 50, 5, 10, NULL, TRUE),
    ('cb591a26-9f4a-4e86-b489-8d195c80521e', 'Premium Plan', 150000.00, 500, 50, 100, 30, TRUE)
ON CONFLICT (id) DO NOTHING;
