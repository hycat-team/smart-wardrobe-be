-- +goose Up
-- +goose StatementBegin
-- ============================================================================
-- Feature 031 — validity intervals for subscription distribution.
--
-- WHY THIS EXISTS. `subscription.user_subscriptions` is one row per account and
-- every plan change overwrites `started_at`/`current_plan_code`. The admin
-- distribution report therefore could not answer "which plan did this user hold
-- in September" once the user changed plan in October: the previous window was
-- gone, so the user disappeared from every bucket. `user_plan_periods` keeps one
-- row per continuous holding instead, so a period query can answer exactly and
-- every plan change has a place to append to (031 FR-001/FR-007).
--
-- WHY THE NON-OVERLAP RULE IS ADAPTIVE. `btree_gist` is needed for the uuid
-- equality operator in an EXCLUDE constraint, but a managed PostgreSQL role may
-- not be allowed to create extensions. The preferred shape is the full
-- exclusion constraint; when the extension is unavailable the migration falls
-- back to a partial unique index that still guarantees "at most one OPEN
-- interval per user". The recorder's `FOR UPDATE` on the open row plus its
-- close rules are the primary control in both variants, so no environment needs
-- a code fork or a configuration switch (constitution VIII, research.md R3).
--
-- WHY THE BACKFILL RUNS HERE. The only history that can be derived without
-- guessing is each account's CURRENT window, and `started_at` is reset on every
-- change. Doing it in the same migration makes the schema and its initial rows
-- atomic and impossible to forget; a re-run is harmless because `BACKFILL:<id>`
-- is a unique source key. The preflight aborts on a corrupt finite window rather
-- than skipping it: a skipped user would silently vanish from the report, which
-- is the very defect this feature removes (FR-013, research.md R6).
-- ============================================================================

CREATE TABLE IF NOT EXISTS subscription.user_plan_periods (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID         NOT NULL
                                      CONSTRAINT fk_user_plan_periods_user
                                      REFERENCES identity.users(id) ON DELETE CASCADE,
    subscription_plan_id UUID         NOT NULL
                                      CONSTRAINT fk_user_plan_periods_plan
                                      REFERENCES subscription.subscription_plans(id) ON DELETE RESTRICT,
    -- Snapshot columns: the report must keep the plan the user actually held,
    -- even after the catalogue renames or reprices that plan.
    plan_code            VARCHAR(100) NOT NULL,
    plan_name            VARCHAR(255) NOT NULL,
    tier_rank            INTEGER      NOT NULL,
    plan_kind            SMALLINT     NOT NULL,
    -- Half-open [valid_from, valid_to): `valid_to = NULL` means open-ended
    -- (free / lifetime). `tstzrange` maps a NULL upper bound to infinity.
    valid_from           TIMESTAMPTZ  NOT NULL,
    valid_to             TIMESTAMPTZ,
    -- Idempotency / traceability. NULL for rows written outside the recorder;
    -- `BACKFILL:<user_id>` for the initial window. UNIQUE is the second guard
    -- behind the publisher's event-key early return (research.md R9).
    source_event_key     VARCHAR(255) CONSTRAINT ux_user_plan_periods_source_event_key UNIQUE,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT chk_user_plan_periods_valid_range
        CHECK (valid_to IS NULL OR valid_to > valid_from)
);

CREATE TABLE IF NOT EXISTS subscription.plan_history_meta (
    id         VARCHAR(20) PRIMARY KEY,
    started_at TIMESTAMPTZ NOT NULL,
    note       TEXT
);

-- The report's overlap predicate is `tstzrange(valid_from, valid_to, '[)') && ...`.
-- A GiST expression index serves it without any extension (unlike the EXCLUDE
-- constraint above), so this index exists in BOTH variants.
CREATE INDEX IF NOT EXISTS idx_user_plan_periods_range
    ON subscription.user_plan_periods
    USING gist (tstzrange(valid_from, valid_to, '[)'));

-- Cheap "started before X" narrowing for one-sided and wide periods.
CREATE INDEX IF NOT EXISTS idx_user_plan_periods_valid_from
    ON subscription.user_plan_periods (valid_from);

-- Per-user history look-up (US2): ordered by valid_from.
CREATE INDEX IF NOT EXISTS idx_user_plan_periods_user_valid_from
    ON subscription.user_plan_periods (user_id, valid_from);

