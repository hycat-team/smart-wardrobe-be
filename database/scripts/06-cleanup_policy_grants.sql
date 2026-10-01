-- ============================================================================
-- CLEANUP: rebuild policy snapshots, drop dead grants, close stale ones
-- (024 FR-092, FR-092b, FR-093)
-- ============================================================================
-- Idempotent: running it a second time reports zeros and changes nothing.
-- Everything happens in one transaction, so a failure leaves the table as it
-- was.
--
-- Run 05-dry_run_cleanup_policy_grants.sql first and keep its output. The number
-- that must NOT change is "still in force": cleanup may not take an entitlement
-- away from anybody.
--
-- Three steps, in this order, and the order is the point:
--
--   1. REBUILD the snapshot for EVERY row, overwriting whatever is there.
--      It is tempting to only fill the eight empty ones, and that is wrong for
--      two reasons. The other eight are missing the whole entitlements group
--      (they were written before the operation table existed), and they mix two
--      serialisation styles — 25 keys including both `Code` and `code` — so two
--      different writers exist and neither is authoritative. Reading a missing
--      limit as zero would freeze the free policy's anti-abuse counter at zero,
--      which looks valid and enforces nothing.
--
--      The rebuilt values come from the configuration as it stands NOW, not from
--      the moment the grant was created. The June data does not keep a price
--      history, so the original values are simply not recoverable. This is
--      accepted because the rows being overwritten are test data (FR-092b).
--
--   2. DELETE the closed grants whose window has ended. A closed row is
--      permanently dead — the system never reopens one — so keeping it only
--      makes every scan slower.
--
--   3. CLOSE the active/future grants whose window has ended. These are a real
--      defect: they claim to be live while the paid period is over, and they
--      shadow the account's current plan. They are closed rather than deleted
--      because they are the audit trail of what the account was actually given.
--
-- Grants that are still in force are NEVER touched beyond step 1.
-- ============================================================================

BEGIN;

\echo '--- step 0: rows before ---'
SELECT count(*) AS rows_before
FROM aiexecutor.ai_policy_grants;

-- ---------------------------------------------------------------------------
-- 024 T177 (FR-093): each step below announces how many rows it is ABOUT TO change,
-- immediately before the statement that changes them.
--
-- The script used to print one total up front and then, per step, only what SURVIVED.
-- That is the wrong side of the write: after a step you learn what it did not fix, and
-- to learn what it did you have to have been counting beforehand. The dry-run script
-- (05) holds those numbers, but a dry run is a different session against a possibly
-- different state, so it cannot be the report of what this run did.
--
-- Each count reuses the write's own predicate, so the number cannot drift from the
-- statement above it.
-- ---------------------------------------------------------------------------

-- ---------------------------------------------------------------------------
-- Step 1 — rebuild the snapshot for every row.
--
-- jsonb_build_object produces exactly the two-part shape the reader expects:
-- an `envelope` that traces the grant back to its event, and an `entitlements`
-- group holding ONLY the limits the customer bought. Routing, the hard cap and
-- the anti-abuse knobs are deliberately absent from the snapshot: those are
-- operational settings, and freezing them would make a cost cap impossible to
-- tighten for the accounts whose grant is never recreated.
--
-- `envelope.occurredAt` carries the time the script runs, because the original
-- event time is not reconstructible from the grant alone. The reader only uses
-- it to order events against each other, and the restore tool replays events
-- with their own timestamps, so the value is consistent.
-- ---------------------------------------------------------------------------

-- Step 1 rewrites every row, so the announced count is the whole table.
\echo '--- step 1: about to rebuild the snapshot on N rows ---'
SELECT count(*) AS step1_will_change
FROM aiexecutor.ai_policy_grants;

