-- ========================================================
-- Bảng quản lý các gói cước (Premium Plans)
-- ========================================================
CREATE TABLE subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    price NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    max_wardrobe_items INT NOT NULL,
    max_outfits INT NOT NULL,
    ai_outfit_daily_quota INT NOT NULL,
    ai_chat_daily_quota INT NOT NULL,
    duration_days INT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================================
-- Bảng người dùng (Users)
-- ========================================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    date_of_birth DATE,
    address VARCHAR(255),
    gender INT,
    role_slug VARCHAR(50) NOT NULL,
    
    -- Cấu hình thông số AI sinh học
    body_profile JSONB,
    
    status SMALLINT NOT NULL DEFAULT 0, -- 0: Active, 1: Inactive
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    avatar_url VARCHAR(500),
    avatar_public_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================================
-- Bảng liên kết gói cước người dùng (User Subscriptions)
-- ========================================================
CREATE TABLE user_subscriptions (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    subscription_plan_id UUID NOT NULL REFERENCES subscription_plans(id) ON DELETE RESTRICT,
    expires_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_auto_renew_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================================
-- Bảng hạn mức sử dụng hàng ngày (User Daily Quotas)
-- ========================================================
CREATE TABLE user_daily_quotas (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    outfit_recommend_count INT NOT NULL DEFAULT 0,
    ai_usage_count INT NOT NULL DEFAULT 0,
    last_reset_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================================
-- Bảng hồ sơ gu thời trang cá nhân (Taste Profiles)
-- ========================================================
CREATE TABLE user_style_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    taste_embedding VECTOR(768), -- Vector gu thời trang người dùng
    preferred_colors JSONB,      -- Lưu danh sách mảng màu tương tác phổ biến
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================================
-- Bảng quản lý session (Refresh Tokens)
-- ========================================================
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(500) UNIQUE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);


-- ========================================================
-- Bảng quản lý phiên hội thoại Chatbot (Sessions)
-- ========================================================
CREATE TABLE conversational_contexts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    context_summary TEXT DEFAULT '', -- Chuỗi nén tóm tắt hội thoại của riêng phiên này
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================================
-- Bảng tin nhắn Chatbot (Messages)
-- ========================================================
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    context_id UUID NOT NULL REFERENCES conversational_contexts(id) ON DELETE CASCADE,
    sender VARCHAR(50) NOT NULL, -- 'user' hoặc 'ai'
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================================
-- Bảng danh mục thời trang (Categories)
-- ========================================================
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================================
-- Bảng tủ đồ cá nhân (Wardrobe Items)
-- ========================================================
CREATE TABLE wardrobe_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id UUID REFERENCES categories(id) ON DELETE RESTRICT,
    image_url VARCHAR(500) NOT NULL,
    image_public_id VARCHAR(255) NOT NULL,
    color VARCHAR(50),
    style VARCHAR(100),
    material VARCHAR(100),
    pattern VARCHAR(100),
    fit VARCHAR(50),
    seasonality VARCHAR(100),
    description TEXT,
    status SMALLINT NOT NULL DEFAULT 0, -- 'in_wardrobe', 'selling', 'sold'
    item_type SMALLINT NOT NULL DEFAULT 0, -- 0: UserItem, 1: SystemCatalogItem
    embedding VECTOR(768),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================================
-- Bảng Outfit (Bộ phối đồ)
-- ========================================================
CREATE TABLE outfits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    cover_image_url VARCHAR(500),
    cover_public_id VARCHAR(255),
    status SMALLINT NOT NULL DEFAULT 1,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================================
-- Bảng trung gian Outfit - Wardrobe Items
-- ========================================================
CREATE TABLE outfit_items (
    outfit_id UUID NOT NULL REFERENCES outfits(id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES wardrobe_items(id) ON DELETE CASCADE,
    position_x DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    position_y DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    scale DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    layer_order SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (outfit_id, item_id)
);

-- ========================================================
-- Bảng bài đăng cộng đồng (Posts)
-- ========================================================
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_type VARCHAR(50) NOT NULL,
    title VARCHAR(255),
    content TEXT NOT NULL,
    contact_info VARCHAR(255),
    total_price DECIMAL(12, 2) DEFAULT 0.00,
    like_count INT DEFAULT 0,
    comment_count INT DEFAULT 0,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================================
-- Bảng lưu Snapshot điểm số bảng tin (1:1 với Posts)
-- ========================================================
CREATE TABLE post_score_snapshots (
    post_id UUID PRIMARY KEY REFERENCES posts(id) ON DELETE CASCADE,
    global_hotness_score DOUBLE PRECISION DEFAULT 0.0, -- Điểm thuật toán giảm nhiệt thời gian thực
    last_calculated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================================
-- Bảng chi tiết sản phẩm rao bán trong bài viết
-- ========================================================
CREATE TABLE post_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES wardrobe_items(id) ON DELETE CASCADE,
    price DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    item_condition SMALLINT NOT NULL DEFAULT 1,
    status SMALLINT NOT NULL DEFAULT 1, -- 0: hidden, 1: available, 2: sold
    buyer_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    transfer_state SMALLINT NOT NULL DEFAULT 0, -- 0: none, 1: pending, 2: accepted, 3: declined
    sold_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_post_item UNIQUE (post_id, item_id)
);

CREATE TABLE post_media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    media_type VARCHAR(20) NOT NULL,
    media_url VARCHAR(500) NOT NULL,
    public_id VARCHAR(255),
    sort_order SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================================
-- Bảng bình luận (Comments)
-- ========================================================
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================================
-- Bảng lượt thích (Likes)
-- ========================================================
CREATE TABLE likes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_like_target CHECK (
        (post_id IS NOT NULL AND comment_id IS NULL) OR
        (post_id IS NULL AND comment_id IS NOT NULL)
    )
);

-- ========================================================
-- Bảng lưu trữ ví người dùng (User Wallets)
-- ========================================================
CREATE TABLE user_wallets (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    balance NUMERIC(12,2) NOT NULL DEFAULT 0.00,
    currency VARCHAR(10) NOT NULL DEFAULT 'VND',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================================
-- Bảng lưu trữ lịch sử giao dịch nạp tiền và đăng ký (Deposit Transactions)
-- ========================================================
CREATE TABLE deposit_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount NUMERIC(12,2) NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'VND',
    status SMALLINT NOT NULL DEFAULT 0, -- 0: PENDING, 1: SUCCESS, 2: FAILED
    transaction_type VARCHAR(50) NOT NULL,
    subscription_plan_id UUID REFERENCES subscription_plans(id) ON DELETE SET NULL,
    order_code BIGSERIAL NOT NULL UNIQUE,
    gateway_reference VARCHAR(255) UNIQUE,
    gateway_details TEXT,
    payment_url VARCHAR(500) NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================================
-- Bảng lưu trữ lịch sử biến động số dư ví (Wallet Statements)
-- ========================================================
CREATE TABLE wallet_statements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount NUMERIC(12,2) NOT NULL,
    transaction_type VARCHAR(50) NOT NULL,
    previous_balance NUMERIC(12,2) NOT NULL,
    new_balance NUMERIC(12,2) NOT NULL,
    reference_id UUID REFERENCES deposit_transactions(id) ON DELETE SET NULL,
    description VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);