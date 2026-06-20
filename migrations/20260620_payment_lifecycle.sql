-- +goose Up
-- +goose StatementBegin
ALTER TABLE subscription_plans ADD COLUMN IF NOT EXISTS plan_kind SMALLINT;
ALTER TABLE subscription_plans ADD COLUMN IF NOT EXISTS tier_rank INT;
ALTER TABLE subscription_plans ADD COLUMN IF NOT EXISTS pricing_version BIGINT NOT NULL DEFAULT 1;
UPDATE subscription_plans SET plan_kind = CASE WHEN slug = 'free' THEN 0 WHEN duration_days IS NULL THEN 2 ELSE 1 END WHERE plan_kind IS NULL;
UPDATE subscription_plans SET tier_rank = CASE WHEN slug = 'free' THEN 0 ELSE 1 END WHERE tier_rank IS NULL;
ALTER TABLE subscription_plans ALTER COLUMN plan_kind SET NOT NULL;
ALTER TABLE subscription_plans ALTER COLUMN tier_rank SET NOT NULL;

ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS current_plan_code VARCHAR(100);
ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS current_tier_rank INT;
ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS current_plan_kind SMALLINT;
ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS current_benefit_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ;
ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS fallback_plan_id UUID REFERENCES subscription_plans(id) ON DELETE RESTRICT;
ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS fallback_plan_code VARCHAR(100);
ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS fallback_tier_rank INT;
ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS fallback_plan_kind SMALLINT;
ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS fallback_benefit_snapshot JSONB;
ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS last_deposit_transaction_id UUID;
ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;
UPDATE user_subscriptions us SET current_plan_code = sp.slug, current_tier_rank = sp.tier_rank, current_plan_kind = sp.plan_kind, started_at = us.created_at FROM subscription_plans sp WHERE sp.id = us.subscription_plan_id AND us.current_plan_code IS NULL;
ALTER TABLE user_subscriptions ALTER COLUMN current_plan_code SET NOT NULL;
ALTER TABLE user_subscriptions ALTER COLUMN current_tier_rank SET NOT NULL;
ALTER TABLE user_subscriptions ALTER COLUMN current_plan_kind SET NOT NULL;
ALTER TABLE user_subscriptions ALTER COLUMN started_at SET NOT NULL;
ALTER TABLE user_subscriptions DROP COLUMN IF EXISTS is_active;
ALTER TABLE user_subscriptions DROP CONSTRAINT IF EXISTS chk_user_subscription_kind_expiry;
ALTER TABLE user_subscriptions ADD CONSTRAINT chk_user_subscription_kind_expiry CHECK ((current_plan_kind = 1 AND expires_at IS NOT NULL) OR (current_plan_kind IN (0,2) AND expires_at IS NULL));
ALTER TABLE user_subscriptions DROP CONSTRAINT IF EXISTS chk_user_subscription_auto_renew;
ALTER TABLE user_subscriptions ADD CONSTRAINT chk_user_subscription_auto_renew CHECK (current_plan_kind = 1 OR is_auto_renew_enabled = FALSE);
ALTER TABLE user_subscriptions DROP CONSTRAINT IF EXISTS chk_user_subscription_fallback;
ALTER TABLE user_subscriptions ADD CONSTRAINT chk_user_subscription_fallback CHECK ((fallback_plan_id IS NULL AND fallback_plan_code IS NULL AND fallback_tier_rank IS NULL AND fallback_plan_kind IS NULL AND fallback_benefit_snapshot IS NULL) OR (fallback_plan_id IS NOT NULL AND fallback_plan_code IS NOT NULL AND fallback_tier_rank IS NOT NULL AND fallback_plan_kind = 2 AND fallback_benefit_snapshot IS NOT NULL AND current_plan_kind = 1 AND current_tier_rank > fallback_tier_rank));

ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS provider VARCHAR(50) NOT NULL DEFAULT 'PAYOS';
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS payment_link_id VARCHAR(255);
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS provider_status VARCHAR(50);
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS successful_provider_reference VARCHAR(255);
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS next_reconciliation_at TIMESTAMPTZ;
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS reconciliation_attempts INT NOT NULL DEFAULT 0;
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS processing_token UUID;
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS processing_lease_until TIMESTAMPTZ;
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS last_provider_error_code VARCHAR(100);
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS last_provider_error_at TIMESTAMPTZ;
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS failure_reason VARCHAR(100);
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMPTZ;
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS expired_at TIMESTAMPTZ;
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS plan_code VARCHAR(100);
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS plan_name VARCHAR(100);
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS tier_rank INT;
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS plan_kind SMALLINT;
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS purchased_duration_days INT;
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS expected_amount NUMERIC(12,2);
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS benefit_snapshot JSONB;
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS pricing_version BIGINT;
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS benefit_resolution VARCHAR(100);
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS benefit_applied_at TIMESTAMPTZ;
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS benefit_result_snapshot JSONB;
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS wallet_credit_amount NUMERIC(12,2);
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS subscription_version_before BIGINT;
ALTER TABLE deposit_transactions ADD COLUMN IF NOT EXISTS subscription_version_after BIGINT;
UPDATE deposit_transactions SET expected_amount = amount WHERE expected_amount IS NULL;
UPDATE deposit_transactions SET successful_provider_reference = gateway_reference WHERE successful_provider_reference IS NULL AND gateway_reference IS NOT NULL;
UPDATE deposit_transactions dt SET plan_code = sp.slug, plan_name = sp.name, tier_rank = sp.tier_rank, plan_kind = sp.plan_kind, purchased_duration_days = sp.duration_days, pricing_version = sp.pricing_version FROM subscription_plans sp WHERE dt.subscription_plan_id = sp.id AND dt.plan_code IS NULL;
ALTER TABLE deposit_transactions ALTER COLUMN expected_amount SET NOT NULL;

