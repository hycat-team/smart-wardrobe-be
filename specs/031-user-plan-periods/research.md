# Phase 0 Research — Lịch sử gói theo khoảng hiệu lực

**Feature**: `031-user-plan-periods` | **Date**: 2026-10-20

All unknowns from `plan.md` Technical Context are resolved below. Code references are to the current
tree; nothing here is speculative beyond the stated alternatives.

## R1 — Storage model: validity intervals, not snapshots

**Decision**: append-only `subscription.user_plan_periods`, one row per continuous period a user holds
a plan (`valid_from`, `valid_to`, plan snapshot).

**Rationale**: the report needs "which plan did this user hold at any point in [from, to)". A validity
interval answers that exactly, for every period, without a per-day row. A user with three plan changes
in a year produces three rows regardless of how many days the report covers. It also preserves
contiguity/overlap invariants that can be enforced by the database.

**Alternatives considered**:
- *Daily/monthly snapshot table*: write path untouched, but 100k users × 365 days ≈ 36.5M rows/year,
  loses intra-day changes, and depends on a scheduler that can silently miss a day. Rejected.
- *Reconstruct on read from `user_subscription_events` + `deposit_transactions`*: `EXTENDED` records
  `effective_at = renewal time`, not the resulting expiry, and events carry plan codes, not ids; the
  resulting windows cannot be reproduced exactly. Rejected (already documented in `029/D-06`).
- *Event sourcing the whole subscription*: far larger refactor than the debt justifies. Rejected.

## R2 — Write hook: extend `PlanChangePublisher`

**Decision**: add a `PlanPeriodRecorder` collaborator to `PlanChangePublisher` and call it after the
history event insert succeeds
(`services/account/internal/modules/subscription/application/usecase/subscription/plan_change_publish_helper.go:66`).

**Rationale**: every plan mutation already depends on the publisher (`024 T045`), it runs inside the
caller's transaction, and it receives the event type, `EffectiveAt`, and the updated subscription. All
twelve call sites are covered by construction; forgetting to record becomes impossible without editing
the publisher. A retry hits the `event_key` early-return and never re-runs the recorder, which gives
idempotency for free.

**Alternatives considered**:
- *Call the recorder at each mutation site*: twelve places to keep in sync, and the next new path will
  forget. Rejected.
- *Database trigger on `user_subscriptions`*: cannot see the event's `effective_at` or the previous
  window reliably, and puts business rules in SQL. Rejected.
- *Consume `subscription.plan.changed` from RabbitMQ*: asynchronous, so the report would lag behind
  commits and lose ordering under retries. Rejected.

## R3 - Non-overlap enforcement

**Decision**: attempt `EXCLUDE USING gist (user_id WITH =, tstzrange(valid_from, valid_to, '[)') WITH &&)`
plus `CHECK (valid_to IS NULL OR valid_to > valid_from)` inside one `DO` block. `btree_gist` is required
for the uuid equality operator; the block attempts `CREATE EXTENSION IF NOT EXISTS btree_gist` and the
constraint in a subtransaction. If either fails, it creates a partial unique index on
`(user_id) WHERE valid_to IS NULL` instead and raises a notice.

**Rationale**: a validity history is wrong the moment two intervals for one user overlap; a database
constraint keeps that true even if a future code path bypasses the recorder. `tstzrange` treats a NULL
upper bound as infinity, matching free/lifetime plans. Adjacent intervals (`end == start`) do not
overlap under `[)`. The adaptive form keeps the schema portable across self-hosted and managed
PostgreSQL (constitution VIII) without a code/config fork; the full exclusion constraint is used
wherever `btree_gist` is available.

**Alternatives considered**:
- *Partial unique only*: guarantees at most one open-ended interval but allows two closed intervals to overlap. It is
  the automatic fallback, not the preferred shape; the recorder's `FOR UPDATE` on the current row is the
  primary control there.
- *Hard-fail when the extension is missing*: turns a schema portability problem into a deploy failure
  and forces a per-environment code/config change. Rejected.
- *Application-only check*: loses the invariant on the first concurrent writer. Rejected.
## R4 — Extend semantics

