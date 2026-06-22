-- +goose Up
-- +goose StatementBegin
CREATE TABLE ai_cost_policies (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), code VARCHAR(100) NOT NULL, version BIGINT NOT NULL, name VARCHAR(150) NOT NULL,
 enforcement_mode VARCHAR(30) NOT NULL CHECK (enforcement_mode IN ('STRICT','OBSERVE_ONLY','FREE_ONLY')),
 period_days INT NOT NULL CHECK (period_days > 0), hard_cost_micro_vnd BIGINT,
 compact_threshold_bps INT NOT NULL DEFAULT 8000 CHECK (compact_threshold_bps BETWEEN 0 AND 10000),
 free_route_threshold_bps INT NOT NULL DEFAULT 9200 CHECK (free_route_threshold_bps BETWEEN 0 AND 10000),
 unknown_hold_minutes INT NOT NULL DEFAULT 1440 CHECK (unknown_hold_minutes > 0),
 max_unknown_paid_requests_per_day INT NOT NULL DEFAULT 2 CHECK (max_unknown_paid_requests_per_day >= 0),
 is_active BOOLEAN NOT NULL DEFAULT TRUE, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE(code, version), CHECK (enforcement_mode <> 'STRICT' OR hard_cost_micro_vnd IS NOT NULL), CHECK (compact_threshold_bps <= free_route_threshold_bps)
);

CREATE TABLE ai_cost_policy_operations (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), policy_id UUID NOT NULL REFERENCES ai_cost_policies(id) ON DELETE CASCADE,
 operation VARCHAR(30) NOT NULL CHECK (operation IN ('chat','outfit','summary','rewriter')),
 normal_route VARCHAR(50) NOT NULL, reduced_route VARCHAR(50) NOT NULL, free_route VARCHAR(50) NOT NULL,
 normal_max_input_tokens INT NOT NULL CHECK (normal_max_input_tokens > 0), normal_max_output_tokens INT NOT NULL CHECK (normal_max_output_tokens > 0),
 reduced_max_input_tokens INT NOT NULL CHECK (reduced_max_input_tokens > 0), reduced_max_output_tokens INT NOT NULL CHECK (reduced_max_output_tokens > 0),
 max_paid_attempts_per_day INT NOT NULL CHECK (max_paid_attempts_per_day > 0), paid_fallback_enabled BOOLEAN NOT NULL DEFAULT FALSE,
 is_enabled BOOLEAN NOT NULL DEFAULT TRUE, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE(policy_id, operation)
);

ALTER TABLE subscription_plans ADD COLUMN ai_cost_policy_id UUID REFERENCES ai_cost_policies(id) ON DELETE RESTRICT;
INSERT INTO ai_cost_policies (id,code,version,name,enforcement_mode,period_days,hard_cost_micro_vnd,compact_threshold_bps,free_route_threshold_bps,unknown_hold_minutes,max_unknown_paid_requests_per_day) VALUES
 ('aa000000-0000-0000-0000-000000000001','free-default',1,'Free AI Policy','OBSERVE_ONLY',30,NULL,8000,9200,1440,0),
 ('aa000000-0000-0000-0000-000000000002','premium-default',1,'Premium AI Policy','STRICT',30,25000000000,8000,9200,1440,2);
INSERT INTO ai_cost_policy_operations (policy_id,operation,normal_route,reduced_route,free_route,normal_max_input_tokens,normal_max_output_tokens,reduced_max_input_tokens,reduced_max_output_tokens,max_paid_attempts_per_day) VALUES
 ('aa000000-0000-0000-0000-000000000001','chat','paid','paid','free',3000,1000,2500,800,5),
 ('aa000000-0000-0000-0000-000000000001','outfit','paid','paid','free',5000,400,4000,350,5),
 ('aa000000-0000-0000-0000-000000000001','summary','free','free','local',3000,250,3000,250,5),
 ('aa000000-0000-0000-0000-000000000001','rewriter','free','free','local',1000,250,1000,250,5),
 ('aa000000-0000-0000-0000-000000000002','chat','paid','paid','free',4000,1000,4000,1000,20),
 ('aa000000-0000-0000-0000-000000000002','outfit','paid','paid','free',7000,400,7000,400,15),
 ('aa000000-0000-0000-0000-000000000002','summary','free','free','local',3000,250,3000,250,20),
 ('aa000000-0000-0000-0000-000000000002','rewriter','free','free','local',1000,250,1000,250,15);
UPDATE subscription_plans SET ai_cost_policy_id=CASE WHEN plan_kind=0 THEN 'aa000000-0000-0000-0000-000000000001'::uuid ELSE 'aa000000-0000-0000-0000-000000000002'::uuid END;
ALTER TABLE subscription_plans ALTER COLUMN ai_cost_policy_id SET NOT NULL;

