-- ============================================================================
-- DRY-RUN: measure what the 024 GĐ3b cleanup would change (FR-092, FR-093)
-- ============================================================================
-- READ-ONLY. Safe to run any number of times; every run prints the same four
-- numbers for an unchanged database.
--
-- Run this IMMEDIATELY before 06-cleanup_policy_grants.sql and compare the
-- numbers it prints to what 06 then reports. Do not compare against figures
-- recorded in documentation: signals arrive continuously, so a hard-coded
-- baseline will not match and you will not know whether the data moved or the
-- documentation is stale.
--
-- The one invariant that must hold after the cleanup is the LAST number: the
-- count of grants that are genuinely still in force. Cleanup must never take an
-- entitlement away from anyone. The other three only have to reach zero.
-- ============================================================================

\echo ''
\echo '=== 1. Empty policy snapshots ==='
\echo '--- The reader requires a two-part snapshot with all four operations.'
\echo '--- An empty one cannot satisfy the entitlement read, so the account'
\echo '--- silently falls back with no limits at all. ---'
SELECT count(*) AS empty_snapshots
FROM aiexecutor.ai_policy_grants
WHERE policy_snapshot = '{}'::jsonb;

\echo ''
\echo '=== 2. Closed grants whose window has ended ==='
\echo '--- Permanently dead. These are deleted, not closed again. ---'
SELECT count(*) AS closed_and_expired
FROM aiexecutor.ai_policy_grants
WHERE status = 'closed'
  AND (effective_to IS NULL OR effective_to <= now());

\echo ''
\echo '=== 3. Active or future grants whose window has ended ==='
\echo '--- A real defect: the row claims to be live but the paid period is over.'
\echo '--- These are closed so they stop shadowing the current plan. ---'
SELECT count(*) AS live_status_but_expired
FROM aiexecutor.ai_policy_grants
WHERE status IN ('active', 'future')
  AND effective_to IS NOT NULL
  AND effective_to <= now();

\echo ''
\echo '=== 4. Grants that are genuinely still in force ==='
\echo '--- THIS NUMBER MUST NOT CHANGE. If it drops, accounts have lost their'
\echo '--- entitlements; roll the database back immediately. ---'
SELECT count(*) AS still_in_force
FROM aiexecutor.ai_policy_grants
WHERE status IN ('active', 'future')
  AND (effective_to IS NULL OR effective_to > now());

\echo ''
\echo '=== 5. Detail of the grants that are still in force ==='
\echo '--- Read this before running the cleanup: it is the list of entitlements'
\echo '--- that must survive. ---'
SELECT user_id, plan_code, status, effective_from, effective_to
FROM aiexecutor.ai_policy_grants
WHERE status IN ('active', 'future')
  AND (effective_to IS NULL OR effective_to > now())
ORDER BY user_id, effective_from;

\echo ''
\echo '=== 6. Accounts holding more than one grant in force at once ==='
\echo '--- Expected: no rows. Each row is an account being granted twice. ---'
SELECT user_id, count(*) AS concurrent_grants
FROM aiexecutor.ai_policy_grants
WHERE status = 'active'
  AND effective_from <= now()
  AND (effective_to IS NULL OR effective_to > now())
GROUP BY user_id
HAVING count(*) > 1;
