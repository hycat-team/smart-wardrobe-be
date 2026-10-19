-- +goose Up
-- +goose StatementBegin
-- 030 DDL ownership handover (US1): the five aiexecutor tables, their seed
-- rows and the grant backfill used to live here. They now live in the AI
-- service baseline (services/ai/database/migrations). This file keeps only
-- the Account-owned subscription_plans column. The REFERENCES clause is
-- dropped: the cross-module FK was cut in 20260801000000 and the AI tables do
-- not exist yet when the Account tree replays on a fresh database.

ALTER TABLE subscription_plans ADD COLUMN ai_cost_policy_id UUID;
UPDATE subscription_plans SET ai_cost_policy_id=CASE WHEN plan_kind=0 THEN 'aa000000-0000-0000-0000-000000000001'::uuid ELSE 'aa000000-0000-0000-0000-000000000002'::uuid END;
ALTER TABLE subscription_plans ALTER COLUMN ai_cost_policy_id SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE subscription_plans DROP COLUMN IF EXISTS ai_cost_policy_id;
-- +goose StatementEnd
