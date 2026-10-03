-- +goose Up
-- +goose StatementBegin
-- ============================================================================
-- Feature 026 T007 (FR-001 / FR-010..FR-015 / FR-020 / FR-021 / FR-031 / FR-032 /
-- FR-055) — the campaign registry.
--
-- WHY THIS EXISTS. Feature 025 made a campaign "data, not code", but kept that
-- data in a YAML file compiled into the image (configs/config*.yaml `campaigns`),
-- so opening one still required a commit, a tag and a deploy, and closing one
-- early had no path at all. 026 moves the registry into the database, which is
-- the only place it can live and be changed without shipping new binaries
-- (spec.md §2.2). The `campaigns` block is deleted from both YAML files in the
-- SAME release cycle — a binary reading this table must not also read the file.
--
-- WHY THE DATABASE IS THE ONLY SOURCE. `quota` and `reserve` decide the hard
-- ceiling, and every historical claim froze which part of that ceiling it drew
-- from (`campaign_claims.grant_bucket`). Two sources that disagree make every
-- count wrong against BOTH, and the reconciliation rule between them is exactly
-- where the immutability of a live campaign is broken (spec.md Câu 1).
--
-- WHY NO FOREIGN KEY FROM campaign_claims / campaign_sweep_runs. Those
-- `campaign_code` columns are historical text keys with no foreign key by
-- design (FR-091), and the ledger may name a code that is not in the registry.
-- Adding a constraint would require reconstructing attributes for every
-- historical code, which this migration cannot do faithfully, and it could
-- block the migration on live data. Integrity is kept instead by there being NO
-- delete path at all: campaigns are created, updated and closed, never removed
-- (FR-005, FR-034).
--
-- WHY quota/reserve/plan_slug ARE IMMUTABLE ONCE GRANTS EXIST. Each claim froze
-- the budget part it came from and the FR-030 watermark is tied to the quota at
-- fill time. Raising quota on a live campaign would move that watermark and
-- corrupt every count derived from it. The trigger below enforces this in the
-- DATA layer, because application discipline does not bind a hand-written UPDATE
-- (FR-021, research.md R3).
--
-- WHY TIMESTAMPTZ AND NOT THE RFC3339 STRINGS 025 STORED. A TIMESTAMPTZ column
-- carries the instant unambiguously; the zone-less string 025 rejected at startup
-- is unrepresentable here, so FR-013's "must be an instant with a zone" becomes a
-- column type rather than a startup check (FR-010..FR-015).
--
-- WHY `version` IS HERE. It is not a business attribute: it is the
-- optimistic-concurrency token of FR-055, so a stale PATCH or close is refused
-- with 412 instead of silently overwriting another administrator's change.
-- Precedent in this schema: `user_subscriptions.version`.
--
-- LOAD-BEARING ORDER: this migration MUST run BEFORE the new binary. A binary
-- reading a table that does not exist yet would grant nothing to anyone, and
-- "grants nothing" is the failure nobody notices until a launch day (FR-090).
-- ============================================================================

