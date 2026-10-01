-- ============================================================================
-- DRY-RUN: report the subscription plan-change history rows that a bulk
-- `is_auto_renew_enabled = false` update left unrecorded (024 FR-012, FR-015)
-- ============================================================================
-- READ-ONLY. This script MUST NOT write anything. It is safe to run any number
-- of times and every run must produce byte-identical output for an unchanged
-- database (FR-003).
--
-- Why this script exists
-- ---------------------
-- Toggling auto-renew through the application (`PUT /subscriptions/me/auto-renew`
-- -> `SetAutoRenewStatus`) writes a `user_subscription_events` row and an outbox
-- record in one transaction, and increments `version`. That is what FR-012
-- requires and what every other plan change does.
--
-- Turning auto-renew off with a single UPDATE statement across every row does
-- none of that. The rows are correct; the TRAIL is not. Two things are lost:
--
--   1. There is no `AUTO_RENEW_UPDATED` row, so there is no way to answer "who
--      turned auto-renew off for this account, and when". FR-012.
--   2. `version` did not increment. It is part of the `RESTORE_GRANT:{user}:{version}`
--      event key and of the optimistic-concurrency check, so a `version` that
--      does not count a change makes both of those slightly wrong.
--
-- This script reports what the paired apply script (10-) WOULD write. Run it
-- first and keep its output, exactly as FR-001 requires for every configuration
-- script in this directory.
--
-- ⚠️ The number this prints is the count of rows the apply script will INSERT.
-- On a second run of the apply script it is 0, because the history rows now
-- exist. That is the idempotency contract: safe to re-run, never double-writes.
-- ============================================================================

\set ON_ERROR_STOP on

-- ⚠️ THE SCOPE IS ONE INSTANT, ON PURPOSE.
--
-- Selecting "every subscription with auto-renew off and no history row" looks
-- equivalent and is NOT. Two accounts in this database have auto-renew off
-- because nobody ever turned it on - default provisioning and a downgrade both
-- leave it false - and a history row for either would be a record of a change
-- nobody made. A fabricated audit entry is worse than a missing one, because a
-- missing one is visibly missing.
--
-- The bulk UPDATE stamped every row it touched with one identical `updated_at`,
-- so that instant is the precise identifier of what needs backfilling.
\if :{?bulk_update_at}
\else
  \set bulk_update_at '''2026-09-30 09:41:48.638213+00'''
\endif

\echo ''
\echo '=== 1. Rows missing an AUTO_RENEW_UPDATED history row ==='
\echo '--- These are exactly the rows the paired apply script would backfill. ---'
\echo '--- Expected 0 after the apply script has been run. ---'
SELECT
    s.user_id,
    s.current_plan_code,
    s.version                     AS current_version,
    s.version + 1                 AS version_after_backfill,
    s.updated_at                  AS auto_renew_disabled_at,
    CASE WHEN s.expires_at IS NULL THEN NULL
         ELSE to_char(s.expires_at, 'YYYY-MM-DD') END AS expires_at
FROM subscription.user_subscriptions s
WHERE s.is_auto_renew_enabled = false
  AND s.updated_at = :'bulk_update_at'
  AND NOT EXISTS (
        SELECT 1
          FROM subscription.user_subscription_events e
         WHERE e.user_id = s.user_id
           AND e.event_type = 'AUTO_RENEW_UPDATED'
      )
ORDER BY s.user_id;

\echo ''
\echo '--- How many rows would be written ---'
SELECT count(*) AS rows_to_backfill
FROM subscription.user_subscriptions s
WHERE s.is_auto_renew_enabled = false
  AND s.updated_at = :'bulk_update_at'
  AND NOT EXISTS (
        SELECT 1
          FROM subscription.user_subscription_events e
         WHERE e.user_id = s.user_id
           AND e.event_type = 'AUTO_RENEW_UPDATED'
      );

\echo ''
\echo '=== 2. Event keys the apply script will use ==='
\echo '--- Shape matches the application: STATE_CHANGE:AUTO_RENEW:{userId}:{version}. ---'
\echo '--- The version in the key is the POST-increment one, because the ---'
\echo '--- application increments before it builds the key. ---'
SELECT
    'STATE_CHANGE:AUTO_RENEW:' || s.user_id || ':' || (s.version + 1) AS event_key_to_insert
FROM subscription.user_subscriptions s
WHERE s.is_auto_renew_enabled = false
  AND s.updated_at = :'bulk_update_at'
  AND NOT EXISTS (
        SELECT 1
          FROM subscription.user_subscription_events e
         WHERE e.user_id = s.user_id
           AND e.event_type = 'AUTO_RENEW_UPDATED'
      )
ORDER BY 1;

\echo ''
\echo '=== 3. Collisions that would make an insert silently skipped ==='
\echo '--- Expected 0 rows. A collision means the key already exists, which ---'
\echo '--- ON CONFLICT DO NOTHING would swallow - and then the version bump ---'
\echo '--- would still happen, leaving version ahead of the history. ---'
SELECT 'STATE_CHANGE:AUTO_RENEW:' || s.user_id || ':' || (s.version + 1) AS colliding_key
FROM subscription.user_subscriptions s
JOIN subscription.user_subscription_events e
  ON e.event_key = 'STATE_CHANGE:AUTO_RENEW:' || s.user_id || ':' || (s.version + 1)
WHERE s.is_auto_renew_enabled = false
  AND s.updated_at = :'bulk_update_at'
  AND e.event_type <> 'AUTO_RENEW_UPDATED';

\echo ''
\echo '=== 4. Plans whose slug is missing from the catalogue ==='
\echo '--- Expected 0 rows. The apply script needs the plan id and its AI cost ---'
\echo '--- policy to build a payload; without them it cannot write the outbox ---'
\echo '--- row, and FR-015 would be only half satisfied. ---'
SELECT s.user_id, s.current_plan_code
FROM subscription.user_subscriptions s
LEFT JOIN subscription.subscription_plans sp ON sp.slug = s.current_plan_code
WHERE s.is_auto_renew_enabled = false
  AND s.updated_at = :'bulk_update_at'
  AND sp.id IS NULL
  AND NOT EXISTS (
        SELECT 1
          FROM subscription.user_subscription_events e
         WHERE e.user_id = s.user_id
           AND e.event_type = 'AUTO_RENEW_UPDATED'
      );

\echo ''
\echo '=== 5. Rows with auto-renew still ON but no history row ==='
\echo '--- Expected 0. Not touched by this script: an account whose auto-renew ---'
\echo '--- was never toggled has nothing to backfill, and inventing a history ---'
\echo '--- row for a change nobody made would be worse than the gap. ---'
SELECT count(*) AS enabled_without_history
FROM subscription.user_subscriptions s
WHERE s.is_auto_renew_enabled = true
  AND NOT EXISTS (
        SELECT 1
          FROM subscription.user_subscription_events e
         WHERE e.user_id = s.user_id
           AND e.event_type = 'AUTO_RENEW_UPDATED'
      );

\echo ''
\echo 'Run 10-backfill_auto_renew_history.sql to apply. Section 1 and 2 above'
\echo 'must both be empty afterwards.'
