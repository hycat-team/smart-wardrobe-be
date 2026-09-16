-- +goose Up
-- +goose StatementBegin

-- Subscription plan lifecycle fields
ALTER TABLE subscription_plans
    ADD COLUMN plan_kind SMALLINT,
    ADD COLUMN tier_rank INT,
    ADD COLUMN pricing_version BIGINT NOT NULL DEFAULT 1;

UPDATE subscription_plans
SET plan_kind = CASE
        WHEN slug = 'free' THEN 0
        WHEN duration_days IS NULL THEN 2
        ELSE 1
    END,
    tier_rank = CASE
        WHEN slug = 'free' THEN 0
        ELSE 1
    END;

ALTER TABLE subscription_plans
    ALTER COLUMN plan_kind SET DEFAULT 0,
    ALTER COLUMN plan_kind SET NOT NULL,
    ALTER COLUMN tier_rank SET DEFAULT 0,
    ALTER COLUMN tier_rank SET NOT NULL;

-- User subscription projection fields
ALTER TABLE user_subscriptions
    ADD COLUMN current_plan_code VARCHAR(100),
    ADD COLUMN current_tier_rank INT,
    ADD COLUMN current_plan_kind SMALLINT,
    ADD COLUMN current_benefit_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN started_at TIMESTAMPTZ,
    ADD COLUMN fallback_plan_id UUID REFERENCES subscription_plans(id) ON DELETE RESTRICT,
    ADD COLUMN fallback_plan_code VARCHAR(100),
    ADD COLUMN fallback_tier_rank INT,
    ADD COLUMN fallback_plan_kind SMALLINT,
    ADD COLUMN fallback_benefit_snapshot JSONB,
    ADD COLUMN last_deposit_transaction_id UUID,
    ADD COLUMN version BIGINT NOT NULL DEFAULT 0;

UPDATE user_subscriptions AS us
SET current_plan_code = sp.slug,
    current_tier_rank = sp.tier_rank,
    current_plan_kind = sp.plan_kind,
    started_at = us.created_at,
    expires_at = CASE
        WHEN sp.plan_kind IN (0, 2) THEN NULL
        WHEN sp.plan_kind = 1 AND us.expires_at IS NULL
            THEN us.created_at + make_interval(days => sp.duration_days)
        ELSE us.expires_at
    END,
    is_auto_renew_enabled = CASE
        WHEN sp.plan_kind = 1 THEN us.is_auto_renew_enabled
        ELSE FALSE
    END
FROM subscription_plans AS sp
WHERE sp.id = us.subscription_plan_id;

ALTER TABLE user_subscriptions
    ALTER COLUMN current_plan_code SET NOT NULL,
    ALTER COLUMN current_tier_rank SET NOT NULL,
    ALTER COLUMN current_plan_kind SET NOT NULL,
    ALTER COLUMN started_at SET DEFAULT NOW(),
    ALTER COLUMN started_at SET NOT NULL;

ALTER TABLE user_subscriptions
    DROP COLUMN is_active;

ALTER TABLE user_subscriptions
    ADD CONSTRAINT chk_user_subscription_kind_expiry
        CHECK (
            (current_plan_kind = 1 AND expires_at IS NOT NULL)
            OR (current_plan_kind IN (0, 2) AND expires_at IS NULL)
        ),
    ADD CONSTRAINT chk_user_subscription_auto_renew
        CHECK (
            current_plan_kind = 1
            OR is_auto_renew_enabled = FALSE
        ),
    ADD CONSTRAINT chk_user_subscription_fallback
        CHECK (
            (
                fallback_plan_id IS NULL
                AND fallback_plan_code IS NULL
                AND fallback_tier_rank IS NULL
                AND fallback_plan_kind IS NULL
                AND fallback_benefit_snapshot IS NULL
            )
            OR
            (
                fallback_plan_id IS NOT NULL
                AND fallback_plan_code IS NOT NULL
                AND fallback_tier_rank IS NOT NULL
                AND fallback_plan_kind = 2
                AND fallback_benefit_snapshot IS NOT NULL
                AND current_plan_kind = 1
                AND current_tier_rank > fallback_tier_rank
            )
        );

