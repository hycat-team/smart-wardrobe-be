-- ========================================================
-- CÁC CHỈ MỤC VECTOR (PGVECTOR HNSW INDEXES)
-- ========================================================

-- Chỉ mục HNSW cho Tủ đồ cá nhân (Phục vụ Module 3: RAG Outfit)
CREATE INDEX IF NOT EXISTS witems_embedding_cosine_idx 
ON wardrobe_items 
USING hnsw (embedding vector_cosine_ops);

-- Chỉ mục HNSW cho Hồ sơ gu thời trang (Phục vụ Module 4: Newsfeed Ranking)
CREATE INDEX IF NOT EXISTS ustyles_embedding_cosine_idx 
ON user_style_profiles 
USING hnsw (taste_embedding vector_cosine_ops);


-- ========================================================
-- CÁC CHỈ MỤC LOGIC THUẬT TOÁN (B-TREE INDEXES)
-- ========================================================

-- Tối ưu Stage 1 lọc thô Newsfeed: Quét nhanh bài viết có điểm số cao nhất từ snapshot
CREATE INDEX IF NOT EXISTS idx_snapshots_global_hotness_score 
ON post_score_snapshots (global_hotness_score DESC);

-- Tối ưu điều kiện lọc cứng: Chỉ quét các sản phẩm đang ở trạng thái rao bán (status = 1)
CREATE INDEX IF NOT EXISTS idx_post_items_status 
ON post_items (status);

-- Tối ưu luồng bốc lịch sử tin nhắn của một phiên Chatbot theo thời gian giảm dần
CREATE INDEX IF NOT EXISTS idx_messages_context_created_at 
ON messages (context_id, created_at DESC);


-- ========================================================
-- TỐI ƯU KHÓA NGOẠI (FOREIGN KEYS) TRÁNH SEQUENTIAL SCAN KHI JOIN BẢNG
-- ========================================================

-- Tối ưu luồng lấy danh sách bài đăng của một User cụ thể trên trang cá nhân
CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts (user_id);

-- Tối ưu luồng JOIN lấy thông tin trang phục nằm trong bài viết marketplace
CREATE INDEX IF NOT EXISTS idx_post_items_item_id ON post_items (item_id);

-- Tối ưu luồng bốc toàn bộ danh sách bình luận của một bài viết cụ thể
CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments (post_id);

-- Tối ưu luồng đếm số lượt thích hoặc kiểm tra trạng thái nút Like của bài viết
CREATE INDEX IF NOT EXISTS idx_likes_post_id ON likes (post_id) WHERE post_id IS NOT NULL;

-- Tối ưu luồng kiểm tra nhanh hạn mức cấu hình gói cước hoạt động của User khi chạy Quota Engine
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_plan_id ON user_subscriptions (subscription_plan_id);

-- Tối ưu luồng lấy danh sách đồ trong tủ theo danh mục thời trang phân loại
CREATE INDEX IF NOT EXISTS idx_wardrobe_items_category_id ON wardrobe_items (category_id);

-- Cải tiến UNIQUE ràng buộc kết hợp điều kiện (Conditional Unique Indexes) cho luồng Like
CREATE UNIQUE INDEX IF NOT EXISTS uidx_user_liked_post ON likes (user_id, post_id) WHERE post_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uidx_user_liked_comment ON likes (user_id, comment_id) WHERE comment_id IS NOT NULL;