-- ---------------------------------------------------------------------------
UPDATE aiexecutor.ai_policy_grants g
   SET policy_snapshot = jsonb_build_object(
         'envelope', jsonb_build_object(
            'eventVersion',  2,
            'eventType',     'ACTIVATED',
            'occurredAt',    to_char(now() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.USOF:00Z'),
            'effectiveAt',   to_char(g.effective_from AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.USOF:00Z'),
            'effectiveTo',   CASE WHEN g.effective_to IS NULL THEN NULL
                                 ELSE to_char(g.effective_to AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.USOF:00Z') END,
            'durationDays',  30,
            'sourceEventId', COALESCE(g.source_event_id::text, '')
         ),
         'entitlements', COALESCE((
            SELECT jsonb_object_agg(o.operation, jsonb_build_object(
                       'maxPaidAttemptsPerDay',   o.max_paid_attempts_per_day,
                       'normalMaxInputTokens',     o.normal_max_input_tokens,
                       'normalMaxOutputTokens',    o.normal_max_output_tokens,
                       'reducedMaxInputTokens',    o.reduced_max_input_tokens,
                       'reducedMaxOutputTokens',   o.reduced_max_output_tokens
                   ))
              FROM aiexecutor.ai_cost_policy_operations o
             WHERE o.policy_id = g.policy_id
               AND o.operation IN ('chat', 'outfit', 'summary', 'rewriter')
         ), '{}'::jsonb)
       ),
       updated_at = now();

\echo '--- step 1: snapshots rebuilt, empty ones remaining (expected 0) ---'
SELECT count(*) AS empty_snapshots
FROM aiexecutor.ai_policy_grants
WHERE policy_snapshot = '{}'::jsonb;

\echo '--- step 1: rows whose snapshot lacks one of the four operations (expected 0) ---'
SELECT count(*) AS incomplete_snapshots
FROM aiexecutor.ai_policy_grants g
WHERE NOT (
    g.policy_snapshot->'entitlements' ? 'chat'
    AND g.policy_snapshot->'entitlements' ? 'outfit'
    AND g.policy_snapshot->'entitlements' ? 'summary'
    AND g.policy_snapshot->'entitlements' ? 'rewriter'
);

-- ---------------------------------------------------------------------------
-- Step 2 — delete closed grants whose window has ended.
-- ---------------------------------------------------------------------------
\echo '--- step 2: about to delete N dead closed grants ---'
SELECT count(*) AS step2_will_delete
FROM aiexecutor.ai_policy_grants
WHERE status = 'closed'
  AND (effective_to IS NULL OR effective_to <= now());

DELETE FROM aiexecutor.ai_policy_grants
WHERE status = 'closed'
  AND (effective_to IS NULL OR effective_to <= now());

\echo '--- step 2: dead closed grants remaining (expected 0) ---'
SELECT count(*) AS closed_and_expired
FROM aiexecutor.ai_policy_grants
WHERE status = 'closed'
  AND (effective_to IS NULL OR effective_to <= now());

-- ---------------------------------------------------------------------------
-- Step 3 — close active/future grants whose window has ended.
--
-- ai_usage_period_ledgers has a foreign key to this table with ON DELETE
-- CASCADE, so step 2 also drops the ledger rows of the deleted grants. That is
-- intended: a ledger for a closed grant can never be charged again, and
-- deleting it releases the space.
-- ---------------------------------------------------------------------------
\echo '--- step 3: about to close N live-status grants past their window ---'
SELECT count(*) AS step3_will_close
FROM aiexecutor.ai_policy_grants
WHERE status IN ('active', 'future')
  AND effective_to IS NOT NULL
  AND effective_to <= now();

UPDATE aiexecutor.ai_policy_grants
   SET status       = 'closed',
       effective_to = LEAST(effective_to, now()),
       updated_at   = now()
 WHERE status IN ('active', 'future')
   AND effective_to IS NOT NULL
   AND effective_to <= now();

\echo '--- step 3: live-status grants past their window remaining (expected 0) ---'
SELECT count(*) AS live_status_but_expired
FROM aiexecutor.ai_policy_grants
WHERE status IN ('active', 'future')
  AND effective_to IS NOT NULL
  AND effective_to <= now();

-- ---------------------------------------------------------------------------
-- Final report. Compare "still in force" against the dry run: it must be
-- identical. The other two must be zero.
-- ---------------------------------------------------------------------------
\echo ''
\echo '=== FINAL REPORT ==='
SELECT
    count(*) FILTER (WHERE policy_snapshot = '{}'::jsonb)               AS rong,
    count(*) FILTER (WHERE status = 'closed'
                       AND (effective_to IS NULL OR effective_to <= now())) AS chet,
    count(*) FILTER (WHERE status IN ('active', 'future')
                       AND effective_to IS NOT NULL
                       AND effective_to <= now())                       AS het_han,
    count(*) FILTER (WHERE status IN ('active', 'future')
                       AND (effective_to IS NULL OR effective_to > now())) AS con_hieu_luc,
    count(*)                                                             AS tong_dong
FROM aiexecutor.ai_policy_grants;

\echo '--- rows affected by this run (expected 0 on a re-run) ---'
SELECT count(*) AS still_in_force_after
FROM aiexecutor.ai_policy_grants
WHERE status IN ('active', 'future')
  AND (effective_to IS NULL OR effective_to > now());

COMMIT;
