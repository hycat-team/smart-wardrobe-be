-- +goose Up
-- +goose StatementBegin
-- ============================================================================
-- Feature 025 T016 (FR-022 / FR-024 / FR-025 / FR-030 / FR-031) — campaign claim
-- ledger.
--
-- WHY THIS EXISTS. A campaign is a data-driven promotion (FR-002), so the
-- "first N accounts" rule needs a durable counter. The counter and the grant must
-- commit together, which is why they are the SAME row rather than a count in one
-- table and a claim in another — two sources of truth that have to be reconciled.
--
-- WHY `user_id` IS UNIQUE ALONE, NOT (campaign_code, user_id). FR-012 gives an
-- account at most one participation across the WHOLE campaign mechanism, so a later
-- campaign must skip an account an earlier one already granted. A composite key
-- would let that account collect a second promotional plan, and each grant would
-- add to its term. `campaign_code` is kept for attribution and per-campaign
-- counting, not as part of the key.
--
-- WHY `grant_bucket` IS A COLUMN HERE, AND WHY NOT `sequence_no`. The campaign
-- splits its budget into a main part and a reserve part, and the admin query
-- (FR-065) has to report them separately. To count them from one column, each row
-- must say which part it came from. Deriving it from `sequence_no` (rows 1..quota
-- are main, the rest are reserve) would tie the meaning of historical rows to the
-- quota in the YAML file: raising `quota` from 50 to 60 on a live campaign would
-- silently reclassify every earlier claim, corrupt the admin numbers, and MOVE the
-- FR-030 watermark. A frozen enum keeps the past correct no matter how the config
-- changes afterwards. Rejected alternative: deriving it from `NOT fills_main_quota`,
-- which is worse still — that flag marks exactly ONE row, so its negation counts
-- the reserve as main and the reserve branch then fails
-- `ux_campaign_claims_watermark` instead of falling through (research.md R2b).
--
-- WHY `fills_main_quota` IS SEPARATE AND NOT REPLACED BY `grant_bucket`. FR-030
-- freezes the priority watermark on the claim that FILLED the main budget, so a
-- later comparison always has the same answer regardless of processing order.
-- That is "the 50th claim", not "a main claim" — 49 other rows are also main. The
-- partial unique index below makes "at most one watermark per campaign" a
-- data-layer guarantee, so two workers running concurrently cannot freeze two
-- different watermarks.
--
-- `expires_at` IS NOT NULL. `subscription.user_subscriptions` carries a CHECK that
-- a finite plan (plan_kind = 1) must have an expiry, and the granted plan is
-- finite. Allowing NULL here would let a claim record a grant that could never
-- have been committed.
--
-- `registered_at` snapshots the account creation instant AT GRANT TIME so FR-031
-- can compare it against the watermark without reading the account table — the
-- campaign module is not allowed to read another module's table (constitution §II).
--
-- NO total cap as a CHECK, ON PURPOSE. `quota` and `reserve` live in a YAML file,
-- not in the database, so a CHECK against a constant would be wrong the first time
-- the config is edited. The ceiling is held by the rules in data-model.md §6.2 —
-- each branch stays under its OWN ceiling — plus the advisory lock, so the total
-- needs no comparison at all.
-- ============================================================================

CREATE TABLE IF NOT EXISTS subscription.campaign_claims (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL
                                  CONSTRAINT fk_campaign_claims_user
                                  REFERENCES identity.users(id) ON DELETE CASCADE,
    campaign_code     VARCHAR(64) NOT NULL,
    plan_slug         VARCHAR(100) NOT NULL,
    plan_id           UUID        NOT NULL,
    -- Snapshot of users.created_at taken when the grant was written, so the
    -- FR-031 comparison against the frozen watermark needs no account read.
    registered_at     TIMESTAMPTZ NOT NULL,
    expires_at        TIMESTAMPTZ NOT NULL,
    granted_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- FR-022: which part of the campaign budget this grant came from — main or
    -- reserve. Written at grant time, NEVER inferred from the configured quota.
    grant_bucket      VARCHAR(16) NOT NULL
                                  CONSTRAINT chk_campaign_claims_bucket
                                  CHECK (grant_bucket IN ('main', 'reserve')),
    -- FR-030: exactly one row per campaign carries TRUE, the grant that filled the
    -- main budget. DO NOT count the main part with this column — see the note on
    -- grant_bucket above (research.md R2b).
    fills_main_quota  BOOLEAN     NOT NULL DEFAULT FALSE,

    CONSTRAINT ux_campaign_claims_user UNIQUE (user_id)
);

-- FR-030: at most one frozen watermark per campaign, enforced in the data layer.
CREATE UNIQUE INDEX IF NOT EXISTS ux_campaign_claims_watermark
    ON subscription.campaign_claims (campaign_code)
    WHERE fills_main_quota;

-- FR-022: every quota count reads through (campaign_code, grant_bucket), so this
-- index lets those reads scan only the main part or only the reserve part instead
-- of the whole table.
CREATE INDEX IF NOT EXISTS ix_campaign_claims_campaign_bucket
    ON subscription.campaign_claims (campaign_code, grant_bucket);

-- FR-062: an operator look-up of "who was skipped, and why" needs to walk one
-- campaign newest-first.
CREATE INDEX IF NOT EXISTS ix_campaign_claims_campaign_granted
    ON subscription.campaign_claims (campaign_code, granted_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Reverse order of Up. The indexes live inside the table, so dropping the table
-- takes them with it; they are named explicitly anyway so the rollback reads as
-- the mirror of the statements above rather than as a single opaque drop.

DROP INDEX IF EXISTS subscription.ix_campaign_claims_campaign_granted;
DROP INDEX IF EXISTS subscription.ix_campaign_claims_campaign_bucket;
DROP INDEX IF EXISTS subscription.ux_campaign_claims_watermark;

DROP TABLE IF EXISTS subscription.campaign_claims;
-- +goose StatementEnd
