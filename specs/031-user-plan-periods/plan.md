# Implementation Plan: Lịch sử gói theo khoảng hiệu lực

**Branch**: `031-user-plan-periods` | **Date**: 2026-10-20 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/031-user-plan-periods/spec.md`

**Note**: This plan closes deferred item `029/D-06`. The deferred entry stays open until implementation lands.

## Summary

`subscription.user_subscriptions` is a one-row-per-user projection, so every plan change overwrites
`started_at`/`current_plan_code`; the admin distribution report therefore loses the previous plan and
drops users who changed plan after the queried period. This plan adds an append-only
`subscription.user_plan_periods` table (validity intervals), writes it inside the same transaction
that already publishes plan-change events, backfills only the current window at deploy time, and moves
the distribution query onto the interval table. The admin response keeps its existing shape and gains
two optional fields describing history completeness.

## Technical Context

**Language/Version**: Go 1.25+ (multi-module Go workspace, `go.work`), Gin, GORM, Wire.

**Primary Dependencies**: PostgreSQL (pgvector image), Goose migrations per service, gRPC + protobuf
for cross-service calls, Swag for Swagger. Optional extension: `btree_gist`, used only when available
to add the full non-overlap exclusion constraint; the migration degrades to a partial unique index
without it.

**Storage**: PostgreSQL. New table `subscription.user_plan_periods` and metadata table
`subscription.plan_history_meta`, both in migrations owned by the Account service
(`services/account/database/migrations/`).

**Testing**: `go test -race` per package; DB-backed integration tests through
`make test-db-integration` with `TEST_DATABASE_URL` pointing at a migration-built database; SQL pin
tests for generated statements; transport mapping tests; a compatibility test for the additive proto
fields.

**Target Platform**: Linux containers (VPS Compose and managed platform); local Windows development
via the repo Makefile.

**Project Type**: Backend multi-service web API (Account, Commerce Core, AI Engine).

**Performance Goals**: distribution for 200k interval rows at p95 < 1 s (SC-004). The upper bound at
5 changes/user/year for 200k users is ~1M rows; T034 measures 200k for SC-004 and runs 1M informationally to check the layout does not collapse.

**Constraints**: no planned downtime, but the first replica's startup is delayed by the one-time
backfill while the others wait on Goose's advisory lock; the write path adds one indexed insert (or
one update for extensions) per plan change; no new infrastructure.

**Scale/Scope**: 200k users, 200k backfilled rows at deploy, up to ~1M rows after a year of changes.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Gate | Verdict |
|---|---|---|
| I. Independently deployable services | No new service, no cross-service deployment coupling | PASS |
| II. Module boundaries & DTO contracts | New DTO fields flow through the existing `ISubscriptionDashboardReader` contract and protobuf; no cross-module entity import | PASS |
| III. Async event-driven processing | Recorder writes in the caller's DB transaction as a read projection; no new event, outbox untouched | PASS |
| IV. Clean architecture layering | Domain repository interface + infrastructure GORM repo; recorder in application; handlers unchanged | PASS |
| V. Transaction & error handling | Recorder uses `ctx`-propagated transaction; failures roll back the business mutation; errors use `apperror` at the presentation edge | PASS |
| VI. AI pipeline | Not touched | PASS |
| VII. Design patterns | Repository naming `Get...`; mappers live in `application/mapper` where a DTO is not a pure alias; single-use helper suffix `_helper.go` | PASS |
| VIII. Infrastructure portability | `btree_gist` is not required: the migration attempts it and falls back to a partial unique index on the same cluster; no configuration switch, no code fork, and no vendor-specific API | PASS |
| Governance: migrations via Goose | New migration files in the owning Account tree; `make migration-check` must stay green | PASS |
| Governance: frontend guide | Distribution response changes (optional fields) → `specs/031-user-plan-periods/frontend-guide.md` is required | PASS (planned artifact) |

No constitutional violations. `Complexity Tracking` stays empty.

**Post-design re-check (Phase 1)**: still PASS. The `btree_gist` path is best-effort inside one
migration; the fallback keeps the "at most one open-ended interval per user" invariant, and the report index is a
plain GiST range index that needs no extension.

## Project Structure

### Documentation (this feature)

```text
specs/031-user-plan-periods/
├── plan.md                          # This file
├── research.md                      # Phase 0 output
├── data-model.md                    # Phase 1 output
├── quickstart.md                    # Phase 1 output
├── contracts/
│   └── distribution-api-delta.md    # Phase 1 output (JSON + gRPC delta)
├── frontend-guide.md                # required by the constitution (written with tasks/implementation)
├── checklists/requirements.md
├── deferred.md                      # optional: out-of-scope findings during implementation
└── tasks.md                         # Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
services/account/
├── database/migrations/
│   └── 20261020150000_add_user_plan_periods.sql     # tables + indexes + adaptive constraint + backfill + down
├── dto/subscription/dashboard_dto.go                # result DTO aliases
└── internal/modules/subscription/
    ├── domain/
    │   ├── entities/user_plan_period.go             # new entities
    │   ├── entities/table_name.go                   # registrations
    │   └── repositories/
    │       ├── user_plan_period_repo.go             # new interface + read model
    │       └── user_subscription_repo.go            # distribution method removed
    ├── application/
    │   ├── interface/usecase/admin_stats_usecase.go # result type
    │   ├── usecase/admin_stats/admin_stats_uc.go    # interval repo + completeness calc
    │   └── usecase/subscription/
    │       ├── plan_change_publish_helper.go        # calls recorder after event insert
    │       ├── plan_period_recorder.go              # new: transition rules
    │       └── plan_period_recorder_{test,idempotency_test,integration_test}.go
    ├── infrastructure/persistence/
    │   ├── user_plan_period_repo_impl.go            # new GORM repo
    │   └── user_plan_period_repo_impl_test.go
    ├── application/provider.go                      # recorder provider
    └── infrastructure/provider.go                   # repo provider
