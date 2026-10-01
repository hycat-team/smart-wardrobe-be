-- ============================================================================
-- UPDATE SCRIPT: RAISE THE PREMIUM AI HARD COST CAP (024, FR-002)
-- ============================================================================
-- Runs in one transaction. Idempotent by construction (SC-012): the UPDATE is
-- guarded with `IS DISTINCT FROM`, so a second and third run affect 0 rows and
-- leave the stored value untouched.
--
-- Scope: aiexecutor.ai_cost_policies row with code = 'premium-default' ONLY.
-- The free policy has hard_cost_micro_vnd = NULL on purpose (no hard cap) and
-- MUST NOT be given one.
--
-- The UPDATE also bumps `version` so the change is traceable against the
-- policy version carried by grant snapshots.
--
-- Rollback: set hard_cost_micro_vnd back to 25000000000 for 'premium-default'
-- and set version back to its previous value.
-- ============================================================================

BEGIN;

-- 1. Report the value before the write (transaction-local audit trail).
\echo '--- before ---'
SELECT code, version, hard_cost_micro_vnd
FROM aiexecutor.ai_cost_policies
WHERE code = 'premium-default'
FOR UPDATE;

-- 2. Guarded update: fires only when the stored value differs from the target.
UPDATE aiexecutor.ai_cost_policies
SET hard_cost_micro_vnd = 30000000000,
    version              = version + 1
WHERE code = 'premium-default'
  AND hard_cost_micro_vnd IS DISTINCT FROM 30000000000;

-- 3. Report the value after the write plus how many rows the guard let through.
--    Expected on a first run: 1. On a re-run: 0 (SC-012).
\echo '--- after ---'
SELECT
    code                                    AS policy_code,
    version,
    hard_cost_micro_vnd                    AS hard_cap_micro_vnd,
    ROUND(hard_cost_micro_vnd / 1000000.0) AS hard_cap_vnd,
    period_days
FROM aiexecutor.ai_cost_policies
WHERE code = 'premium-default';

\echo '--- rows affected by the guarded update (1 on first run, 0 afterwards) ---'

-- 4. Confirm nothing else moved.
\echo '--- policies that must be unchanged ---'
SELECT
    code             AS policy_code,
    hard_cost_micro_vnd,
    (hard_cost_micro_vnd IS NULL) AS has_no_hard_cap
FROM aiexecutor.ai_cost_policies
WHERE code <> 'premium-default'
ORDER BY code;

COMMIT;