CREATE TABLE IF NOT EXISTS subscription.campaigns (
    code        VARCHAR(64)  PRIMARY KEY,
    plan_slug   VARCHAR(100) NOT NULL,
    quota       INTEGER      NOT NULL,
    reserve     INTEGER      NOT NULL DEFAULT 0,
    starts_at   TIMESTAMPTZ  NOT NULL,
    ends_at     TIMESTAMPTZ,
    -- NULL = not force-closed. Non-NULL is the operator's closure instant
    -- (FR-031). Distinct from being past `ends_at`: an open-ended campaign can be
    -- closed early, and a closed campaign is asleep wherever the clock is
    -- (FR-032).
    closed_at   TIMESTAMPTZ,
    -- FR-055: incremented by every successful write, so a stale PATCH or close is
    -- refused rather than overwriting. Never incremented by a no-op.
    version     INTEGER      NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),

    -- FR-011: restricted alphabet so a code that ends up in a queue name, an
    -- audit payload or a file name cannot break path traversal. PRIMARY KEY above
    -- already gives NOT NULL and uniqueness, which is the third of 025's seven
    -- startup rules and the only one the database can enforce absolutely.
    CONSTRAINT chk_campaigns_code_format
        CHECK (code ~ '^[a-z0-9][a-z0-9-]{2,63}$'),
    -- FR-014: non-blank at write time. Whether the plan EXISTS and is ACTIVE is
    -- NOT checked here — loading configuration must not reach into the database
    -- (FR-003, constitution §I/§VIII), and a wrong slug must never be able to
    -- crash a service. A grant against a missing plan is skipped with
    -- `plan_unavailable` at RUN time, as 025 decided.
    CONSTRAINT chk_campaigns_plan_slug_not_blank
        CHECK (btrim(plan_slug) <> ''),
    -- FR-012: zero would be a campaign that grants nothing, i.e. meaningless
    -- rather than invalid.
    CONSTRAINT chk_campaigns_quota
        CHECK (quota >= 1),
    CONSTRAINT chk_campaigns_reserve
        CHECK (reserve >= 0),
    -- FR-013: an open-ended campaign has no ends_at; when it is set it must be
    -- strictly after starts_at.
    CONSTRAINT chk_campaigns_window
        CHECK (ends_at IS NULL OR ends_at > starts_at)
);

-- FR-040: the selection reads awake campaigns newest-first, so the sort axis gets
-- an index. The table is a handful of rows for the whole product lifetime, so one
-- index is enough and a composite one for the window terms would be dead weight
-- (spec.md Câu 6).
CREATE INDEX IF NOT EXISTS ix_campaigns_starts_at
    ON subscription.campaigns (starts_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
-- FR-020 / FR-021: a live campaign's budget is immutable.
--
-- WHY A TRIGGER AND NOT A CHECK. A CHECK is a per-row constraint and cannot see
-- subscription.campaign_claims, so it cannot express "immutable only once a grant
-- exists". The application layer refuses first with a friendly Vietnamese 422;
-- this trigger is the backstop that makes a hand-written SQL UPDATE impossible
-- (FR-021, FR-062). Plain PostgreSQL, no vendor feature (constitution §VIII).
--
-- WHY plan_slug IS IN THE SAME GROUP. campaign_claims.plan_slug is a COPY of the
-- plan each grant actually received (data-model.md §2.2); changing the plan
-- mid-campaign would leave the ledger describing a mixture the registry no longer
-- explains.
--
-- WHY `IS DISTINCT FROM` RATHER THAN `<>`. Both columns are NOT NULL today, so
-- `=` would behave identically — but the NULL-safe form stays correct if a future
-- migration ever relaxes a column, and it is the one that cannot be silently
-- wrong (research.md R15).

CREATE OR REPLACE FUNCTION subscription.campaigns_guard_immutable_budget()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF (NEW.quota, NEW.reserve, NEW.plan_slug)
           IS DISTINCT FROM (OLD.quota, OLD.reserve, OLD.plan_slug)
       AND EXISTS (
           SELECT 1 FROM subscription.campaign_claims
           WHERE campaign_code = OLD.code
       )
    THEN
        RAISE EXCEPTION
            'campaign % has grants; quota, reserve and plan_slug are immutable (026 FR-020/FR-021)',
            OLD.code
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_campaigns_immutable_budget
    BEFORE UPDATE ON subscription.campaigns
    FOR EACH ROW
    EXECUTE FUNCTION subscription.campaigns_guard_immutable_budget();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Reverse of Up, in reverse order.
--
-- NO campaign_claims ROW IS LOST. There is no foreign key from the ledger to this
-- table (FR-091), so dropping it orphans nothing: the claim rows keep their
-- `campaign_code` text and are readable, exactly as they were before this
-- migration ran (FR-092). Rolling the schema back therefore costs the registry,
-- not a single grant.

DROP TRIGGER IF EXISTS trg_campaigns_immutable_budget ON subscription.campaigns;
DROP FUNCTION IF EXISTS subscription.campaigns_guard_immutable_budget();

DROP INDEX IF EXISTS subscription.ix_campaigns_starts_at;

DROP TABLE IF EXISTS subscription.campaigns;
-- +goose StatementEnd