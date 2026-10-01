-- ============================================================================
-- VERIFY: six invariants of aiexecutor.ai_policy_grants (024 FR-090, FR-091)
-- ============================================================================
-- READ-ONLY and idempotent. Run it as often as you like.
--
-- Every one of the six counts must be 0. Each corresponds to a way the table
-- could be wrong, and each of those ways produced a real customer-visible
-- defect before this feature existed.
-- ============================================================================

\echo ''
\echo '=== 1. Accounts with a plan but no policy grant ==='
\echo '--- Expected 0. A non-zero count means those accounts cannot use paid AI'
\echo '--- at all: the read path finds no grant and degrades, and the customer'
\echo '--- sees a free-model answer for something they paid for. The restore'
\echo '--- tool (POST /api/v1/admin/subscription-grants/restore) is the fix. ---'
SELECT count(*) AS accounts_missing_grant
FROM subscription.user_subscriptions s
WHERE NOT EXISTS (
    SELECT 1 FROM aiexecutor.ai_policy_grants g
     WHERE g.user_id = s.user_id
       AND g.status IN ('active', 'future')
       AND g.effective_from <= now()
       AND (g.effective_to IS NULL OR g.effective_to > now())
);

\echo ''
\echo '=== 2. Accounts with more than one grant in force at once ==='
\echo '--- Expected 0. Each row is an account granted twice, which means the'
\echo '--- close and the create were not atomic. ---'
SELECT user_id, count(*) AS concurrent_grants
FROM aiexecutor.ai_policy_grants
WHERE status = 'active'
  AND effective_from <= now()
  AND (effective_to IS NULL OR effective_to > now())
GROUP BY user_id
HAVING count(*) > 1;

\echo ''
\echo '=== 3. Grant plan_code disagreeing with the account current plan ==='
\echo '--- Expected 0 rows. The grant records which plan it was written for; a'
\echo '--- mismatch means an event was lost or arrived out of order. ---'
SELECT g.id, g.user_id, g.plan_code AS grant_plan, s.current_plan_code AS subscription_plan
FROM aiexecutor.ai_policy_grants g
JOIN subscription.user_subscriptions s ON s.user_id = g.user_id
WHERE g.status = 'active'
  AND g.effective_from <= now()
  AND (g.effective_to IS NULL OR g.effective_to > now())
  AND g.plan_code <> s.current_plan_code;

\echo ''
\echo '=== 4. Grants with an empty or incomplete policy snapshot ==='
\echo '--- Expected 0. The reader needs an `entitlements` group with all four'
\echo '--- operations. Missing it means no limits at all are applied, which is'
\echo '--- silent: nothing errors, the limits are simply not there. ---'
SELECT count(*) AS bad_snapshots
FROM aiexecutor.ai_policy_grants
WHERE policy_snapshot = '{}'::jsonb
   OR NOT (policy_snapshot ? 'envelope')
   OR NOT (policy_snapshot ? 'entitlements')
   OR NOT (
        policy_snapshot->'entitlements' ? 'chat'
    AND policy_snapshot->'entitlements' ? 'outfit'
    AND policy_snapshot->'entitlements' ? 'summary'
    AND policy_snapshot->'entitlements' ? 'rewriter'
   );

\echo ''
\echo '=== 5. Active or future grants whose window has ended ==='
\echo '--- Expected 0. A row claiming to be live after its paid period is over'
\echo '--- shadows the account current plan with a revoked entitlement. ---'
SELECT count(*) AS live_but_expired
FROM aiexecutor.ai_policy_grants
WHERE status IN ('active', 'future')
  AND effective_to IS NOT NULL
  AND effective_to <= now();

\echo ''
\echo '=== 6. Gate 1 and gate 2 disagreeing, per operation ==='
\echo '--- Expected 0 rows. This is the most customer-visible check in the'
\echo '--- file: a customer whose profile shows more chat attempts per day than'
\echo '--- the engine actually allows has no way to tell, because the engine'
\echo '--- silently downgrades the model instead of returning an error.'
\echo '---'
\echo '--- Exactly ONE approved exception is excluded below, because it is a'
\echo '--- decision rather than drift: the free plan outfit limit is deliberately'
\echo '--- one attempt below what the customer sees, i.e. gate 1 = 5 and'
\echo '--- gate 2 = 4 (FR-005d). Any OTHER difference is unexplained and must'
\echo '--- be investigated.'
\echo '---'
\echo '--- premium-default is NOT a second exception. It does not drift because'
\echo '--- the WHERE clause above never lets it in: a strict policy never reads'
\echo '--- max_paid_attempts_per_day, so its value is NULL on purpose (FR-005e)'
\echo '--- and the rows are dropped by enforcement_mode = observe_only before any'
\echo '--- comparison happens. SC-011 approves a one-item exception list, and'
\echo '--- this query honours exactly that list.'
\echo '---'
\echo '--- Section 6b below prints every row with a verdict, so the approved'
\echo '--- exception stays visible instead of being silently hidden. ---'
-- 024 T157: this comment used to say "sections 6 AND 6b now report ALL FOUR
-- operations. ... So the rows are all shown". That was only true of 6b. Section 6
-- still filters `o.operation IN ('outfit','chat')` below, and it should - so the
-- comment went on describing a query that does not exist.
--
-- Why that matters more than a stale comment usually does: section 6 IS the drift
-- query. Its whole output is supposed to be the unexplained cases and nothing else,
-- and FR-090 promises a result of 0 means verified. Anyone auditing this script
-- reads the comment to decide whether the filter is intended or a leftover - and the
-- comment said there was no filter. The query was right; the description of it was
-- wrong.
--
-- What the two sections actually do, and why they differ:
--
--   - Section 6 is the drift DETECTOR. It lists only operations that CAN disagree:
--     `outfit` and `chat`, the only two with a customer-visible quota column
--     (`ai_outfit_daily_quota`, `ai_chat_daily_quota` - there is no column for
--     summary or rewriter). A row with no gate-1 counterpart has nothing to be
--     compared against, so including it would print NULL as if it were a finding.
--
--   - Section 6b is the full INVENTORY. It prints every metered operation with a
--     verdict, so the two operations section 6 cannot judge stay visible instead of
--     being silently absent - which is what FR-090 asks for by way of "an
--     explanation for every tier-2 value that has no tier-1 counterpart".
--
-- So the four operations are all accounted for; they are just accounted for in two
-- places doing two different jobs.
SELECT
    sp.slug                       AS plan_slug,
    p.code                        AS policy_code,
    p.enforcement_mode,
    o.operation,
    CASE o.operation
        WHEN 'outfit' THEN sp.ai_outfit_daily_quota
        WHEN 'chat'   THEN sp.ai_chat_daily_quota
    END                           AS gate1_customer_visible,
    o.max_paid_attempts_per_day   AS gate2_ai_enforced,
    CASE WHEN o.operation IN ('outfit', 'chat')
         THEN o.max_paid_attempts_per_day - CASE o.operation
                WHEN 'outfit' THEN sp.ai_outfit_daily_quota
                WHEN 'chat'   THEN sp.ai_chat_daily_quota
              END
    END                           AS drift
