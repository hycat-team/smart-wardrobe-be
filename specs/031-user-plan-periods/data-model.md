# Phase 1 Data Model — Lịch sử gói theo khoảng hiệu lực

**Feature**: `031-user-plan-periods` | **Date**: 2026-10-20 | **Owner**: Account service

This model is additive. `subscription.user_subscriptions` keeps its current meaning and is not
modified; nothing else reads the new tables except the distribution report and the documented
operator query.

## 1. Entity: `UserPlanPeriod` → `subscription.user_plan_periods`

One row per continuous period during which a user held a plan.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | PK, `DEFAULT gen_random_uuid()` | |
| `user_id` | `uuid` | `NOT NULL`, FK `identity.users(id) ON DELETE CASCADE` | history follows the account lifetime (`FR-014`) |
| `subscription_plan_id` | `uuid` | `NOT NULL`, FK `subscription.subscription_plans(id) ON DELETE RESTRICT` | a plan with history cannot be deleted |
| `plan_code` | `varchar(100)` | `NOT NULL` | snapshot at the time of the period |
| `plan_name` | `varchar(255)` | `NOT NULL` | snapshot; `FR-001` requires the name the user saw |
| `tier_rank` | `int` | `NOT NULL` | snapshot |
| `plan_kind` | `smallint` | `NOT NULL` | snapshot (`0` free, `1` finite, `2` lifetime) |
| `valid_from` | `timestamptz` | `NOT NULL` | inclusive start (`started_at`, or the event's effective time) |
| `valid_to` | `timestamptz` | nullable | exclusive end (`expires_at`); `NULL` means open-ended (free / lifetime) |
| `source_event_key` | `varchar(255)` | `UNIQUE` (nullable) | event key that created/updated the row; `BACKFILL:<user_id>` for the initial window |
| `created_at` | `timestamptz` | `NOT NULL DEFAULT now()` | |
| `updated_at` | `timestamptz` | `NOT NULL DEFAULT now()` | `EXTENDED` moves `valid_to`, so this column moves too |

Constraints:

```sql
CHECK (valid_to IS NULL OR valid_to > valid_from)
```

Primary non-overlap constraint (only when `btree_gist` is available):

```sql
ALTER TABLE subscription.user_plan_periods
    ADD CONSTRAINT ex_user_plan_periods_no_overlap
    EXCLUDE USING gist (
        user_id WITH =,
        tstzrange(valid_from, valid_to, '[)') WITH &&
    );
```

Fallback when the extension cannot be created (managed/restricted role): the migration creates

```sql
CREATE UNIQUE INDEX ux_user_plan_periods_open
    ON subscription.user_plan_periods (user_id)
    WHERE valid_to IS NULL;
```

Both variants keep "at most one open interval per user"; only the exclusion variant guarantees that
two **closed** intervals cannot overlap. The recorder's `FOR UPDATE` on the open row plus the close
rules are the primary control in both cases.

Indexes (always created, no extension required):

```sql
CREATE INDEX idx_user_plan_periods_valid_from
    ON subscription.user_plan_periods (valid_from);

CREATE INDEX idx_user_plan_periods_user_valid_from
    ON subscription.user_plan_periods (user_id, valid_from);

CREATE INDEX idx_user_plan_periods_range
    ON subscription.user_plan_periods
    USING gist (tstzrange(valid_from, valid_to, '[)'));
```

The GiST range index serves the report's overlap predicate; the btree index on `valid_from` is kept
for cheap "started before X" narrowing and the per-user index serves the US2 look-up.

## 2. Entity: `PlanHistoryMeta` → `subscription.plan_history_meta`

One row describing when complete history starts.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `varchar(20)` | PK | only `'cutover'` is used |
| `started_at` | `timestamptz` | `NOT NULL` | set by the backfill to `now()` |
| `note` | `text` | nullable | operator-facing explanation |

Read once per distribution request. The migration guarantees the row. If it is absent (manual
truncation, partial restore), the report treats any **period** request as `unavailable` (enum emitted,
`dataAvailableFrom` omitted because the date is unknown), keeps the no-period snapshot as `complete`,
and logs a warning — it never invents a
cutoff or claims complete history.

## 3. State transition table (recorder rules)

`source=admin_restore` is checked FIRST and overrides every row below: such an event is an absolute
no-op — it repairs AI grants, not the subscription window, and must never overwrite a newer interval.

`open` = the row for the user with `valid_to IS NULL`, read `FOR UPDATE` inside the caller's
transaction. `effective = event.EffectiveAt`. `closeAt = min(open.valid_to, effective)` when
`open.valid_to` is set and earlier, otherwise `effective`.

| Event | Condition | Action |
|---|---|---|
| `ACTIVATED` | target plan code == `open.plan_code` | no-op (admin restore; plan and window unchanged) |
| `ACTIVATED`, `UPGRADED`, `OVERLAID` | `closeAt <= open.valid_from` | replace the open row: new plan snapshot, `valid_from = max(effective, open.valid_from)`, `valid_to = sub.ExpiresAt`, new `source_event_key` |
| `ACTIVATED`, `UPGRADED`, `OVERLAID` | otherwise | close at `closeAt`; insert new row from `effective` with `valid_to = sub.ExpiresAt` |
| `EXTENDED` | open row exists | `valid_to = sub.ExpiresAt`; no new row |
| `EXTENDED` | no open row | insert the current window from `sub.StartedAt` |
| `SUBSCRIPTION_DOWNGRADED_TO_FREE` | any | close at `closeAt`; insert free row from `closeAt`, `valid_to NULL` |
| `LIFETIME_FALLBACK_RESTORED` | any | close at `closeAt`; insert lifetime row from `closeAt`, `valid_to NULL` |
| `AUTO_RENEW_UPDATED` | any | no-op |

Rules that follow:

- `valid_from` of a new row is `max(effective, open.valid_from)`; a zero/negative interval is never
  written, and the replacement branch handles the same-instant case.
- If there is no open interval, the recorder inserts the new interval and does not invent a close.
- A lapse leaves a gap; continuous switches never do.
- The admin restore path (`restore_policy_grants_uc.go`) emits `ACTIVATED` with
  `EffectiveAt = sub.StartedAt` for the **same** plan. Without the same-plan guard it would close the
  open interval at its own start and violate the check constraint. That guard is required, not an
  optimisation.
- The recorder takes the target `plan` from the publisher so `plan_name` is the plan's name at that
  moment; it never re-reads the catalogue.

## 4. Backfill

Executed inside the schema migration, in this order:

1. Preflight: a `DO` block aborts the migration with `RAISE EXCEPTION` naming the user ids if any
   row has `current_plan_kind = 1 AND expires_at <= started_at`. Such a row cannot be represented as
   a valid interval and silently skipping it would remove that user from the report (FR-013).
2. `INSERT INTO subscription.user_plan_periods (...) SELECT ... FROM subscription.user_subscriptions
   JOIN subscription.subscription_plans ON id = subscription_plan_id
   ON CONFLICT (source_event_key) DO NOTHING` — one row per user, `valid_from = started_at`,
   `valid_to = expires_at`, snapshot of `current_plan_*` plus `plan_name`, key `BACKFILL:<user_id>`.
3. `INSERT INTO subscription.plan_history_meta (id, started_at, note) VALUES ('cutover', now(), ...)
   ON CONFLICT (id) DO NOTHING` — first deploy wins; a re-run does not move the cutover forward.

Expired current rows are backfilled with a `valid_to` in the past: they are history, and the report
needs them for the window in which they were valid. Corrupt rows (`expires_at <= started_at`) stop
the migration before the insert; they are repaired by an operator, never skipped. The migration logs its duration via `RAISE NOTICE` so the
deploy record shows how long the first replica's startup was delayed. Backfilled `plan_name` is the
catalogue name at deploy time, not necessarily the name when the window started; earlier renames are
not recoverable.

## 5. Read model and report semantics

```go
type SubscriptionDistributionResult struct {
    Items             []SubscriptionDistributionDTO
    DataAvailableFrom *time.Time // date in VN; formatter emits YYYY-MM-DD
    DataCompleteness  string     // complete | partial | unavailable
}
```

`SubscriptionDistributionDTO` keeps `PlanCode` and `UserCount`; percentages and `TotalUsers` remain in
the Commerce use case, computed from the buckets (`FR-011`, clarified: sum of buckets, so a mid-period
switcher appears in two buckets and `totalUsers` is not a distinct-user count).

Query (period supplied; a zero bound means infinity):

```sql
SELECT plan_code, COUNT(DISTINCT user_id) AS user_count
FROM subscription.user_plan_periods
WHERE tstzrange(valid_from, valid_to, '[)') && tstzrange($from, $to, '[)')
GROUP BY plan_code
ORDER BY user_count DESC;
```

Bound handling (applied in the repo; `$from`/`$to` become SQL `NULL` for an absent side):

| Bounds | Predicate |
|---|---|
| both zero (snapshot) | `tstzrange(valid_from, valid_to, '[)') @> now()` |
| `to` only | range overlaps `(-infinity, to)` **and** `valid_from <= now()` |
| `from` only | range overlaps `[from, infinity)` |
| both | range overlaps `[from, to)` |

`COUNT(DISTINCT user_id)` implements `FR-010` (leave and return to the same plan inside the window
counts once). With the exclusion constraint a user cannot be in two overlapping rows of the same plan;
the distinct handles disjoint rows of the same plan inside the window.

## 6. Completeness matrix and clipping

`C` = cutover, `F` = from (zero = unbounded), `T` = to (zero = unbounded).

| F | T | Result | Query window |
|---|---|---|---|
| zero | zero | `complete` (current snapshot) | `now()` |
| any | provided and `T <= C` | `unavailable` | none (empty result, `totalUsers = 0`, flagged) |
| zero or `F < C` | `T > C` or zero | `partial` | `[max(F, C), T)`; if `F` zero, `[C, T)` |
| `F >= C` | any | `complete` | `[F, T)` |

`unavailable` must not be rendered as a real zero chart (`FR-012`); the response carries the enum and,
when known, the cutover date so the client can say so. `partial` counts only the period after the
cutover — the earlier part is not silently included.

## 7. Ownership and migration

- Schema `subscription`, owned by Account (`030-ddl-ownership-handover` contract); migration file in
  `services/account/database/migrations/`.
- New tables appear in Account's migration tree only; no baseline edit (030 handover is complete).
- Down migration drops both tables and their indexes. It deliberately does **not** drop `btree_gist`,
  which is shared infrastructure and may pre-exist.
- Rollback loses history. This is not routine: a down/up cycle in production discards every interval and
  re-backfills only the current window, moving the cutover forward. Release notes must say so.