**Decision**: an `EXTENDED` event updates `valid_to` of the current interval to the new
`sub.ExpiresAt`; it never inserts a row.

**Rationale**: `UserSubscription.Extend` (`domain/entities/subscription_behavior.go:72`) explicitly
does not change `StartedAt`; it only moves `ExpiresAt`. Inserting a new interval would split one
continuous holding into two and double-count the user if a report window touched both. The event key
still identifies the change, and the retry path never reaches the recorder (R9).

**Alternatives considered**: insert a row per renewal and close the previous one at the renewal time.
Rejected: it misrepresents "same plan, continuous" as a gap/overlap and inflates row count.

## R5 — Downgrade and expiry boundary

**Decision**: for `SUBSCRIPTION_DOWNGRADED_TO_FREE`, close the paid interval at
`min(current.valid_to, event.EffectiveAt)` and open the free interval at that same instant. The same
`min` rule closes the finite interval in `LIFETIME_FALLBACK_RESTORED`, whose `EffectiveAt` is already
the previous expiry (`subscription_behavior.go:256`).

**Rationale**: the worker may run minutes after expiry; using `EffectiveAt` alone would extend the
paid plan past the moment it ended, contradicting `FR-009`. Using the interval's own `valid_to` when it
is earlier keeps the boundary at the contractual expiry and avoids a hole before free starts. This is
`FR-005`.

**Alternatives considered**: use `EffectiveAt` unconditionally (overstates paid time); use the job's
run time and leave a gap (drops the user from every bucket). Both rejected.

**Same-instant and same-plan guard**: the admin restore path emits `ACTIVATED` with
`EffectiveAt = sub.StartedAt` for the plan the user already holds. Closing at that instant would produce
`valid_to == valid_from` and violate the check constraint. The recorder therefore no-ops when the
target plan equals the current interval and replaces the current row (rather than closing it) whenever
`closeAt <= current.valid_from`. This also covers provisioning + first grant in the same transaction.

## R6 — Backfill location and form

**Decision**: one `INSERT ... SELECT` statement inside the schema migration (joining
`subscription_plans` for `plan_name`), plus a cutover row in `subscription.plan_history_meta` with
`now()`, and a `RAISE NOTICE` carrying the measured duration.

**Rationale**: the only derivable history is the current row's `[started_at, expires_at]` window, and it
is exactly correct for that window (any change resets `started_at`). Doing it in the migration makes
backfill atomic with the schema and impossible to forget; `ON CONFLICT (source_event_key) DO NOTHING`
makes a re-run harmless. No deploy-order coupling between code and a job.

**Alternatives considered**: a Go startup job (multiple replicas, needs a lock and its own metadata
write); a CLI command run by an operator (easy to forget, leaves the report broken until someone runs
it). Both rejected.

**Volume**: one row per `user_subscriptions` row (PK `user_id`), so the insert is bounded by the user
count, not by history.

## R7 — Cutover metadata: table, not config

**Decision**: `subscription.plan_history_meta(id text PRIMARY KEY, started_at timestamptz NOT NULL,
note text)` with a single `cutover` row.

**Rationale**: the cutover is data produced by the backfill, not an environment setting. A config value
would have to be hand-maintained in every environment after the deploy and could disagree with the rows
actually inserted. The report reads the same row the backfill wrote.

**Alternatives considered**: env var (`SUBSCRIPTION_PLAN_HISTORY_STARTED_AT`) — rejected for the drift
risk; deriving the minimum `valid_from` from the table — wrong, because backfilled intervals start at
each user's `started_at`, which can predate the cutover.

## R8 — How the response reports history completeness

**Decision**: two additive fields: `dataAvailableFrom` (string, VN `YYYY-MM-DD`) and
`dataCompleteness` (`complete` | `partial` | `unavailable`). On the wire:
`google.protobuf.Timestamp data_available_from` and an enum `DistributionDataCompleteness`; the
Commerce edge formats the timestamp to the string and treats zero bounds as infinity when computing
the status.