FROM subscription.subscription_plans sp
JOIN aiexecutor.ai_cost_policies p          ON p.id = sp.ai_cost_policy_id
JOIN aiexecutor.ai_cost_policy_operations o ON o.policy_id = p.id
WHERE -- Scope filter, NOT an exception: a strict policy never reads
        -- max_paid_attempts_per_day, so premium-default does not take part in the
        -- two-gate comparison at all (FR-005e). Listing it as an exception would
        -- double-count: it is already excluded here, before any comparison.
  p.enforcement_mode = 'observe_only'
  AND o.max_paid_attempts_per_day IS NOT NULL
  -- Only rows that CAN disagree. summary and rewriter have no gate-1 column, so
  -- there is nothing to compare them against; section 6b lists them with their
  -- value and says so.
  AND o.operation IN ('outfit', 'chat')
  -- The single approved exception (SC-011): the deliberate one-attempt difference
  -- on the free outfit cap, i.e. gate 1 = 5 and gate 2 = 4. This predicate is the
  -- whole approved exception list, so nothing else may be excluded here.
  AND NOT (sp.slug = 'free' AND o.operation = 'outfit'
           AND o.max_paid_attempts_per_day = sp.ai_outfit_daily_quota - 1)
  AND o.max_paid_attempts_per_day <> CASE o.operation
        WHEN 'outfit' THEN sp.ai_outfit_daily_quota
        WHEN 'chat'   THEN sp.ai_chat_daily_quota
      END;

\echo ''
\echo '--- 6b. Full gate comparison, including the approved exception ---'
\echo '--- Every metered operation is listed, not just the two with a customer-'
\echo '--- visible quota. Each row gets a verdict so the approved exception stays'
\echo '--- visible instead of being silently hidden. ---'
SELECT
    sp.slug                       AS plan_slug,
    p.code                        AS policy_code,
    p.enforcement_mode,
    o.operation,
    CASE o.operation
        WHEN 'outfit' THEN sp.ai_outfit_daily_quota
        WHEN 'chat'   THEN sp.ai_chat_daily_quota
    END                           AS gate1_customer_visible,
    o.max_paid_attempts_per_day   AS gate2_ai_enforced,
    CASE
        WHEN p.enforcement_mode = 'strict'
            THEN 'strict: column is never read (FR-005e)'
        -- Listed for completeness rather than omitted: an administrator can see the
        -- value the engine enforces, and knows there is nothing customer-visible to
        -- compare it against. Reporting these as drift would be wrong; hiding them
        -- would be worse.
        WHEN o.operation NOT IN ('outfit', 'chat')
            THEN 'no customer-visible quota for this operation; nothing to compare'
        WHEN sp.slug = 'free' AND o.operation = 'outfit'
             AND o.max_paid_attempts_per_day = sp.ai_outfit_daily_quota - 1
            THEN 'deliberate one-attempt difference (FR-005d)'
        WHEN o.max_paid_attempts_per_day IS NULL
            THEN 'observe_only with NULL: MISCONFIGURATION (FR-005e)'
        WHEN o.max_paid_attempts_per_day = CASE o.operation
                 WHEN 'outfit' THEN sp.ai_outfit_daily_quota
                 WHEN 'chat'   THEN sp.ai_chat_daily_quota
             END
            THEN 'match'
        ELSE 'UNEXPLAINED DRIFT - investigate'
    END                           AS verdict
FROM subscription.subscription_plans sp
JOIN aiexecutor.ai_cost_policies p          ON p.id = sp.ai_cost_policy_id
JOIN aiexecutor.ai_cost_policy_operations o ON o.policy_id = p.id
ORDER BY sp.slug, o.operation;
