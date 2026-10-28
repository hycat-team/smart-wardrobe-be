-- +goose Up
ALTER TABLE brand.brands DROP CONSTRAINT IF EXISTS chk_brands_status;
ALTER TABLE brand.brands
    ADD CONSTRAINT chk_brands_status
    CHECK (status IN ('pending_review', 'active', 'suspended', 'archived', 'rejected'));

-- +goose Down
UPDATE brand.brands SET status = 'archived' WHERE status = 'rejected';
ALTER TABLE brand.brands DROP CONSTRAINT IF EXISTS chk_brands_status;
ALTER TABLE brand.brands
    ADD CONSTRAINT chk_brands_status
    CHECK (status IN ('pending_review', 'active', 'suspended', 'archived'));
