-- +goose Up
-- +goose StatementBegin
-- 030 DDL ownership handover (US1): AI service baseline. Sole owner of all
-- seven aiexecutor tables (FR-001). Self-standing definitions in FINAL state:
-- the grant window check carries the explicit new name from the start, the
-- source_event_id and audit-log indexes are included, nullability matches the
-- post-20260930120000 state (nullable operations cap, nullable ledger grant,
-- nullable event ledger/user).
--
-- Convergence: every CREATE/INDEX uses IF NOT EXISTS and seeds use
-- no-overwrite guards, so databases built from the old Account history boot
-- this file as a convergent no-op (only the one-time constraint rename
-- below does real work there). Fresh databases get the full chain from blank,
-- so this file has no to_regclass precondition: it must CREATE, not gate.
-- Seeds hold RECORDED chain values (chain-final operation caps verified on
-- dev 2026-10-16); unrecorded hand-edits on long-lived databases are data
-- drift outside schema convergence (see verification-log T014 note).

CREATE SCHEMA IF NOT EXISTS aiexecutor;

CREATE TABLE IF NOT EXISTS aiexecutor.ai_cost_policies (
    id UUID DEFAULT gen_random_uuid() NOT NULL,
    code VARCHAR(100) NOT NULL,
    version BIGINT NOT NULL,
    name VARCHAR(150) NOT NULL,
    enforcement_mode VARCHAR(30) NOT NULL,
    period_days INT NOT NULL,
    hard_cost_micro_vnd BIGINT,
    compact_threshold_bps INT DEFAULT 8000 NOT NULL,
    free_route_threshold_bps INT DEFAULT 9200 NOT NULL,
    unknown_hold_minutes INT DEFAULT 1440 NOT NULL,
    max_unknown_paid_requests_per_day INT DEFAULT 2 NOT NULL,
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT ai_cost_policies_pkey PRIMARY KEY (id),
    CONSTRAINT ai_cost_policies_code_version_key UNIQUE (code, version),
    CONSTRAINT ai_cost_policies_check CHECK ((enforcement_mode::text <> 'strict'::text) OR (hard_cost_micro_vnd IS NOT NULL)),
    CONSTRAINT ai_cost_policies_check1 CHECK ((compact_threshold_bps <= free_route_threshold_bps)),
    CONSTRAINT ai_cost_policies_compact_threshold_bps_check CHECK ((compact_threshold_bps >= 0) AND (compact_threshold_bps <= 10000)),
    CONSTRAINT ai_cost_policies_enforcement_mode_check CHECK ((enforcement_mode::text = ANY (ARRAY['strict', 'observe_only', 'free_only']))),
    CONSTRAINT ai_cost_policies_free_route_threshold_bps_check CHECK ((free_route_threshold_bps >= 0) AND (free_route_threshold_bps <= 10000)),
    CONSTRAINT ai_cost_policies_max_unknown_paid_requests_per_day_check CHECK ((max_unknown_paid_requests_per_day >= 0)),
    CONSTRAINT ai_cost_policies_period_days_check CHECK ((period_days > 0)),
    CONSTRAINT ai_cost_policies_unknown_hold_minutes_check CHECK ((unknown_hold_minutes > 0))
);

CREATE TABLE IF NOT EXISTS aiexecutor.ai_cost_policy_operations (
    id UUID DEFAULT gen_random_uuid() NOT NULL,
    policy_id UUID NOT NULL REFERENCES aiexecutor.ai_cost_policies(id) ON DELETE CASCADE,
    operation VARCHAR(30) NOT NULL,
    normal_route VARCHAR(50) NOT NULL,
    reduced_route VARCHAR(50) NOT NULL,
    free_route VARCHAR(50) NOT NULL,
    normal_max_input_tokens INT NOT NULL,
    normal_max_output_tokens INT NOT NULL,
    reduced_max_input_tokens INT NOT NULL,
    reduced_max_output_tokens INT NOT NULL,
    max_paid_attempts_per_day INT,
    paid_fallback_enabled BOOLEAN DEFAULT FALSE NOT NULL,
    is_enabled BOOLEAN DEFAULT TRUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT ai_cost_policy_operations_pkey PRIMARY KEY (id),
    CONSTRAINT ai_cost_policy_operations_policy_id_operation_key UNIQUE (policy_id, operation),
    CONSTRAINT ai_cost_policy_operations_max_paid_attempts_per_day_check CHECK ((max_paid_attempts_per_day > 0)),
    CONSTRAINT ai_cost_policy_operations_normal_max_input_tokens_check CHECK ((normal_max_input_tokens > 0)),
    CONSTRAINT ai_cost_policy_operations_normal_max_output_tokens_check CHECK ((normal_max_output_tokens > 0)),
    CONSTRAINT ai_cost_policy_operations_operation_check CHECK ((operation::text = ANY (ARRAY['chat', 'outfit', 'summary', 'rewriter']))),
    CONSTRAINT ai_cost_policy_operations_reduced_max_input_tokens_check CHECK ((reduced_max_input_tokens > 0)),
    CONSTRAINT ai_cost_policy_operations_reduced_max_output_tokens_check CHECK ((reduced_max_output_tokens > 0))
);