services/account/contract/subscription/
├── dashboard_reader.go                              # interface returns result DTO
└── transport/{to_proto.go,mapper.go,mapper_test.go} # mapping + round-trip tests

contracts/subscription/v1/subscription.proto         # enum + two response fields; buf-gen output
contracts/subscription/v1/client/dashboard_reader_client.go

services/commerce/internal/modules/dashboard/
├── application/dto/admin_dashboard_dto.go           # optional dataAvailableFrom/dataCompleteness
├── application/usecase/admin_dashboard_uc_impl.go   # pass-through
└── presentation/handler/admin_dashboard_handler.go  # swagger annotation

docs/api/dashboard/dashboard-admin-api.md            # client docs
docs/development/technical-notes.md                   # operator query + measurements
docs/system-design/architecture.md                    # §VII table/FK inventory
specs/029-admin-observability/frontend-guide.md       # warning narrowed to the cutover date
specs/029-admin-observability/deferred.md             # D-06 → RESOLVED on completion
```

**Structure Decision**: the new data lives in the Account service because `user_subscriptions` and
`user_subscription_events` are Account-owned (`029`/`030` ownership contract). Commerce only forwards
the period and renders the response; it gains no new repository.

## Design Decisions

### D1 — Write path: one recorder call inside `PlanChangePublisher`

Every plan mutation already goes through `PlanChangePublisher.PublishPlanChange`
(`services/account/internal/modules/subscription/application/usecase/subscription/plan_change_publish_helper.go:66`)
inside the caller's transaction. That makes it the single place that sees the event type, the effective
time, the updated subscription, and the target plan. The publisher gains a `PlanPeriodRecorder`
collaborator and calls it **after** the event insert succeeds:

- retries are already idempotent — an existing `event_key` returns early and never reaches the recorder;
- a recorder failure rolls back the business mutation with the event, so no orphan intervals;
- no production call-site changes, so no path can forget to record (the same structural argument as `024 T045`).

The recorder signature is `Record(ctx, event, sub, plan)`: `plan` is needed because the subscription row
does not carry the plan name, and `FR-001` requires a name snapshot. `current` means the user's latest
interval by `valid_from` (read `FOR UPDATE`), not only the row whose `valid_to` is NULL: a finite paid
holding already carries its future expiry.

Per event type:

| Event | Interval action |
|---|---|
| `ACTIVATED`, `UPGRADED`, `OVERLAID` where the target plan differs from the current interval | close at `closeAt = min(current.valid_to, event.EffectiveAt)`, then open a new interval from `event.EffectiveAt` |
| `ACTIVATED` where the target plan equals the current interval's plan (admin restore) | no-op — the plan and window did not change (`FR-003`) |
| `ACTIVATED`, `UPGRADED`, `OVERLAID` where `closeAt <= current.valid_from` | replace the current row in place (new plan fields, `valid_from = max(event.EffectiveAt, current.valid_from)`, `valid_to = sub.ExpiresAt`) instead of creating a zero-length interval |
| `EXTENDED` | move `valid_to` of the current interval to `sub.ExpiresAt`; no new row |
| `SUBSCRIPTION_DOWNGRADED_TO_FREE` | close at `closeAt`; open the free interval at that same instant (`FR-005`) |
| `LIFETIME_FALLBACK_RESTORED` | same as downgrade; `event.EffectiveAt` is the previous expiry |
| `AUTO_RENEW_UPDATED` | no-op — the plan did not change |

Rules that follow:

- `valid_from` of a new row is `max(event.EffectiveAt, current.valid_from)` — clock skew cannot produce a
  zero or negative interval, and the exclusion constraint rejects anything still inconsistent.
- If there is no current interval (the user predates the cutover and the backfill did not cover them), the
  recorder inserts the new interval and does not invent a close.
- A lapse (paid interval expired before a later purchase) leaves a gap between `valid_to` and the new
  `valid_from`. `FR-002` only requires contiguity for continuous switches; a lapse is not a switch, and
  a gap is the honest answer.
- The recorder reads the current row `FOR UPDATE`. The admin-restore path does not take the subscription
row lock, so the interval row lock plus the non-overlap constraint serializes writes. Additionally,
any event whose metadata carries `source=admin_restore` is an absolute no-op in the recorder: a repair
changes AI grants, not the subscription window, and must not overwrite a newer interval that a
concurrent purchase created between the restore's read and its write.
- `EXTENDED` for a user without a row (should not happen after backfill) falls back to inserting the
  current window from `sub.StartedAt`.

### D2 — Non-overlap guarantee: adaptive constraint

Primary: `EXCLUDE USING gist (user_id WITH =, tstzrange(valid_from, valid_to, '[)') WITH &&)` plus
`CHECK (valid_to IS NULL OR valid_to > valid_from)`. `tstzrange` maps a NULL end to infinity, so free
and lifetime intervals are open-ended.

`btree_gist` is required for the uuid equality operator. The migration attempts extension creation and
constraint creation inside one `DO` block whose exception handler catches only `insufficient_privilege`,
`undefined_file` and `feature_not_supported` (anything else is re-raised); if the extension is unavailable
or the role lacks the privilege, it creates a partial unique index on `(user_id) WHERE valid_to IS NULL`
instead and raises a notice. Both variants keep the "at most one open-ended interval" invariant; only the exclusion
variant guarantees full non-overlap, and the write path's row lock plus the recorder's close rules are
the primary control either way. This keeps the schema portable without a code fork (constitution VIII).

Independent of that choice, a GiST index on the range expression serves the report:

```sql
CREATE INDEX idx_user_plan_periods_range
    ON subscription.user_plan_periods
    USING gist (tstzrange(valid_from, valid_to, '[)'));
