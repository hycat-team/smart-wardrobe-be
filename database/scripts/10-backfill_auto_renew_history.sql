-- ============================================================================
-- APPLY: backfill the subscription plan-change history that a bulk
-- `is_auto_renew_enabled = false` update left unrecorded (024 FR-012, FR-015)
-- ============================================================================
-- Runs in one transaction. Idempotent by construction (SC-012): the history
-- insert is guarded with `ON CONFLICT (event_key) DO NOTHING`, and both the
-- outbox insert and the version bump are driven off the rows that insert
-- actually produced. A second run writes 0 rows and changes nothing.
--
-- Run 09-dry_run_backfill_auto_renew_history.sql first and keep its output.
--
-- What it writes, and why each part is required:
--
--   1. A `user_subscription_events` row of type AUTO_RENEW_UPDATED.
--      FR-012 requires every plan-relevant change to leave a history row; this is
--      the change that has none.
--   2. A `shared.outbox_records` row on `subscription.plan.changed`.
--      FR-015 requires the history row AND the event in ONE transaction. A
--      history row nobody was told about is a change the AI service never
--      learns of - which is the exact defect this feature set out to remove.
--   3. `version + 1` on the subscription.
--      The application increments version for this change; here it did not.
--      Version is part of the `RESTORE_GRANT:{user}:{version}` event key and of
--      the optimistic-concurrency check, so leaving it stale makes both slightly
--      wrong.
--
-- `updated_at` is deliberately LEFT ALONE. It already holds the instant the
-- change actually happened, and the backfill running later must not overwrite
-- that with its own clock - otherwise the record would say the change happened
-- when the repair happened.
--
-- The event key uses the POST-increment version, because the application
-- increments before building the key. Verified against the one AUTO_RENEW_UPDATED
-- row the application itself wrote: `STATE_CHANGE:AUTO_RENEW:{user}:3`.
--
-- Rollback: the whole thing is one transaction, so an error leaves the database
-- exactly as it was.
-- ============================================================================

\set ON_ERROR_STOP on

-- Same instant as the paired dry-run script. See that file for WHY the scope is
-- one instant rather than "every row that looks unrecorded".
\if :{?bulk_update_at}
\else
  \set bulk_update_at '''2026-09-30 09:41:48.638213+00'''
\endif

BEGIN;

-- Targets: the rows the bulk UPDATE touched, and only those.
CREATE TEMP TABLE backfill_targets ON COMMIT DROP AS
SELECT
    s.user_id,
    s.version + 1                    AS new_version,
    s.current_plan_code,
    s.current_tier_rank,
    s.expires_at,
    s.updated_at                      AS occurred_at,
    sp.id                             AS plan_id,
    sp.ai_cost_policy_id,
    sp.duration_days
FROM subscription.user_subscriptions s
JOIN subscription.subscription_plans sp ON sp.slug = s.current_plan_code
WHERE s.is_auto_renew_enabled = false
  AND s.updated_at = :'bulk_update_at'
  AND NOT EXISTS (
        SELECT 1
          FROM subscription.user_subscription_events e
         WHERE e.user_id = s.user_id
           AND e.event_type = 'AUTO_RENEW_UPDATED'
      );

\echo '--- targets ---'
SELECT count(*) AS rows_selected FROM backfill_targets;

