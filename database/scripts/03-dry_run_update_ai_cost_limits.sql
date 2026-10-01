-- ============================================================================
-- DRY-RUN SCRIPT: PRINT CURRENT AI COST POLICY HARD CAPS (024, FR-001 / FR-003)
-- ============================================================================
-- READ-ONLY. This script MUST NOT write anything. It is safe to run any number
-- of times and every run must produce byte-identical output (FR-003).
--
-- Why this script exists
-- ---------------------
-- The Premium hard cap is raised from 25,000,000,000 to 30,000,000,000
-- microVND so the cycle budget covers the committed 30 outfit / 10 chat daily
-- allowance. The cap is read from LIVE configuration, never from
-- ai_policy_grants.policy_snapshot (FR-029b / research.md R13), so a single
-- UPDATE on aiexecutor.ai_cost_policies takes effect immediately for every
-- account that currently holds a grant row.
--
-- Units
-- -----
-- hard_cost_micro_vnd is an integer amount of Vietnamese Dong multiplied by
-- 1,000,000 ("microVND"). It is a bigint because VND has no minor unit in
-- practice, so storing a small currency unit would waste half the range.
--   30,000,000,000 microVND = 30,000,000,000 / 1,000,000 = 30,000 VND
--   25,000,000,000 microVND = 25,000,000,000 / 1,000,000 = 25,000 VND (previous)
-- To convert back to VND for display: hard_cost_micro_vnd / 1000000.
-- period_days is the length of the cycle the cap applies to (30 days today).
--
-- NULL semantics
-- ---------------
-- NULL means "this policy has no hard cap at all" and is a deliberate value
-- for the free policy, NOT missing data. A NULL is therefore never "fixed" by
-- this feature.
-- ============================================================================

\echo '=== 1. Current hard cap per AI cost policy (024 / FR-001) ==='

SELECT
    code                                       AS policy_code,
    name                                       AS policy_name,
    enforcement_mode,
    period_days,
    hard_cost_micro_vnd                        AS hard_cap_micro_vnd,
    ROUND(hard_cost_micro_vnd / 1000000.0, 0) AS hard_cap_vnd,
    (hard_cost_micro_vnd IS NULL)             AS has_no_hard_cap,
    is_active
FROM aiexecutor.ai_cost_policies
ORDER BY code;

\echo ''
\echo '=== 2. Rows this script WOULD change (expected: exactly 1) ==='
\echo '--- target: premium-default, current 25000000000 -> new 30000000000 ---'

SELECT
    code             AS policy_code,
    hard_cost_micro_vnd AS current_micro_vnd,
    30000000000::bigint AS new_micro_vnd,
    30000 - ROUND(hard_cost_micro_vnd / 1000000.0, 0) AS delta_vnd
FROM aiexecutor.ai_cost_policies
WHERE code = 'premium-default'
  AND hard_cost_micro_vnd IS DISTINCT FROM 30000000000;

\echo ''
\echo '=== 3. Target policy after the update, for reference ==='

SELECT
    code            AS policy_code,
    30000000000::bigint AS new_hard_cap_micro_vnd,
    30              AS new_hard_cap_vnd,
    period_days,
    (30000000000::numeric / period_days / 1000000) AS cap_divided_by_period_days_vnd
FROM aiexecutor.ai_cost_policies
WHERE code = 'premium-default';

\echo '--- NOTE: the column above is the cap simply divided by period_days, NOT'
\echo '--- the expected daily spend. Real consumption is far lower (roughly'
\echo '--- 2,000 VND/day at the 30 outfit + 10 chat target), which is why the'
\echo '--- 30,000 VND budget lasts 12-15 days of full usage (FR-005c). ---'

\echo ''
\echo '=== 4. Policies that must NOT be touched ==='

SELECT
    code             AS policy_code,
    hard_cost_micro_vnd,
    (hard_cost_micro_vnd IS NULL) AS has_no_hard_cap,
    'must stay unchanged'         AS expectation
FROM aiexecutor.ai_cost_policies
WHERE code <> 'premium-default'
ORDER BY code;

\echo ''
\echo '=== 5. Gate-1 vs Gate-2 cross-check reference (read-only) ==='
\echo '--- Gate 1 = what the customer sees (subscription_plans); Gate 2 = what'
\echo '--- the AI engine enforces (ai_cost_policy_operations). Read from the'
\echo '--- plan->policy foreign key subscription_plans.ai_cost_policy_id. ---'

SELECT
    sp.slug                                             AS plan_slug,
    p.code                                              AS policy_code,
    p.enforcement_mode,
    o.operation,
    CASE o.operation
        WHEN 'outfit' THEN sp.ai_outfit_daily_quota
        WHEN 'chat'   THEN sp.ai_chat_daily_quota
    END                                                 AS gate1_customer_visible,
    o.max_paid_attempts_per_day                         AS gate2_ai_enforced,
    (p.enforcement_mode = 'strict')                     AS gate2_is_dead_config
FROM subscription.subscription_plans sp
JOIN aiexecutor.ai_cost_policies p          ON p.id = sp.ai_cost_policy_id
JOIN aiexecutor.ai_cost_policy_operations o ON o.policy_id = p.id
WHERE o.operation IN ('chat', 'outfit')
ORDER BY sp.slug, o.operation;