CREATE TABLE IF NOT EXISTS aiexecutor.ai_model_configs (
    id UUID DEFAULT gen_random_uuid() NOT NULL,
    provider VARCHAR(50) NOT NULL,
    model VARCHAR(100) NOT NULL,
    input_usd_per_million NUMERIC(20,8) NOT NULL,
    output_usd_per_million NUMERIC(20,8) NOT NULL,
    currency VARCHAR(10) DEFAULT 'VND' NOT NULL,
    usd_to_vnd NUMERIC(20,4) NOT NULL,
    version VARCHAR(50) DEFAULT 'v1' NOT NULL,
    effective_from TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    effective_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    created_by UUID REFERENCES identity.users(id),
    CONSTRAINT ai_model_configs_pkey PRIMARY KEY (id),
    CONSTRAINT uq_provider_model_active UNIQUE (provider, model, effective_from)
);

CREATE TABLE IF NOT EXISTS aiexecutor.ai_policy_grants (
    id UUID DEFAULT gen_random_uuid() NOT NULL,
    user_id UUID NOT NULL,
    policy_id UUID NOT NULL,
    plan_id UUID NOT NULL,
    plan_code VARCHAR(100) NOT NULL,
    tier_rank INT NOT NULL,
    policy_snapshot JSONB NOT NULL,
    effective_from TIMESTAMPTZ NOT NULL,
    effective_to TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL,
    source_event_id UUID,
    source_deposit_transaction_id UUID,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT user_ai_policy_grants_pkey PRIMARY KEY (id),
    CONSTRAINT user_ai_policy_grants_policy_id_fkey FOREIGN KEY (policy_id) REFERENCES aiexecutor.ai_cost_policies(id) ON DELETE RESTRICT,
    CONSTRAINT user_ai_policy_grants_user_id_fkey FOREIGN KEY (user_id) REFERENCES identity.users(id) ON DELETE CASCADE,
    CONSTRAINT ai_policy_grants_effective_window_check CHECK ((effective_to IS NULL) OR (effective_to > effective_from)),
    CONSTRAINT user_ai_policy_grants_status_check CHECK ((status::text = ANY (ARRAY['active', 'future', 'closed'])))
);

CREATE TABLE IF NOT EXISTS aiexecutor.ai_usage_period_ledgers (
    id UUID DEFAULT gen_random_uuid() NOT NULL,
    grant_id UUID REFERENCES aiexecutor.ai_policy_grants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES identity.users(id) ON DELETE CASCADE,
    period_index INT NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    paid_input_tokens BIGINT DEFAULT 0 NOT NULL,
    paid_output_tokens BIGINT DEFAULT 0 NOT NULL,
    free_input_tokens BIGINT DEFAULT 0 NOT NULL,
    free_output_tokens BIGINT DEFAULT 0 NOT NULL,
    actual_cost_micro_vnd BIGINT DEFAULT 0 NOT NULL,
    reserved_cost_micro_vnd BIGINT DEFAULT 0 NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT ai_usage_period_ledgers_pkey PRIMARY KEY (id),
    CONSTRAINT ai_usage_period_ledgers_grant_id_period_index_key UNIQUE (grant_id, period_index),
    CONSTRAINT ai_usage_period_ledgers_period_index_check CHECK ((period_index >= 0))
);