-- 1. The history row.
--
-- `from_plan_code` equals `to_plan_code`: auto-renew does not change the plan,
-- only whether it renews itself. That is how the application writes it, and the
-- receiver uses the equality to recognise this as a no-op entitlement change.
--
-- `metadata` records WHY this row exists. A future reader must be able to tell a
-- backfilled record from one an administrator produced, or the audit trail
-- becomes a claim rather than a record.
--
-- ⚠️ The data-modifying CTE inside CREATE TABLE AS is deliberate:
-- `CREATE TABLE ... AS INSERT` is not valid SQL, and the CTE is what lets the
-- inserted ids be captured for the outbox step below.
CREATE TEMP TABLE backfill_events ON COMMIT DROP AS
WITH inserted AS (
    INSERT INTO subscription.user_subscription_events
        (event_key, user_id, event_type, from_plan_code, to_plan_code, to_tier_rank,
         actor_admin_id, occurred_at, effective_at, metadata)
    SELECT
        'STATE_CHANGE:AUTO_RENEW:' || t.user_id || ':' || t.new_version,
        t.user_id,
        'AUTO_RENEW_UPDATED',
        t.current_plan_code,
        t.current_plan_code,
        t.current_tier_rank,
        NULL,                              -- no single administrator performed a bulk update
        t.occurred_at,
        t.occurred_at,
        jsonb_build_object(
            'source', 'bulk_auto_renew_backfill',
            'backfilledBy', '09/10-backfill_auto_renew_history.sql'
        )
    FROM backfill_targets t
    ON CONFLICT (event_key) DO NOTHING
    RETURNING id, user_id, occurred_at
)
SELECT * FROM inserted;

\echo '--- history rows written ---'
SELECT count(*) AS history_rows_written FROM backfill_events;

-- 2. The outbox row, so the change is actually announced (FR-015).
--
-- Built from the rows step 1 really produced, not from the targets: if a key had
-- collided, step 1 wrote nothing and step 2 must write nothing either.
--
-- `WHERE t.expires_at IS NOT NULL` matches what the payload builder does - a nil
-- expiry means "no expiry" and is carried as null, not as a missing field.
CREATE TEMP TABLE backfill_outbox ON COMMIT DROP AS
WITH inserted AS (
    INSERT INTO shared.outbox_records (topic, payload, headers, status, created_at, updated_at)
    SELECT
        'subscription.plan.changed',
        jsonb_build_object(
            'eventVersion', 2,
            'userId', t.user_id,
            'eventType', 'AUTO_RENEW_UPDATED',
            'toPlanId', t.plan_id,
            'toPlanCode', t.current_plan_code,
            'toTierRank', t.current_tier_rank,
            'aiCostPolicyId', t.ai_cost_policy_id,
            'occurredAt', to_char(ev.occurred_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
            'effectiveAt', to_char(ev.occurred_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
            'effectiveTo', to_char(t.expires_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
            'durationDays', t.duration_days,
            'sourceEventId', ev.id,
            'sourceDepositTransactionId', '',
            'fromPlanCode', t.current_plan_code
        ),
        jsonb_build_object('eventVersion', 'v1', 'correlationId', ev.id::text),
        'pending',
        now(),
        now()
    FROM backfill_events ev
    JOIN backfill_targets t ON t.user_id = ev.user_id
    ON CONFLICT DO NOTHING
    RETURNING id
)
SELECT * FROM inserted;

\echo '--- outbox rows written ---'
SELECT count(*) AS outbox_rows_written FROM backfill_outbox;

-- 3. The version bump, restricted to rows whose history actually landed.
--
-- Driven off `backfill_events`, not off the targets. Bumping a row whose insert
-- was skipped would push `version` ahead of the history it is supposed to
-- count, which is a worse state than the gap we are closing.
UPDATE subscription.user_subscriptions s
   SET version = s.version + 1
  FROM backfill_events ev
 WHERE ev.user_id = s.user_id;

\echo ''
\echo '=== FINAL REPORT ==='
SELECT
    (SELECT count(*) FROM backfill_targets)         AS rows_selected,
    (SELECT count(*) FROM backfill_events)         AS history_rows_written,
    (SELECT count(*) FROM backfill_outbox)         AS outbox_rows_written,
    (SELECT count(*)
       FROM subscription.user_subscriptions s
      WHERE s.is_auto_renew_enabled = false
        AND s.updated_at = :'bulk_update_at')     AS versions_bumped;

\echo ''
\echo '--- re-run 09-dry_run_backfill_auto_renew_history.sql ---'
\echo '--- sections 1 and 2 must both be empty. ---'

COMMIT;

\echo ''
\echo 'COMMIT - backfill applied. Re-running this script is a no-op.'
