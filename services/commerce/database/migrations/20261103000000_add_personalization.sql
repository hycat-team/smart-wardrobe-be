-- +goose Up
-- +goose StatementBegin
-- Feature 033: personalized styling foundation (Commerce DDL ownership).
-- Adds fashion.user_style_profiles, fashion.style_signals and
-- wardrobe.wardrobe_items.last_recommended_at.
-- Additive only: no existing column is renamed, retyped or dropped.
-- The cross-schema FK to identity.users is the sanctioned exception of the
-- 030 ownership contract (§2, G-COMM).

CREATE TABLE IF NOT EXISTS fashion.user_style_profiles (
    user_id UUID PRIMARY KEY REFERENCES identity.users(id) ON DELETE CASCADE,
    styles JSONB NOT NULL DEFAULT '[]'::jsonb,
    excluded_styles JSONB NOT NULL DEFAULT '[]'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    style_embedding VECTOR(768),
    embedding_hash TEXT,
    embedding_model VARCHAR(100),
    learned_styles JSONB NOT NULL DEFAULT '{}'::jsonb,
    learned_embedding VECTOR(768),
    learned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_user_style_profiles_styles_array CHECK (jsonb_typeof(styles) = 'array'),
    CONSTRAINT chk_user_style_profiles_excluded_array CHECK (jsonb_typeof(excluded_styles) = 'array'),
    CONSTRAINT chk_user_style_profiles_learned_object CHECK (jsonb_typeof(learned_styles) = 'object'),
    CONSTRAINT chk_user_style_profiles_styles_limit CHECK (jsonb_array_length(styles) <= 50),
    CONSTRAINT chk_user_style_profiles_excluded_limit CHECK (jsonb_array_length(excluded_styles) <= 50)
);

CREATE TABLE IF NOT EXISTS fashion.style_signals (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    source SMALLINT NOT NULL,
    outfit_id UUID,
    recommendation_id UUID,
    fashion_item_ids UUID[] NOT NULL,
    style_snapshot JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_style_signals_source CHECK (source IN (1, 2, 3, 4)),
    CONSTRAINT chk_style_signals_items CHECK (cardinality(fashion_item_ids) > 0)
);

-- outfit_id is intentionally NOT a FK: a signal is history and must survive
-- outfit deletion. fashion_item_ids is a uuid[] so it cannot carry a FK either.
CREATE INDEX IF NOT EXISTS idx_style_signals_user_created
    ON fashion.style_signals (user_id, created_at DESC);

-- Guards feedback duplicates. Outfit-save signals carry no recommendation_id,
-- so their idempotency rests on the primary key (signal id from the outbox).
CREATE UNIQUE INDEX IF NOT EXISTS ux_style_signals_feedback
    ON fashion.style_signals (user_id, recommendation_id, source)
    WHERE recommendation_id IS NOT NULL;

ALTER TABLE wardrobe.wardrobe_items
    ADD COLUMN IF NOT EXISTS last_recommended_at TIMESTAMPTZ;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Reverse order: column first, then signals, then profiles.
ALTER TABLE wardrobe.wardrobe_items DROP COLUMN IF EXISTS last_recommended_at;
DROP TABLE IF EXISTS fashion.style_signals;
DROP TABLE IF EXISTS fashion.user_style_profiles;
-- +goose StatementEnd