```

It needs no extension. `research.md` records the measurement and the fallback.

### D3 — Backfill as part of the schema migration

The migration inserts the current window of every `user_subscriptions` row (a preflight `DO` block aborts
if a finite row has `expires_at <= started_at` rather than skipping it)
(`valid_from = started_at`, `valid_to = expires_at`, snapshot of `current_plan_*`, join
`subscription_plans` for `plan_name`, `source_event_key = 'BACKFILL:<user_id>'`) and records the
cutover in `subscription.plan_history_meta('cutover', now())`. This is the only history that can be
derived without guessing; older windows stay unknown (`FR-013`, out of scope). A `ON CONFLICT DO
NOTHING` on the event key makes a re-run harmless. No separate job, no deploy-order coupling. The
migration measures its own duration in a `RAISE NOTICE` for the deploy log.

### D4 — Query, bounds, and completeness

All predicates use half-open ranges and a zero bound means "unbounded" (`FR-008`, `FR-012`):

```sql
SELECT plan_code, COUNT(DISTINCT user_id) AS user_count
FROM subscription.user_plan_periods
WHERE tstzrange(valid_from, valid_to, '[)') && tstzrange($from, $to, '[)')
GROUP BY plan_code
ORDER BY user_count DESC
```

- both bounds zero → snapshot: `tstzrange(valid_from, valid_to, '[)') @> now()`;
- `to` zero → the range parameter's upper bound is NULL (infinity) **and** `valid_from <= now()` is
  added, so future-dated rows are not counted (`FR-008`/A2);
- `from` zero → the range parameter's lower bound is NULL (infinity);
- `COUNT(DISTINCT user_id)` per plan so a user who left and returned to the same plan counts once
  (`FR-010`);
- `totalUsers` remains the sum of buckets and percentages stay relative to it (`FR-011`, clarified).

Completeness is computed in the use case from `from`, `to`, and the cutover (`FR-012`), treating a zero
bound as infinity, never as the zero time:

| From | To | Result |
|---|---|---|
| zero | zero | `complete` (current snapshot) |
| any | provided and `to <= cutover` | `unavailable` |
| zero or `from < cutover` | `to > cutover` or zero | `partial` |
| `from >= cutover` | any | `complete` |

When the result is `partial`, the use case raises the query's `from` to the cutover (or to the cutover
when `from` is zero), so the numbers describe exactly the period the label claims. When the cutover row
is missing, a period request returns `dataCompleteness = unavailable` (with a logged warning) and omits
only `dataAvailableFrom` (the date is unknown); the snapshot stays `complete` and may omit both. A client
must still see the `unavailable` signal.

### D5 — Wire contract: additive fields, new enum

`SubscriptionDistributionList` gains
`google.protobuf.Timestamp data_available_from = 2` and
`DistributionDataCompleteness data_completeness = 3` (values `COMPLETE`, `PARTIAL`, `UNAVAILABLE`).
The JSON response gains `dataAvailableFrom` as a `*string` formatted `YYYY-MM-DD` in VN time (not a
timestamp) and `dataCompleteness` as a lowercase string. Both are omitted when absent; no existing
field changes name, type, or meaning. The in-process and gRPC adapters and the Commerce use case map
them through unchanged. The response shape is covered by a compatibility test that decodes the old
JSON shape (without the two fields) and by a proto test that unknown fields are tolerated.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Recorder closes an interval at the wrong instant and silently corrupts history | unit tests per event type with fixed clocks; integration tests assert contiguous windows for a four-step lifecycle, same-instant replacement, admin restore no-op, and lapse gaps |
| Backfill on a large table extends the first replica's startup | single `INSERT ... SELECT` of one row per user; measure and log duration in the migration; quickstart records the number |
| Recorder adds a dependency to `PlanChangePublisher`, breaking test constructors | task updates every constructor call site with a no-op recorder and runs the affected packages |
| `CREATE EXTENSION` privilege differs per environment | adaptive `DO` block with partial-unique fallback; both variants tested on the scratch DB |
| Report query scans most intervals for wide windows | GiST range index on `tstzrange(valid_from, valid_to)`; measurement at 200k (SC-004) plus an informational 1M run |
| Adding fields to the proto breaks older consumers | additive fields only; compatibility test for old JSON/proto; `make buf-gen` drift check |
| Clock skew makes `valid_from` earlier than the currently current interval's start | recorder clamps `valid_from`; zero-length case replaces the current row; exclusion constraint rejects residual overlap |
| A missing cutover row silently produces "complete" zeros | data-model defines `unavailable` + warning; integration test for the absent-row case |

## Complexity Tracking

Empty — no constitutional violations.
