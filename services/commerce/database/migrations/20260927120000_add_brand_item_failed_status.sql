-- +goose Up
-- Feature 023 (US4): allow brand items to be marked as "failed" so system
-- analysis errors are distinct from the "draft" (needs-review) state.
-- Schema-qualified because the Goose DSN has no search_path.
ALTER TABLE brand.brand_items DROP CONSTRAINT IF EXISTS chk_brand_items_status;
ALTER TABLE brand.brand_items
    ADD CONSTRAINT chk_brand_items_status
    CHECK (status IN ('draft', 'active', 'archived', 'failed'));

-- +goose Down
ALTER TABLE brand.brand_items DROP CONSTRAINT IF EXISTS chk_brand_items_status;
ALTER TABLE brand.brand_items
    ADD CONSTRAINT chk_brand_items_status
    CHECK (status IN ('draft', 'active', 'archived'));
