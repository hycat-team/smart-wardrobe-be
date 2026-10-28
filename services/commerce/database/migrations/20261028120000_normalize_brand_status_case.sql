-- +goose Up
-- Legacy rows created before commit 68a925a2 ("fix: constant value convention")
-- stored brand/member statuses in UPPERCASE, while the Go constants and every
-- query use lowercase. Those rows were silently excluded from membership checks,
-- which surfaced as 403s for brand staff on the whole portal.
UPDATE brand.brands SET status = lower(status) WHERE status <> lower(status);
UPDATE brand.brand_members
SET status = lower(status),
    role = lower(role)
WHERE status <> lower(status) OR role <> lower(role);
UPDATE brand.brand_customers SET status = lower(status) WHERE status <> lower(status);
UPDATE brand.brand_benefits SET status = lower(status) WHERE status <> lower(status);
UPDATE brand.benefit_redemptions SET status = lower(status) WHERE status <> lower(status);
UPDATE brand.brand_conversations SET status = lower(status) WHERE status <> lower(status);
UPDATE brand.loyalty_point_lots SET status = lower(status) WHERE status <> lower(status);
UPDATE brand.loyalty_programs SET rounding_mode = lower(rounding_mode) WHERE rounding_mode <> lower(rounding_mode);

ALTER TABLE brand.brands ALTER COLUMN status SET DEFAULT 'pending_review';
ALTER TABLE brand.brand_members ALTER COLUMN status SET DEFAULT 'active';
ALTER TABLE brand.brand_customers ALTER COLUMN status SET DEFAULT 'active';
ALTER TABLE brand.brand_benefits ALTER COLUMN status SET DEFAULT 'active';
ALTER TABLE brand.benefit_redemptions ALTER COLUMN status SET DEFAULT 'redeemed';
ALTER TABLE brand.brand_conversations ALTER COLUMN status SET DEFAULT 'open';
ALTER TABLE brand.loyalty_point_lots ALTER COLUMN status SET DEFAULT 'active';
ALTER TABLE brand.loyalty_programs ALTER COLUMN rounding_mode SET DEFAULT 'floor';
ALTER TABLE brand.brand_items ALTER COLUMN status SET DEFAULT 'draft';

-- +goose Down
-- Data normalisation cannot be reversed deterministically.
SELECT 1;