-- Deposit transaction lifecycle fields
ALTER TABLE deposit_transactions
    ADD COLUMN provider VARCHAR(50) NOT NULL DEFAULT 'PAYOS',
    ADD COLUMN payment_link_id VARCHAR(255),
    ADD COLUMN provider_status VARCHAR(50),
    ADD COLUMN successful_provider_reference VARCHAR(255),
    ADD COLUMN expires_at TIMESTAMPTZ,
    ADD COLUMN next_reconciliation_at TIMESTAMPTZ,
    ADD COLUMN reconciliation_attempts INT NOT NULL DEFAULT 0,
    ADD COLUMN processing_token UUID,
    ADD COLUMN processing_lease_until TIMESTAMPTZ,
    ADD COLUMN last_provider_error_code VARCHAR(100),
    ADD COLUMN last_provider_error_at TIMESTAMPTZ,
    ADD COLUMN failure_reason VARCHAR(100),
    ADD COLUMN cancelled_at TIMESTAMPTZ,
    ADD COLUMN expired_at TIMESTAMPTZ,
    ADD COLUMN plan_code VARCHAR(100),
    ADD COLUMN plan_name VARCHAR(100),
    ADD COLUMN tier_rank INT,
    ADD COLUMN plan_kind SMALLINT,
    ADD COLUMN purchased_duration_days INT,
    ADD COLUMN expected_amount NUMERIC(12,2),
    ADD COLUMN benefit_snapshot JSONB,
    ADD COLUMN pricing_version BIGINT,
    ADD COLUMN benefit_resolution VARCHAR(100),
    ADD COLUMN benefit_applied_at TIMESTAMPTZ,
    ADD COLUMN benefit_result_snapshot JSONB,
    ADD COLUMN wallet_credit_amount NUMERIC(12,2),
    ADD COLUMN subscription_version_before BIGINT,
    ADD COLUMN subscription_version_after BIGINT;

UPDATE deposit_transactions
SET expected_amount = amount,
    successful_provider_reference = gateway_reference;

UPDATE deposit_transactions AS dt
SET plan_code = sp.slug,
    plan_name = sp.name,
    tier_rank = sp.tier_rank,
    plan_kind = sp.plan_kind,
    purchased_duration_days = sp.duration_days,
    pricing_version = sp.pricing_version
FROM subscription_plans AS sp
WHERE dt.subscription_plan_id = sp.id;

ALTER TABLE deposit_transactions
    ALTER COLUMN expected_amount SET NOT NULL;

-- Wallet statement lifecycle fields
ALTER TABLE wallet_statements
    ADD COLUMN source_plan_code VARCHAR(100),
    ADD COLUMN source_tier_rank INT,
    ADD COLUMN active_tier_rank_at_completion INT,
    ADD COLUMN renewal_attempt_key VARCHAR(255);

-- Provider and subscription lifecycle tables
CREATE TABLE provider_payment_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL,
    provider_reference VARCHAR(255) NOT NULL,
    event_code VARCHAR(100) NOT NULL,
    order_code BIGINT NOT NULL,
    payment_link_id VARCHAR(255),
    amount NUMERIC(12,2) NOT NULL,
    currency VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_reference)
);

CREATE TABLE provider_webhook_inbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL,
    provider_reference VARCHAR(255),
    event_code VARCHAR(100) NOT NULL,
    order_code BIGINT NOT NULL,
    payment_link_id VARCHAR(255),
    amount NUMERIC(12,2) NOT NULL,
    currency VARCHAR(10) NOT NULL,
    canonical_payload_hash VARCHAR(64) NOT NULL,
    raw_payload JSONB NOT NULL,
    processing_status VARCHAR(50) NOT NULL,
    processing_attempts INT NOT NULL DEFAULT 0,
    next_processing_at TIMESTAMPTZ,
    processing_token UUID,
    processing_lease_until TIMESTAMPTZ,
    processing_error TEXT,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_subscription_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_key VARCHAR(255) NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    from_plan_code VARCHAR(100),
    from_tier_rank INT,
    to_plan_code VARCHAR(100),
    to_tier_rank INT,
    source_deposit_transaction_id UUID REFERENCES deposit_transactions(id),
    actor_admin_id UUID REFERENCES users(id),
    occurred_at TIMESTAMPTZ NOT NULL,
    effective_at TIMESTAMPTZ NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE subscription_renewal_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    renewal_attempt_key VARCHAR(255) NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expected_plan_id UUID NOT NULL REFERENCES subscription_plans(id),
    expected_expires_at TIMESTAMPTZ NOT NULL,
    expected_subscription_version BIGINT NOT NULL,
    status VARCHAR(50) NOT NULL,
    attempt_count INT NOT NULL DEFAULT 0,
    last_error_code VARCHAR(100),
    last_error_message TEXT,
    processing_token UUID,
    processing_lease_until TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Lifecycle indexes
CREATE UNIQUE INDEX ux_single_active_default_free_plan
    ON subscription_plans(plan_kind)
    WHERE plan_kind = 0 AND is_active = TRUE;

CREATE UNIQUE INDEX ux_active_direct_purchase_per_user
    ON deposit_transactions(user_id)
    WHERE transaction_type = 'DIRECT_PURCHASE'
      AND status IN (0, 3, 4, 5);

CREATE UNIQUE INDEX ux_wallet_statement_renewal_attempt
    ON wallet_statements(renewal_attempt_key)
    WHERE renewal_attempt_key IS NOT NULL;

CREATE UNIQUE INDEX ux_webhook_provider_reference
    ON provider_webhook_inbox(provider, provider_reference, event_code)
    WHERE provider_reference IS NOT NULL;

