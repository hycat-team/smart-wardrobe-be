-- +goose Up
-- +goose StatementBegin
-- 033 D2: personalization is a plan entitlement, so it follows the existing
-- plan -> benefit snapshot -> contract DTO -> overview cache -> proto chain
-- instead of being re-derived from `price > 0` at runtime.
--
-- The column is NOT NULL with a false default: adding it online never rewrites
-- the table, and every plan created after this migration starts without the
-- entitlement until an administrator enables it explicitly.
ALTER TABLE subscription.subscription_plans
    ADD COLUMN IF NOT EXISTS personalization_enabled boolean NOT NULL DEFAULT false;

-- One-time backfill for plans that are already selling at this deploy. The
-- runtime source of truth stays the column, not the price: zero-priced
-- promotional plans stay disabled, and plans created later default to false.
UPDATE subscription.subscription_plans
SET personalization_enabled = true,
    updated_at = now()
WHERE is_active = true
  AND price > 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE subscription.subscription_plans
    DROP COLUMN IF EXISTS personalization_enabled;
-- +goose StatementEnd