CREATE TABLE user_ai_policy_grants (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 policy_id UUID NOT NULL REFERENCES ai_cost_policies(id) ON DELETE RESTRICT, plan_id UUID NOT NULL REFERENCES subscription_plans(id) ON DELETE RESTRICT,
 plan_code VARCHAR(100) NOT NULL, tier_rank INT NOT NULL, policy_snapshot JSONB NOT NULL, effective_from TIMESTAMPTZ NOT NULL,
 effective_to TIMESTAMPTZ, status VARCHAR(20) NOT NULL CHECK (status IN ('ACTIVE','FUTURE','CLOSED')),
 source_event_id UUID REFERENCES user_subscription_events(id) ON DELETE SET NULL, source_deposit_transaction_id UUID REFERENCES deposit_transactions(id) ON DELETE SET NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), CHECK (effective_to IS NULL OR effective_to > effective_from)
);
CREATE INDEX ix_user_ai_policy_grants_active ON user_ai_policy_grants(user_id,effective_from,effective_to);

CREATE TABLE ai_usage_period_ledgers (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), grant_id UUID NOT NULL REFERENCES user_ai_policy_grants(id) ON DELETE CASCADE,
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, period_index INT NOT NULL CHECK (period_index >= 0),
 period_start TIMESTAMPTZ NOT NULL, period_end TIMESTAMPTZ NOT NULL, paid_input_tokens BIGINT NOT NULL DEFAULT 0,
 paid_output_tokens BIGINT NOT NULL DEFAULT 0, free_input_tokens BIGINT NOT NULL DEFAULT 0, free_output_tokens BIGINT NOT NULL DEFAULT 0,
 actual_cost_micro_vnd BIGINT NOT NULL DEFAULT 0, reserved_cost_micro_vnd BIGINT NOT NULL DEFAULT 0,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE(grant_id,period_index)
);

CREATE TABLE ai_usage_events (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), request_id UUID NOT NULL UNIQUE,
 ledger_id UUID NOT NULL REFERENCES ai_usage_period_ledgers(id) ON DELETE CASCADE, user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 operation VARCHAR(30) NOT NULL, logical_route VARCHAR(50) NOT NULL, provider VARCHAR(50), model VARCHAR(150), pricing_version VARCHAR(100),
 input_usd_per_million NUMERIC(18,8), output_usd_per_million NUMERIC(18,8), usd_to_vnd NUMERIC(18,4),
 prompt_tokens BIGINT NOT NULL DEFAULT 0, output_tokens BIGINT NOT NULL DEFAULT 0, thinking_tokens BIGINT NOT NULL DEFAULT 0,
 reserved_cost_micro_vnd BIGINT NOT NULL DEFAULT 0, actual_cost_micro_vnd BIGINT NOT NULL DEFAULT 0, estimated_max_cost_micro_vnd BIGINT NOT NULL DEFAULT 0,
 status VARCHAR(30) NOT NULL CHECK (status IN ('RESERVED','IN_FLIGHT','CONFIRMED','RELEASED','UNKNOWN_USAGE','EXPIRED_UNVERIFIED')),
 finish_reason VARCHAR(100), error_code VARCHAR(100), sent_at TIMESTAMPTZ, completed_at TIMESTAMPTZ, unknown_expires_at TIMESTAMPTZ,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ix_ai_usage_events_unknown ON ai_usage_events(status,unknown_expires_at) WHERE status IN ('RESERVED','UNKNOWN_USAGE');
CREATE INDEX ix_ai_usage_events_user_day ON ai_usage_events(user_id,created_at,status);

INSERT INTO user_ai_policy_grants (user_id,policy_id,plan_id,plan_code,tier_rank,policy_snapshot,effective_from,effective_to,status)
SELECT us.user_id,sp.ai_cost_policy_id,sp.id,sp.slug,sp.tier_rank,
 jsonb_build_object('policyId',p.id,'code',p.code,'version',p.version,'mode',p.enforcement_mode,'periodDays',p.period_days,
 'hardCostMicroVnd',p.hard_cost_micro_vnd,'compactThresholdBps',p.compact_threshold_bps,'freeRouteThresholdBps',p.free_route_threshold_bps,
 'unknownHoldMinutes',p.unknown_hold_minutes,'maxUnknownPaidRequestsPerDay',p.max_unknown_paid_requests_per_day),
 us.started_at,us.expires_at,'ACTIVE'
FROM user_subscriptions us JOIN subscription_plans sp ON sp.id=us.subscription_plan_id JOIN ai_cost_policies p ON p.id=sp.ai_cost_policy_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ai_usage_events;
DROP TABLE IF EXISTS ai_usage_period_ledgers;
DROP TABLE IF EXISTS user_ai_policy_grants;
ALTER TABLE subscription_plans DROP COLUMN IF EXISTS ai_cost_policy_id;
DROP TABLE IF EXISTS ai_cost_policy_operations;
DROP TABLE IF EXISTS ai_cost_policies;
-- +goose StatementEnd
