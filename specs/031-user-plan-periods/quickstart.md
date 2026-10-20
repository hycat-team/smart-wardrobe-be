# Quickstart — Lịch sử gói theo khoảng hiệu lực

**Feature**: `031-user-plan-periods` | **Date**: 2026-10-20

Run guide only. Implementation details live in `plan.md`, `data-model.md`, and `tasks.md`.

## Prerequisites

- Docker dev stack reachable (`postgres-closy-dev` container), Go toolchain, Goose installed via
  `make install-goose`.
- Recommended shell on Windows: `make SHELL=C:/Windows/System32/cmd.exe <target>`; without it `sh.exe`
  expands `$files` in the lint target and breaks the command.
- No email path is exercised by this feature; `EMAIL_ENABLED` is irrelevant.

## 1. Migration round-trip on a temporary database

Never run this against the shared dev database. Create a scratch database, seed the base schemas, then
apply each service's migrations.

```powershell
docker exec -e PGPASSWORD=123456 postgres-closy-dev psql -U admin -d smart_wardrobe_db -c "DROP DATABASE IF EXISTS verify_plan031;"
docker exec -e PGPASSWORD=123456 postgres-closy-dev psql -U admin -d smart_wardrobe_db -c "CREATE DATABASE verify_plan031;"
Get-ChildItem database/init/*.sql | Sort-Object Name | ForEach-Object {
  Get-Content $_.FullName -Raw | docker exec -i -e PGPASSWORD=123456 postgres-closy-dev psql -U admin -d verify_plan031 -v ON_ERROR_STOP=1
}
make migrate-up svc=account DB_NAME=verify_plan031
make migrate-up svc=commerce DB_NAME=verify_plan031
make migrate-up svc=ai DB_NAME=verify_plan031
```

Expected: the new migration (`*_add_user_plan_periods.sql`) applies inside one transaction, and
`goose` reports the new version as the latest for account.

Verify schema, backfill, and cutover:

```powershell
docker exec -e PGPASSWORD=123456 postgres-closy-dev psql -U admin -d verify_plan031 -c "\d subscription.user_plan_periods"
docker exec -e PGPASSWORD=123456 postgres-closy-dev psql -U admin -d verify_plan031 -c "SELECT count(*) FROM subscription.user_plan_periods;"
docker exec -e PGPASSWORD=123456 postgres-closy-dev psql -U admin -d verify_plan031 -c "SELECT * FROM subscription.plan_history_meta;"
```

Expect: interval count equals `SELECT count(*) FROM subscription.user_subscriptions;` (one row per
user), no row with `valid_to <= valid_from`, and one `cutover` row whose `started_at` is the migration
timestamp.

Check the overlap guard. On the exclusion variant it must fail with `23P01`; on the fallback
variant, inserting a second open interval for the same user must fail with `23505`:

```sql
INSERT INTO subscription.user_plan_periods
  (user_id, subscription_plan_id, plan_code, plan_name, tier_rank, plan_kind, valid_from, valid_to, source_event_key)
SELECT user_id, subscription_plan_id, plan_code, plan_name, tier_rank, plan_kind,
       valid_from, valid_to, 'OVERLAP-PROBE'
FROM subscription.user_plan_periods WHERE valid_to IS NULL LIMIT 1;
```

To exercise the fallback deliberately, run the migration on a second scratch database as a role that
is neither the database owner nor a member of a role with `CREATE` on the database (so
`CREATE EXTENSION` fails). Expected: the migration completes with a notice, `ux_user_plan_periods_open`
exists, the exclusion constraint does not, and a second open interval fails with `23505`. Record the
variant and the error code in the verification notes.

Rollback check (destroys the scratch history; never on a shared database):

```powershell
make migrate-down svc=account DB_NAME=verify_plan031
make migrate-up   svc=account DB_NAME=verify_plan031
```

After the cycle the tables must exist again, `plan_history_meta` has a fresh cutover row, and the
interval count is back to one per user.

## 2. Gates

```powershell
make migration-check
make SHELL=C:/Windows/System32/cmd.exe lint
$env:TEST_DATABASE_URL="postgres://admin:123456@localhost:5432/verify_plan031?sslmode=disable&search_path=public,identity,wardrobe,fashion,brand,subscription,aiexecutor,dashboard,shared,community"
$env:TEST_DATABASE_ALLOW_REGISTRY_ROLLBACK="true"
make test-db-integration
```

The integration harness requires a migration-built schema (025/D-01, 026/D-06); the scratch database
above satisfies that. `make test-db-integration` runs with `-p 1 -count=1`.

## 3. End-to-end scenarios

Run the feature's integration tests by name from the Account module:

```powershell
Set-Location services/account
go test -race -count=1 -run 'PlanPeriod|DistributionByPeriod' ./internal/modules/subscription/... -v
Set-Location ../..
```

The distribution tests set `subscription.plan_history_meta.started_at` to a fixed date in the past
(for example `2026-01-01`) so that the fixture periods are after the cutover. Without that, a period
named in the examples below is before the deploy date and correctly returns `unavailable`.

What each scenario must show:

| Scenario | Expected buckets |
|---|---|
| Premium all September, downgraded 01/10 09:00; report September | premium = 1, free = 0, user present |
| Upgrade A→B on 15/09; report September | A = 1 **and** B = 1; `totalUsers` counts the user twice |
| Renewal of the same plan on 10/09 | still one interval; plan bucket unchanged; `valid_to` moved |
| Free account for the whole period | free = 1; interval has `valid_to IS NULL` |
| Lifetime, overlaid with a finite plan, then restored | lifetime closed at overlay, finite closed at expiry, lifetime open again at that expiry |
| `fromDate` only / `toDate` only | same numbers as the equivalent two-sided window; no future or long-expired rows |
| Period before the cutover | `dataCompleteness = unavailable`, empty `distribution` |
| Period straddling the cutover | `dataCompleteness = partial`, buckets computed after the cutover |

## 4. API smoke check

Boot Account and Commerce with the scratch database (or the normal dev stack after migrating it), then:

```powershell
curl.exe -H "Authorization: Bearer <admin-token>" `
  "http://localhost:<port>/api/v1/admin/dashboard/subscriptions/distribution?fromDate=2026-10-01&toDate=2026-10-31"
```

Expect the existing `distribution`/`totalUsers` shape plus `dataAvailableFrom` and
`dataCompleteness`. A period before the cutover returns `dataCompleteness: "unavailable"` and an empty
list, not a zero-valued chart.

## 5. Cleanup

```powershell
docker exec -e PGPASSWORD=123456 postgres-closy-dev psql -U admin -d smart_wardrobe_db -c "DROP DATABASE IF EXISTS verify_plan031;"
```

`btree_gist` is left installed in the cluster; the down migration intentionally does not drop it.
