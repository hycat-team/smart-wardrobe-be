-- +goose Up
-- Community module schema (feature 022). FK to identity.users is allowed
-- cross-schema (user FKs kept per architecture §VII); no cross-module FK to
-- wardrobe.outfits — outfit linkage is enforced in the application layer.

CREATE SCHEMA IF NOT EXISTS community;

CREATE TABLE IF NOT EXISTS community.posts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    public_id varchar(32) NOT NULL UNIQUE,
    user_id uuid NOT NULL REFERENCES identity.users(id),
    post_type varchar(20) NOT NULL,
    title varchar(150),
    content text NOT NULL DEFAULT '',
    outfit_id uuid,
    like_count integer NOT NULL DEFAULT 0,
    comment_count integer NOT NULL DEFAULT 0,
    status varchar(20) NOT NULL DEFAULT 'published',
    moderated_by uuid,
    hidden_reason varchar(30),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_posts_post_type CHECK (post_type IN ('outfit', 'media')),
    CONSTRAINT chk_posts_status CHECK (status IN ('published', 'hidden', 'deleted')),
    CONSTRAINT chk_posts_hidden_reason CHECK (hidden_reason IS NULL OR hidden_reason IN ('admin', 'author_disabled')),
    CONSTRAINT chk_posts_outfit_ref CHECK (
        (post_type = 'outfit' AND outfit_id IS NOT NULL) OR
        (post_type = 'media' AND outfit_id IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_posts_published_created ON community.posts (created_at DESC, id DESC) WHERE status = 'published';
CREATE INDEX IF NOT EXISTS idx_posts_user_created ON community.posts (user_id, created_at DESC) WHERE status = 'published';
CREATE INDEX IF NOT EXISTS idx_posts_type_created ON community.posts (post_type, created_at DESC) WHERE status = 'published';
CREATE INDEX IF NOT EXISTS idx_posts_outfit ON community.posts (outfit_id) WHERE outfit_id IS NOT NULL AND status = 'published';

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_posts_search ON community.posts USING gin (lower(left(content, 200)) gin_trgm_ops) WHERE status = 'published';

CREATE TABLE IF NOT EXISTS community.post_media (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id uuid NOT NULL REFERENCES community.posts(id) ON DELETE CASCADE,
    media_type varchar(10) NOT NULL,
    media_url varchar(500) NOT NULL,
    public_id varchar(255),
    sort_order smallint NOT NULL DEFAULT 0,
    CONSTRAINT chk_post_media_type CHECK (media_type IN ('image', 'video'))
);

CREATE INDEX IF NOT EXISTS idx_post_media_post_sort ON community.post_media (post_id, sort_order);

CREATE TABLE IF NOT EXISTS community.post_likes (
    post_id uuid NOT NULL REFERENCES community.posts(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES identity.users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (post_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_post_likes_post ON community.post_likes (post_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_post_likes_user ON community.post_likes (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS community.post_comments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id uuid NOT NULL REFERENCES community.posts(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES identity.users(id),
    parent_comment_id uuid REFERENCES community.post_comments(id) ON DELETE CASCADE,
    root_comment_id uuid,
    content text NOT NULL,
    status varchar(20) NOT NULL DEFAULT 'published',
    moderated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_post_comments_status CHECK (status IN ('published', 'hidden', 'deleted')),
    CONSTRAINT chk_post_comments_depth CHECK (parent_comment_id IS NULL OR (root_comment_id IS NOT NULL AND parent_comment_id <> id))
);

CREATE INDEX IF NOT EXISTS idx_comments_post_parent_created ON community.post_comments (post_id, parent_comment_id, created_at);
CREATE INDEX IF NOT EXISTS idx_comments_root_created ON community.post_comments (root_comment_id, created_at);

CREATE TABLE IF NOT EXISTS community.follows (
    follower_id uuid NOT NULL REFERENCES identity.users(id),
    followee_id uuid NOT NULL REFERENCES identity.users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (follower_id, followee_id),
    CONSTRAINT chk_follows_no_self CHECK (follower_id <> followee_id)
);

CREATE INDEX IF NOT EXISTS idx_follows_follower_created ON community.follows (follower_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_follows_followee_created ON community.follows (followee_id, created_at DESC);

CREATE TABLE IF NOT EXISTS community.user_follow_stats (
    user_id uuid PRIMARY KEY REFERENCES identity.users(id),
    follower_count integer NOT NULL DEFAULT 0,
    following_count integer NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS community.user_follow_stats;
DROP TABLE IF EXISTS community.follows;
DROP TABLE IF EXISTS community.post_comments;
DROP TABLE IF EXISTS community.post_likes;
DROP TABLE IF EXISTS community.post_media;
DROP TABLE IF EXISTS community.posts;
DROP SCHEMA IF EXISTS community CASCADE;
