# Deferred — 025 Signup Campaign Grant

Two defects discovered while closing feature 025. **Neither is a task of 025**:
both are latent faults in the shared integration-test harness, found because
025 was the first feature to run against it.

Neither blocks closing 025. Both were recorded here rather than left as
unchecked boxes in `tasks.md`, where an unchecked box would have claimed 025 is
unfinished — which it is not.

- **`[X] T118` and `[X] T119` were ticked on close-out and are recorded below as D-01 and D-02.**
- **D-01 weakens the evidence of `[X] T116`.** Read D-01 before trusting T116.

---

## D-01 — `AutoMigrate` in the webhook test harness cannot run on a migration-built schema

- **Vấn đề**: `openWebhookTestDB` in
  `services/account/internal/modules/subscription/application/usecase/webhook/`
  calls GORM `AutoMigrate` against a schema that the Goose migrations already
  created. It dies immediately on
  `constraint "uni_subscription_plans_slug" ... does not exist`.

  Root cause: the entity declares `uniqueIndex`, so GORM expects an index named
  `uni_subscription_plans_slug`, while the migration created a UNIQUE CONSTRAINT
  named `subscription_plans_slug_key`. The two names disagree and `AutoMigrate`
  issues destructive DDL to "fix" it before failing.

  Fix direction: use a controlled `MigrateColumn`, or drop `AutoMigrate`
  entirely and require the schema to be the one the migrations built.

- **Vì sao ngoài phạm vi 025**: this is the shared integration-test harness, not
  campaign logic. 025 changed no entity, no migration and no harness code. Any
  future feature needing a Postgres-backed integration test hits this first, so
  the fix belongs to the feature that introduces such a harness — or to a small
  dedicated cleanup feature.

- **Gỡ bằng cách nào**: pick the fix direction above and change `openWebhookTestDB`.
  Roughly a one-file change plus a test run against a migration-built schema.

- **Ảnh hưởng tới task đã tick**: **`[X] T116`.**

  T116's whole purpose was to supply the FR-095 "mua gói" path evidence, and it
  did — on a schema `AutoMigrate` built, which is **not** the production shape.
  The constraint name mismatch above proves the two shapes differ; any further
  divergence was never compared, because nothing compared them.

  So T116 is ticked, but its evidence is **conditional**: it demonstrates the
  PayOS purchase branch of FR-095, on a schema that resembles production rather
  than matching it. FR-095 asks for six paths and the "mua gói" count should be
  read with that caveat until D-01 is fixed and the test is re-run unchanged.

  Not checked: whether the auto-built schema diverges in any way that would change
  this path's outcome. Establishing that is exactly what fixing D-01 would show.

---

## D-02 — `cleanupWebhookTables` truncates without a WHERE clause, against a table with a bare FK

- **Vấn đề**: `cleanupWebhookTables` deletes unrestricted (`WHERE 1 = 1`) from
  `user_subscriptions`, `subscription_plans`, `user_wallets`, `wallet_statements`
  and `deposit_transactions`. That is how T116 was made safe to run at all —
  requiring developer consent — rather than how it should have been safe.

  Separately, `subscription_renewal_attempts` is **not** in the cleanup list but
  holds FK `expected_plan_id → subscription_plans(id)` with **no** `ON DELETE`.
  The table is currently empty so nothing fails, but a single renewal-attempt row
  makes `DELETE FROM subscription_plans` violate that FK and kill the test before
  it starts.

- **Vì sao ngoài phạm vi 025**: the harness is shared. 025's run happened to be
  the first one on a database with a non-empty `subscription_renewal_attempts`
  table, which is what exposed the second half.

- **Gỡ bằng cách nào**: scope every delete by account, following the existing
  pattern at `eager_provision_postgres_integration_test.go:438`
  (`DELETE ... WHERE user_id = ?`). Add `subscription_renewal_attempts` to the
  list, ordered so children go before parents. Once that is done, bulk-test runs
  stop needing an explicit consent gate — which is worth doing, because the
  consent gate is what let the Brevo incident in 025 go unnoticed in the first
  place.

- **Ảnh hưởng tới task đã tick**: none. T116 ran on a disposable schema, so the
  unrestricted delete destroyed only schema-local data. The hazard is to the
  next run that points this harness at a shared database.