**Rationale**: `FR-012` requires the report to say when a period is before the cutover (no 0s that look
real) and when it straddles it. A boolean cannot distinguish "partial" from "unavailable", and the
absence of a field cannot be distinguished from an old server. The enum is explicit and additive. The
string date avoids an RFC 3339 value disagreeing with the VN day boundaries.

**Absent-cutover behaviour**: the migration always writes the row, but a manual truncation must not
silently turn into "complete". A period request then returns `dataCompleteness = unavailable` and omits only
`dataAvailableFrom` (warning logged); the snapshot stays `complete` because it reads current state only.
**Alternatives considered**: only a timestamp (the client has to compare dates itself, duplicating the
rule); a boolean `partial` (cannot express unavailable); no fields with docs only (fails `FR-012` for a
machine client). Rejected.

## R9 — Idempotency

**Decision**: rely on the publisher's existing `event_key` uniqueness and its early return: the
recorder runs only after a *new* history event is inserted. `source_event_key` on the interval table is
`UNIQUE` (nullable) as a second guard and for traceability.

**Rationale**: the recorder is called once per newly inserted event; a retry or a lost insert race
returns the existing row before the recorder is reached. Adding a second idempotency mechanism would
duplicate the publisher's contract and could itself disagree with it.

**Alternatives considered**: recorder-side "has this event key already been applied" check. Rejected as
redundant; a crash between event insert and recorder cannot happen because both are in one transaction.

## R10 — Query shape and performance

**Decision**: one indexed range predicate plus `GROUP BY plan_code` with `COUNT(DISTINCT user_id)`.

```sql
WHERE tstzrange(valid_from, valid_to, '[)') && tstzrange($from, $to, '[)')
```

`$from`/`$to` are SQL `NULL` for an absent side. The GiST index on
`tstzrange(valid_from, valid_to, '[)')` serves this predicate and needs no extension; `(user_id,
valid_from)` serves the per-user history lookup of `US2`; `(valid_from)` narrows "started before".
`COUNT(DISTINCT user_id)` is required by `FR-010` (a user who left and returned to the same plan
within the window counts once).

**Rationale**: at 200k rows the aggregate is small; the half-open predicate matches the VN day bounds
already produced by `ParseVNDateRangeQuery` (`shared/pkg/utils/validation/date_range.go:24`). No join to
plans at read time: `plan_code`/`plan_name` are snapshotted, which also preserves renamed plans as they
were.

**Alternatives considered**: only the btree index on `valid_from` (wide windows scan nearly the whole
table); the exclusion constraint's GiST index alone (its leading column is `user_id`, so a report
without a user filter cannot use it); `COUNT(*)` (violates `FR-010`). Rejected.

## R11 — Testing strategy

**Decision**:
- unit tests for the recorder transition table with a fake repository and fixed clocks (each event
  type, same-plan admin restore no-op, same-instant replacement, retry, expired-open-window);
- unit tests for the completeness calculation (zero bounds, one-sided bounds, before/after/straddling
  the cutover, absent cutover row);
- integration tests on a migration-built Postgres: backfill row counts and cutover, both constraint
variants (exclusion and partial-unique fallback, the latter under a role without `CREATE EXTENSION`), distribution for the canonical lifecycles
  (downgrade after period, mid-period switch, extend, lapse gap), one-sided periods, `COUNT(DISTINCT)`
  for return-to-plan, and the unaffected-paths check for admin restore;
- SQL pin tests updated for the new statement (inspection only, as the existing ones at
  `infrastructure/persistence/admin_stats_query_sql_test.go:244`);
- transport mapping tests for the two new proto fields plus a compatibility test for the old JSON shape
  and unknown proto fields;
- bounded concurrency test for two simultaneous changes to one user (row lock + constraint).

**Rationale**: the existing suite already distinguishes statement-shape tests from real-Postgres
behavior; keeping that split makes a regression show up in the right layer. The integration harness
requires a migration-built schema (025/D-01, 026/D-06), so the new migration is exercised by the same
gate as production.

**Alternatives considered**: sqlite for the repository tests — rejected for the exclusion constraint and
`COUNT(DISTINCT)` semantics, which are Postgres-specific.