CREATE UNIQUE INDEX ux_webhook_payload_hash
    ON provider_webhook_inbox(provider, canonical_payload_hash)
    WHERE provider_reference IS NULL;

CREATE INDEX idx_webhook_inbox_processing
    ON provider_webhook_inbox(next_processing_at, processing_lease_until)
    WHERE processing_status IN ('RECEIVED', 'RETRY_REQUIRED', 'PROCESSING');

CREATE INDEX idx_deposit_reconciliation
    ON deposit_transactions(next_reconciliation_at, expires_at)
    WHERE status IN (0, 3, 4, 5);

CREATE INDEX idx_user_subscriptions_expires
    ON user_subscriptions(expires_at, user_id)
    WHERE current_plan_kind = 1;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Lifecycle indexes
DROP INDEX idx_user_subscriptions_expires;
DROP INDEX idx_deposit_reconciliation;
DROP INDEX idx_webhook_inbox_processing;
DROP INDEX ux_webhook_payload_hash;
DROP INDEX ux_webhook_provider_reference;
DROP INDEX ux_wallet_statement_renewal_attempt;
DROP INDEX ux_active_direct_purchase_per_user;
DROP INDEX ux_single_active_default_free_plan;

-- Provider and subscription lifecycle tables
DROP TABLE subscription_renewal_attempts;
DROP TABLE user_subscription_events;
DROP TABLE provider_webhook_inbox;
DROP TABLE provider_payment_events;

-- Wallet statement lifecycle fields
ALTER TABLE wallet_statements
    DROP COLUMN renewal_attempt_key,
    DROP COLUMN active_tier_rank_at_completion,
    DROP COLUMN source_tier_rank,
    DROP COLUMN source_plan_code;

-- Preserve provider reference before removing the success-only snapshot
UPDATE deposit_transactions
SET gateway_reference = successful_provider_reference
WHERE gateway_reference IS NULL
  AND successful_provider_reference IS NOT NULL;

-- Deposit transaction lifecycle fields
ALTER TABLE deposit_transactions
    DROP COLUMN subscription_version_after,
    DROP COLUMN subscription_version_before,
    DROP COLUMN wallet_credit_amount,
    DROP COLUMN benefit_result_snapshot,
    DROP COLUMN benefit_applied_at,
    DROP COLUMN benefit_resolution,
    DROP COLUMN pricing_version,
    DROP COLUMN benefit_snapshot,
    DROP COLUMN expected_amount,
    DROP COLUMN purchased_duration_days,
    DROP COLUMN plan_kind,
    DROP COLUMN tier_rank,
    DROP COLUMN plan_name,
    DROP COLUMN plan_code,
    DROP COLUMN expired_at,
    DROP COLUMN cancelled_at,
    DROP COLUMN failure_reason,
    DROP COLUMN last_provider_error_at,
    DROP COLUMN last_provider_error_code,
    DROP COLUMN processing_lease_until,
    DROP COLUMN processing_token,
    DROP COLUMN reconciliation_attempts,
    DROP COLUMN next_reconciliation_at,
    DROP COLUMN expires_at,
    DROP COLUMN successful_provider_reference,
    DROP COLUMN provider_status,
    DROP COLUMN payment_link_id,
    DROP COLUMN provider;

-- Restore the legacy active flag before removing projection fields
ALTER TABLE user_subscriptions
    ADD COLUMN is_active BOOLEAN;

UPDATE user_subscriptions
SET is_active = CASE
    WHEN current_plan_kind = 1 THEN expires_at > NOW()
    ELSE TRUE
END;

ALTER TABLE user_subscriptions
    ALTER COLUMN is_active SET DEFAULT TRUE,
    ALTER COLUMN is_active SET NOT NULL;

-- User subscription projection fields
ALTER TABLE user_subscriptions
    DROP CONSTRAINT chk_user_subscription_fallback,
    DROP CONSTRAINT chk_user_subscription_auto_renew,
    DROP CONSTRAINT chk_user_subscription_kind_expiry;

ALTER TABLE user_subscriptions
    DROP COLUMN version,
    DROP COLUMN last_deposit_transaction_id,
    DROP COLUMN fallback_benefit_snapshot,
    DROP COLUMN fallback_plan_kind,
    DROP COLUMN fallback_tier_rank,
    DROP COLUMN fallback_plan_code,
    DROP COLUMN fallback_plan_id,
    DROP COLUMN started_at,
    DROP COLUMN current_benefit_snapshot,
    DROP COLUMN current_plan_kind,
    DROP COLUMN current_tier_rank,
    DROP COLUMN current_plan_code;

-- Subscription plan lifecycle fields
ALTER TABLE subscription_plans
    DROP COLUMN pricing_version,
    DROP COLUMN tier_rank,
    DROP COLUMN plan_kind;

-- +goose StatementEnd
