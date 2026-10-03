-- +goose Up
-- +goose StatementBegin
-- ============================================================================
-- Feature 026 T008 (FR-005 exception / FR-090) — seed the campaign that is
-- already running on production.
--
-- This is the ONE deliberate exception to FR-005 ("in operation only the admin
-- API writes this table"): a migration INSERT, once, idempotent.
--
-- WHY THE VALUES ARE COPIED VERBATIM FROM configs/config.production.yaml.
-- `launch-2026-10` has been running with these six values and has roughly 55 real
-- rows in subscription.campaign_claims. The registry must say exactly what the
-- file said, or the switch from configuration to database would change a meaning
-- nobody asked to change (spec.md Câu 5, US6/SC-004). `ends_at` stays NULL: the
-- campaign is open-ended, and NULL is not "the zero time".
--
-- WHY THIS RUNS BEFORE THE NEW BINARY, AND AFTER THE SCHEMA MIGRATION. Three
-- steps in a fixed order (T007 → T008 → removing the YAML block):
--
--   1. schema  — creates the table
--   2. seed    — inserts the campaign that already exists in the file
--   3. binary  — stops reading the file
--
-- Any other order loses a running promotion. Seeding after the binary ships means
-- a window in which the binary reads an empty registry and grants nothing — the
-- failure nobody notices until a launch day. Removing the file first means a
-- campaign nobody has migrated.
--
-- WHY IT IS SAFE ON EVERY ENVIRONMENT INCLUDING DEV. The INSERT is idempotent, so
-- a developer who already created `launch-2026-10` through the admin API keeps
-- their row untouched, and it never touches subscription.campaign_claims (FR-091).
-- ============================================================================

INSERT INTO subscription.campaigns (code, plan_slug, quota, reserve, starts_at, ends_at)
VALUES ('launch-2026-10', 'premium-monthly', 50, 5, '2026-10-02T00:00:00+07:00', NULL)
ON CONFLICT (code) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Reverse of Up: remove the row this migration inserted, and ONLY that row.
--
-- Scoped to the seeded code rather than truncating the table, so rolling this
-- back after an operator has created further campaigns through the admin API
-- cannot delete their work. `campaign_claims` is untouched: no foreign key points
-- at this row (FR-091), so the ~55 real claims survive the rollback intact
-- (FR-092).

DELETE FROM subscription.campaigns WHERE code = 'launch-2026-10';
-- +goose StatementEnd