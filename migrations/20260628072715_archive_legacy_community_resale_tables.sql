-- +goose Up
-- Phase 02 B2B2C rebuild: archive legacy community/resale schema after routes and workers
-- have been removed from the MVP runtime.
DROP TABLE IF EXISTS likes;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS transfer_requests;
DROP TABLE IF EXISTS post_media;
DROP TABLE IF EXISTS post_score_snapshots;
DROP TABLE IF EXISTS post_items;
DROP TABLE IF EXISTS posts;

-- +goose Down
CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_id VARCHAR(32) NOT NULL UNIQUE,
    post_type VARCHAR(50) NOT NULL,
    title VARCHAR(255),
    content TEXT NOT NULL,
    contact_info VARCHAR(255),
    total_price DECIMAL(12, 2) DEFAULT 0.00,
    like_count INT DEFAULT 0,
    comment_count INT DEFAULT 0,
    hotness_dirty_at TIMESTAMP WITH TIME ZONE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS post_score_snapshots (
    post_id UUID PRIMARY KEY REFERENCES posts(id) ON DELETE CASCADE,
    global_hotness_score DOUBLE PRECISION DEFAULT 0.0,
    last_calculated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS post_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES wardrobe_items(id) ON DELETE CASCADE,
    price DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    item_condition SMALLINT NOT NULL DEFAULT 1,
    status SMALLINT NOT NULL DEFAULT 1,
    buyer_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    transfer_state SMALLINT NOT NULL DEFAULT 0,
    sold_at TIMESTAMP WITH TIME ZONE,
    declined_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_post_item UNIQUE (post_id, item_id)
);

CREATE TABLE IF NOT EXISTS post_media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    media_type VARCHAR(20) NOT NULL,
    media_url VARCHAR(500) NOT NULL,
    public_id VARCHAR(255),
    sort_order SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transfer_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_item_id UUID NOT NULL REFERENCES post_items(id) ON DELETE CASCADE,
    buyer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_pending_request UNIQUE (post_item_id, buyer_id)
);

CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS likes (
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

CREATE INDEX IF NOT EXISTS idx_snapshots_global_hotness_score
ON post_score_snapshots (global_hotness_score DESC);

CREATE INDEX IF NOT EXISTS idx_posts_hotness_dirty_at
ON posts (hotness_dirty_at ASC)
WHERE hotness_dirty_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_posts_created_at
ON posts (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_post_items_status
ON post_items (status);

CREATE INDEX IF NOT EXISTS idx_posts_user_id
ON posts (user_id);

CREATE INDEX IF NOT EXISTS idx_post_items_item_id
ON post_items (item_id);

CREATE INDEX IF NOT EXISTS idx_comments_post_id
ON comments (post_id);

CREATE INDEX IF NOT EXISTS idx_likes_post_id
ON likes (post_id)
WHERE post_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uidx_user_liked_post
ON likes (user_id, post_id)
WHERE post_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uidx_user_liked_comment
ON likes (user_id, comment_id)
WHERE comment_id IS NOT NULL;