-- ----------------------------------------------------------------------------
-- Non-overlap guard, adaptive.
--
-- The inner block is a subtransaction: if `CREATE EXTENSION` or the ALTER fails
-- with one of the three portability errors below, both statements roll back and
-- the handler creates the partial unique index instead. Any other error is
-- re-raised, because a genuine defect must not be mistaken for a restricted
-- environment.
-- ----------------------------------------------------------------------------
DO $$
BEGIN
    BEGIN
        CREATE EXTENSION IF NOT EXISTS btree_gist;

        ALTER TABLE subscription.user_plan_periods
            ADD CONSTRAINT ex_user_plan_periods_no_overlap
            EXCLUDE USING gist (
                user_id WITH =,
                tstzrange(valid_from, valid_to, '[)') WITH &&
            );

        RAISE NOTICE '031: exclusion constraint ex_user_plan_periods_no_overlap installed (btree_gist available)';
    EXCEPTION
        WHEN insufficient_privilege OR undefined_file OR feature_not_supported THEN
            CREATE UNIQUE INDEX IF NOT EXISTS ux_user_plan_periods_open
                ON subscription.user_plan_periods (user_id)
                WHERE valid_to IS NULL;

            RAISE NOTICE '031: btree_gist unavailable (%), falling back to ux_user_plan_periods_open; closed intervals are protected by the recorder row lock only', SQLERRM;
    END;
END
$$;

-- ----------------------------------------------------------------------------
-- Preflight: refuse to migrate a corrupt finite window.
--
-- `chk_user_subscription_kind_expiry` already requires an expiry for plan_kind
-- 1, but it does not require `expires_at > started_at`. The new CHECK above
-- cannot accept such a row, and silently skipping it would delete that user
-- from every future report. Abort and name the accounts so an operator can fix
-- the data before retrying.
-- ----------------------------------------------------------------------------
DO $$
DECLARE
    invalid_count BIGINT;
    invalid_users TEXT;
BEGIN
    SELECT COUNT(*), string_agg(user_id::text, ', ' ORDER BY user_id)
      INTO invalid_count, invalid_users
      FROM subscription.user_subscriptions
     WHERE current_plan_kind = 1
       AND expires_at IS NOT NULL
       AND expires_at <= started_at;

    IF invalid_count > 0 THEN
        RAISE EXCEPTION
            '031 backfill aborted: % subscription row(s) have current_plan_kind = 1 and expires_at <= started_at. Repair these user_id values before retrying: %',
            invalid_count, invalid_users;
    END IF;
END
$$;

-- ----------------------------------------------------------------------------
-- Backfill + cutover.
--
-- One row per `user_subscriptions` row, using the only window that is exactly
-- known. History before this instant is unknown and deliberately not invented
-- (FR-013). `ON CONFLICT (source_event_key)` makes a re-run harmless.
-- ----------------------------------------------------------------------------
DO $$
DECLARE
    backfill_started TIMESTAMPTZ := clock_timestamp();
    inserted_rows    BIGINT;
    cutover_at       TIMESTAMPTZ;
BEGIN
    INSERT INTO subscription.user_plan_periods (
        user_id,
        subscription_plan_id,
        plan_code,
        plan_name,
        tier_rank,
        plan_kind,
        valid_from,
        valid_to,
        source_event_key
    )
    SELECT
        us.user_id,
        us.subscription_plan_id,
        us.current_plan_code,
        sp.name,
        us.current_tier_rank,
        us.current_plan_kind,
        us.started_at,
        us.expires_at,
        'BACKFILL:' || us.user_id::text
    FROM subscription.user_subscriptions AS us
    JOIN subscription.subscription_plans AS sp
      ON sp.id = us.subscription_plan_id
    ON CONFLICT (source_event_key) DO NOTHING;

    GET DIAGNOSTICS inserted_rows = ROW_COUNT;

    -- First deploy wins: a re-run must not move the cutover forward and thereby
    -- claim history that was never backfilled.
    INSERT INTO subscription.plan_history_meta (id, started_at, note)
    VALUES (
        'cutover',
        now(),
        'Complete plan history starts at this instant. Windows before it are unknown (031 FR-013).'
    )
    ON CONFLICT (id) DO NOTHING;

    SELECT started_at INTO cutover_at
      FROM subscription.plan_history_meta
     WHERE id = 'cutover';

    RAISE NOTICE
        '031: backfilled % user_plan_periods row(s) in % ms; cutover = %',
        inserted_rows,
        ROUND(EXTRACT(EPOCH FROM (clock_timestamp() - backfill_started)) * 1000),
        cutover_at;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Deliberately does NOT drop `btree_gist`: the extension is shared cluster
-- infrastructure and may predate this migration. Dropping the tables also drops
-- their indexes and the exclusion constraint.
DROP TABLE IF EXISTS subscription.user_plan_periods;
DROP TABLE IF EXISTS subscription.plan_history_meta;
-- +goose StatementEnd
