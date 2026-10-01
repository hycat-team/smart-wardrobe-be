-- +goose Up
-- 024 T131 (FR-005d): the free plan's tier-2 per-day paid-attempt cap for `outfit`.
--
-- WHY THIS EXISTS. FR-005d settled on 4, and the value in the development database
-- is 4 - but it was put there by hand, with no artifact recording it. The only
-- committed statement of the value was migration 20260627225700, which set it to
-- 3. A fresh environment replaying the migrations therefore ended up on 3:
--
--   - free accounts were blocked at their 4th outfit attempt instead of their 5th,
--     so US2/AC9 was off by one;
--   - 07-verify_policy_grants.sql encodes the agreed rule as "free outfit pays for
--     one attempt fewer than the gate-1 allowance of 5", i.e. 4. On a value of 3
--     that predicate does not match and the row is reported as UNEXPLAINED DRIFT,
--     which breaks FR-090's promise that a result of 0 means verified.
--
-- A value that only exists on one machine is not configuration. This migration
-- makes the decision reproducible.
--
-- FR-029c requires the rationale in ENGLISH for a data script, which is why this
-- comment is in English while the user-facing messages are in Vietnamese.
--
-- IDEMPOTENT: guarded by IS DISTINCT FROM, so re-running it writes nothing.
-- The value is written only where it still disagrees with the decision - an
-- operator who has deliberately set something else is not silently overwritten
-- on every deploy. This is the same guard 04-update_ai_cost_limits.sql uses.

UPDATE aiexecutor.ai_cost_policy_operations AS o
SET max_paid_attempts_per_day = 4
FROM aiexecutor.ai_cost_policies AS p
WHERE o.policy_id = p.id
  AND p.code = 'free-default'
  AND o.operation = 'outfit'
  AND o.max_paid_attempts_per_day IS DISTINCT FROM 4;

-- +goose Down
-- Restore the value this migration replaced: 3, which is what
-- 20260627225700_update_ai_cost_quota_limits.sql established. That is the same
-- convention that migration follows (its own Down puts 5 back), and it is a real
-- rollback - "undo this migration" means returning to the state the migration chain
-- produced, not to whatever a machine happened to hold.
--
-- 024 T156: THE COMMENT HERE USED TO CLAIM THIS "lands on the same state a pre-T131
-- environment would have had". That sentence was false, and false in the way that
-- matters, because it hid a real operational consequence.
--
-- A pre-T131 environment did NOT necessarily have 3. The development database had
-- 4, put there by hand with no artifact recording it - which is precisely why this
-- migration exists. On that database Up is `UPDATE 0` (already 4, nothing to write)
-- and Down is `UPDATE 1` (4 down to 3), so up-then-down does not round-trip there.
--
-- What that means operationally, stated rather than left to be discovered:
--
--   - Rolling back on the development database moves the cap to 3. Free accounts are
--     then blocked at their 4th outfit attempt instead of their 5th, so US2/AC9 is
--     off by one.
--   - 07-verify_policy_grants.sql encodes the agreed rule as "free outfit pays for
--     one attempt fewer than the gate-1 allowance of 5", i.e. 4. On 3 that predicate
--     does not match and the row is reported as UNEXPLAINED DRIFT, which breaks
--     FR-090's promise that a result of 0 means verified.
--
-- This asymmetry is accepted rather than engineered away. The alternative - a Down
-- that refuses to move the value - would break the rollback convention the rest of
-- this directory follows, and would leave Down describing something it did not do.
-- The correct operational response is to re-run Up after a rollback, which is
-- idempotent.

UPDATE aiexecutor.ai_cost_policy_operations AS o
SET max_paid_attempts_per_day = 3
FROM aiexecutor.ai_cost_policies AS p
WHERE o.policy_id = p.id
  AND p.code = 'free-default'
  AND o.operation = 'outfit'
  AND o.max_paid_attempts_per_day IS DISTINCT FROM 3;