ALTER TABLE wallet_statements ADD COLUMN IF NOT EXISTS source_plan_code VARCHAR(100);
ALTER TABLE wallet_statements ADD COLUMN IF NOT EXISTS source_tier_rank INT;
ALTER TABLE wallet_statements ADD COLUMN IF NOT EXISTS active_tier_rank_at_completion INT;
ALTER TABLE wallet_statements ADD COLUMN IF NOT EXISTS renewal_attempt_key VARCHAR(255);

CREATE TABLE IF NOT EXISTS provider_payment_events (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), provider VARCHAR(50) NOT NULL, provider_reference VARCHAR(255) NOT NULL, event_code VARCHAR(100) NOT NULL, order_code BIGINT NOT NULL, payment_link_id VARCHAR(255), amount NUMERIC(12,2) NOT NULL, currency VARCHAR(10) NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE(provider, provider_reference));
CREATE TABLE IF NOT EXISTS provider_webhook_inbox (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), provider VARCHAR(50) NOT NULL, provider_reference VARCHAR(255), event_code VARCHAR(100) NOT NULL, order_code BIGINT NOT NULL, payment_link_id VARCHAR(255), amount NUMERIC(12,2) NOT NULL, currency VARCHAR(10) NOT NULL, canonical_payload_hash VARCHAR(64) NOT NULL, raw_payload JSONB NOT NULL, processing_status VARCHAR(50) NOT NULL, processing_attempts INT NOT NULL DEFAULT 0, next_processing_at TIMESTAMPTZ, processing_token UUID, processing_lease_until TIMESTAMPTZ, processing_error TEXT, received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), processed_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
CREATE TABLE IF NOT EXISTS user_subscription_events (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), event_key VARCHAR(255) NOT NULL UNIQUE, user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, event_type VARCHAR(100) NOT NULL, from_plan_code VARCHAR(100), from_tier_rank INT, to_plan_code VARCHAR(100), to_tier_rank INT, source_deposit_transaction_id UUID REFERENCES deposit_transactions(id), actor_admin_id UUID REFERENCES users(id), occurred_at TIMESTAMPTZ NOT NULL, effective_at TIMESTAMPTZ NOT NULL, metadata JSONB, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
CREATE TABLE IF NOT EXISTS subscription_renewal_attempts (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), renewal_attempt_key VARCHAR(255) NOT NULL UNIQUE, user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, expected_plan_id UUID NOT NULL REFERENCES subscription_plans(id), expected_expires_at TIMESTAMPTZ NOT NULL, expected_subscription_version BIGINT NOT NULL, status VARCHAR(50) NOT NULL, attempt_count INT NOT NULL DEFAULT 0, last_error_code VARCHAR(100), last_error_message TEXT, processing_token UUID, processing_lease_until TIMESTAMPTZ, completed_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW());

CREATE UNIQUE INDEX IF NOT EXISTS ux_single_active_default_free_plan ON subscription_plans(plan_kind) WHERE plan_kind = 0 AND is_active = TRUE;
CREATE UNIQUE INDEX IF NOT EXISTS ux_active_direct_purchase_per_user ON deposit_transactions(user_id) WHERE transaction_type = 'DIRECT_PURCHASE' AND status IN (3,0,4,5);
CREATE UNIQUE INDEX IF NOT EXISTS ux_wallet_statement_renewal_attempt ON wallet_statements(renewal_attempt_key) WHERE renewal_attempt_key IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS ux_webhook_provider_reference ON provider_webhook_inbox(provider, provider_reference, event_code) WHERE provider_reference IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS ux_webhook_payload_hash ON provider_webhook_inbox(provider, canonical_payload_hash) WHERE provider_reference IS NULL;
CREATE INDEX IF NOT EXISTS idx_deposit_reconciliation ON deposit_transactions(next_reconciliation_at, expires_at) WHERE status IN (0,3,4,5);
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_expires ON user_subscriptions(expires_at, user_id) WHERE current_plan_kind = 1;

SELECT setval(pg_get_serial_sequence('deposit_transactions','order_code'), GREATEST(COALESCE((SELECT MAX(order_code) FROM deposit_transactions),1),1), true);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS ux_active_direct_purchase_per_user;
DROP INDEX IF EXISTS ux_single_active_default_free_plan;
DROP INDEX IF EXISTS ux_webhook_provider_reference;
DROP INDEX IF EXISTS ux_webhook_payload_hash;
DROP INDEX IF EXISTS idx_deposit_reconciliation;
DROP TABLE IF EXISTS subscription_renewal_attempts;
DROP TABLE IF EXISTS user_subscription_events;
DROP TABLE IF EXISTS provider_webhook_inbox;
DROP TABLE IF EXISTS provider_payment_events;
ALTER TABLE user_subscriptions ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;
-- +goose StatementEnd