CREATE TABLE IF NOT EXISTS aiexecutor.ai_usage_events (
    id UUID DEFAULT gen_random_uuid() NOT NULL,
    request_id UUID NOT NULL,
    ledger_id UUID REFERENCES aiexecutor.ai_usage_period_ledgers(id) ON DELETE CASCADE,
    user_id UUID REFERENCES identity.users(id) ON DELETE CASCADE,
    operation VARCHAR(30) NOT NULL,
    logical_route VARCHAR(50) NOT NULL,
    provider VARCHAR(50),
    model VARCHAR(150),
    input_usd_per_million NUMERIC(18,8),
    output_usd_per_million NUMERIC(18,8),
    usd_to_vnd NUMERIC(18,4),
    prompt_tokens BIGINT DEFAULT 0 NOT NULL,
    output_tokens BIGINT DEFAULT 0 NOT NULL,
    thinking_tokens BIGINT DEFAULT 0 NOT NULL,
    reserved_cost_micro_vnd BIGINT DEFAULT 0 NOT NULL,
    actual_cost_micro_vnd BIGINT DEFAULT 0 NOT NULL,
    estimated_max_cost_micro_vnd BIGINT DEFAULT 0 NOT NULL,
    status VARCHAR(30) NOT NULL,
    finish_reason VARCHAR(100),
    error_code VARCHAR(100),
    sent_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    unknown_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    estimated_prompt_tokens BIGINT,
    token_estimation_method VARCHAR(30),
    token_count_latency_ms BIGINT,
    pricing_config_id UUID REFERENCES aiexecutor.ai_model_configs(id),
    CONSTRAINT ai_usage_events_pkey PRIMARY KEY (id),
    CONSTRAINT ai_usage_events_request_id_key UNIQUE (request_id),
    CONSTRAINT ai_usage_events_estimated_prompt_tokens_check CHECK ((estimated_prompt_tokens IS NULL) OR (estimated_prompt_tokens >= 0)),
    CONSTRAINT ai_usage_events_status_check CHECK ((status::text = ANY (ARRAY['reserved', 'in_flight', 'confirmed', 'released', 'unknown_usage', 'expired_unverified']))),
    CONSTRAINT ai_usage_events_token_count_latency_ms_check CHECK ((token_count_latency_ms IS NULL) OR (token_count_latency_ms >= 0)),
    CONSTRAINT ai_usage_events_token_estimation_method_check CHECK ((token_estimation_method IS NULL) OR ((token_estimation_method)::text = ANY (ARRAY['local', 'provider_count'])))
);

