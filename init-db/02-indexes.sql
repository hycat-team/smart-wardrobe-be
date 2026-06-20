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

CREATE INDEX IF NOT EXISTS idx_posts_hotness_dirty_at
ON posts (hotness_dirty_at ASC)
WHERE hotness_dirty_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_posts_created_at
ON posts (created_at DESC);

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
CREATE INDEX IF NOT EXISTS idx_wardrobe_items_last_used_at ON wardrobe_items (last_used_at);

-- Cải tiến UNIQUE ràng buộc kết hợp điều kiện (Conditional Unique Indexes) cho luồng Like
CREATE UNIQUE INDEX IF NOT EXISTS uidx_user_liked_post ON likes (user_id, post_id) WHERE post_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uidx_user_liked_comment ON likes (user_id, comment_id) WHERE comment_id IS NOT NULL;

-- Tối ưu luồng quét các gói cước hết hạn cần gia hạn tự động theo lô (Batch Processing)
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_expires ON user_subscriptions (expires_at, user_id) WHERE current_plan_kind = 1;

CREATE UNIQUE INDEX IF NOT EXISTS ux_single_active_default_free_plan ON subscription_plans(plan_kind) WHERE plan_kind = 0 AND is_active = TRUE;
CREATE UNIQUE INDEX IF NOT EXISTS ux_active_direct_purchase_per_user ON deposit_transactions(user_id)
WHERE transaction_type = 'DIRECT_PURCHASE' AND status IN (3, 0, 4, 5);
CREATE UNIQUE INDEX IF NOT EXISTS ux_webhook_provider_reference ON provider_webhook_inbox(provider, provider_reference, event_code) WHERE provider_reference IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS ux_webhook_payload_hash ON provider_webhook_inbox(provider, canonical_payload_hash) WHERE provider_reference IS NULL;
CREATE INDEX IF NOT EXISTS idx_deposit_reconciliation ON deposit_transactions(next_reconciliation_at, expires_at) WHERE status IN (0, 3, 4, 5);
CREATE INDEX IF NOT EXISTS idx_webhook_inbox_processing ON provider_webhook_inbox(next_processing_at, processing_lease_until) WHERE processing_status IN ('RECEIVED', 'RETRY_REQUIRED', 'PROCESSING');

-- ========================================================
-- CHỈ MỤC TÌM KIẾM TỪ KHÓA (LEXICAL GIN INDEX)
-- ========================================================

-- Tối ưu tìm kiếm FTS (Full-Text Search) trên các thuộc tính thời trang của tủ đồ
CREATE INDEX IF NOT EXISTS idx_wardrobe_items_lexical_search 
ON wardrobe_items 
USING gin (
  to_tsvector('simple', immutable_unaccent(lower(
    coalesce(color, '') || ' ' ||
    coalesce(style, '') || ' ' ||
    coalesce(material, '') || ' ' ||
    coalesce(pattern, '') || ' ' ||
    coalesce(fit, '') || ' ' ||
    coalesce(seasonality, '') || ' ' ||
    coalesce(description, '')
  )))
);
