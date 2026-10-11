-- +goose Up
-- +goose StatementBegin
-- Feature: drop identity.user_style_profiles (ADR 015).
--
-- The table backed the taste-embedding personalised feed, which was removed
-- with the community feed (ADR 002). After that, the only remaining access
-- path (UserRepository.GetStyleProfile) had no caller and no contract
-- exposure, and taste_embedding was never written by application code.
-- Dropping it also removes the pgvector HNSW index ustyles_embedding_cosine_idx.
DROP TABLE IF EXISTS identity.user_style_profiles CASCADE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Restore the production shape (matches database/init/01-schema.sql before the
-- drop; note the table lived in the identity schema after migration
-- 20260801000000_separate_module_schemas).
CREATE TABLE IF NOT EXISTS identity.user_style_profiles (
    user_id UUID PRIMARY KEY REFERENCES identity.users(id) ON DELETE CASCADE,
    taste_embedding VECTOR(768),
    preferred_colors JSONB,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ustyles_embedding_cosine_idx
ON identity.user_style_profiles
USING hnsw (taste_embedding vector_cosine_ops);
-- +goose StatementEnd