CREATE TABLE IF NOT EXISTS aiexecutor.ai_cost_policy_audit_logs (
    id UUID DEFAULT gen_random_uuid() NOT NULL,
    actor_user_id UUID NOT NULL,
    changed_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    request_ip VARCHAR(64),
    policy_code VARCHAR(100) NOT NULL,
    operation VARCHAR(30) NOT NULL,
    field_name VARCHAR(60) NOT NULL,
    old_value VARCHAR(255),
    new_value VARCHAR(255),
    action VARCHAR(60) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    CONSTRAINT ai_cost_policy_audit_logs_pkey PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS ix_ai_policy_grants_source_event_id
    ON aiexecutor.ai_policy_grants (source_event_id);
CREATE INDEX IF NOT EXISTS ix_user_ai_policy_grants_active
    ON aiexecutor.ai_policy_grants (user_id, effective_from, effective_to);
CREATE INDEX IF NOT EXISTS ix_ai_usage_events_unknown
    ON aiexecutor.ai_usage_events (status, unknown_expires_at) WHERE status IN ('reserved', 'unknown_usage');
CREATE INDEX IF NOT EXISTS ix_ai_usage_events_user_day
    ON aiexecutor.ai_usage_events (user_id, created_at, status);
CREATE UNIQUE INDEX IF NOT EXISTS ux_ledgers_user_period_fallback
    ON aiexecutor.ai_usage_period_ledgers (user_id, period_index) WHERE grant_id IS NULL;
CREATE INDEX IF NOT EXISTS ix_ai_cost_policy_audit_logs_actor_user_id
    ON aiexecutor.ai_cost_policy_audit_logs (actor_user_id);
CREATE INDEX IF NOT EXISTS ix_ai_cost_policy_audit_logs_changed_at
    ON aiexecutor.ai_cost_policy_audit_logs (changed_at);

-- Default policies: recorded chain values, never overwrite applied databases.
INSERT INTO aiexecutor.ai_cost_policies (id, code, version, name, enforcement_mode, period_days, hard_cost_micro_vnd, compact_threshold_bps, free_route_threshold_bps, unknown_hold_minutes, max_unknown_paid_requests_per_day)
VALUES
    ('aa000000-0000-0000-0000-000000000001', 'free-default', 1, 'Free AI Policy', 'observe_only', 30, NULL, 8000, 9200, 1440, 0),
    ('aa000000-0000-0000-0000-000000000002', 'premium-default', 1, 'Premium AI Policy', 'strict', 30, 25000000000, 8000, 9200, 1440, 2)
ON CONFLICT (code, version) DO NOTHING;

-- Default operation caps at chain-final values (premium NULL = strict hard-cap
-- mode; free outfit 4 = FR-005d). Resolved by policy code so fresh UUIDs still
-- converge; never overwrite applied databases.
INSERT INTO aiexecutor.ai_cost_policy_operations (policy_id, operation, normal_route, reduced_route, free_route, normal_max_input_tokens, normal_max_output_tokens, reduced_max_input_tokens, reduced_max_output_tokens, max_paid_attempts_per_day)
SELECT p.id, v.operation, v.normal_route, v.reduced_route, v.free_route, v.nin, v.nout, v.rin, v.rout, v.maxpaid
FROM aiexecutor.ai_cost_policies p
JOIN (VALUES
    ('free-default', 'chat', 'paid', 'paid', 'free', 3000, 500, 2500, 400, 3),
    ('free-default', 'outfit', 'paid', 'paid', 'free', 5000, 300, 4000, 250, 4),
    ('free-default', 'summary', 'free', 'free', 'local', 3000, 250, 3000, 250, 5),
    ('free-default', 'rewriter', 'free', 'free', 'local', 1000, 250, 1000, 250, 5),
    ('premium-default', 'chat', 'paid', 'paid', 'free', 4000, 600, 3500, 500, NULL),
    ('premium-default', 'outfit', 'paid', 'paid', 'free', 6000, 400, 5000, 350, NULL),
    ('premium-default', 'summary', 'free', 'free', 'local', 3000, 250, 3000, 250, NULL),
    ('premium-default', 'rewriter', 'free', 'free', 'local', 1000, 250, 1000, 250, NULL)
) AS v(code, operation, normal_route, reduced_route, free_route, nin, nout, rin, rout, maxpaid)
    ON v.code = p.code
ON CONFLICT (policy_id, operation) DO NOTHING;

-- Model catalog: recorded chain rows. Guarded by provider+model (not by
-- effective_from, which defaults to NOW() and would never conflict).
INSERT INTO aiexecutor.ai_model_configs (provider, model, input_usd_per_million, output_usd_per_million, usd_to_vnd)
SELECT v.provider, v.model, v.inusd, v.outusd, v.tovnd
FROM (VALUES
    ('google', 'gemini-3.1-flash-lite', 0.25, 1.50, 27500),
    ('google', 'gemini-2.5-flash-lite', 0.10, 0.40, 27500),
    ('google', 'gemini-embedding-2', 0.20, 0.20, 27500),
    ('google', 'gemini-embedding-001', 0.15, 0.15, 27500),
    ('google', 'gemma-4-31b-it', 0, 0, 27500),
    ('google', 'gemma-4-26b-a4b-it', 0, 0, 27500)
) AS v(provider, model, inusd, outusd, tovnd)
WHERE NOT EXISTS (
    SELECT 1 FROM aiexecutor.ai_model_configs m
    WHERE m.provider = v.provider AND m.model = v.model
);

-- One-time orphan rename (FR-004): runs only when the old name exists and the
-- new one does not. Fresh databases already carry the new name (skipped);
-- converged databases skip on every later boot.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint c JOIN pg_class t ON t.oid = c.conrelid
        WHERE t.relname = 'ai_policy_grants' AND c.conname = 'user_ai_policy_grants_check'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint c JOIN pg_class t ON t.oid = c.conrelid
        WHERE t.relname = 'ai_policy_grants' AND c.conname = 'ai_policy_grants_effective_window_check'
    ) THEN
        ALTER TABLE aiexecutor.ai_policy_grants
            RENAME CONSTRAINT user_ai_policy_grants_check TO ai_policy_grants_effective_window_check;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Exact reverse of the new Up: drop owned objects in dependency order.
DROP TABLE IF EXISTS aiexecutor.ai_cost_policy_audit_logs;
DROP TABLE IF EXISTS aiexecutor.ai_usage_events;
DROP TABLE IF EXISTS aiexecutor.ai_usage_period_ledgers;
DROP TABLE IF EXISTS aiexecutor.ai_policy_grants;
DROP TABLE IF EXISTS aiexecutor.ai_model_configs;
DROP TABLE IF EXISTS aiexecutor.ai_cost_policy_operations;
DROP TABLE IF EXISTS aiexecutor.ai_cost_policies;
DROP SCHEMA IF EXISTS aiexecutor;
-- +goose StatementEnd
