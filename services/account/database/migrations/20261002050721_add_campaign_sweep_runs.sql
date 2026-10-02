-- +goose Up
-- +goose StatementBegin
-- ============================================================================
-- Feature 025 T079a (FR-059) — catch-up sweep health ledger.
--
-- WHY THIS EXISTS, AND WHY IT IS A TABLE. FR-059 asks an operator to be able to
-- tell three states apart: a campaign is running, a campaign has gone to sleep,
-- and a campaign is RUNNING BUT BROKEN — configured, inside its window, and not
-- granting anybody anything. Only the third is a fault, and no per-account log
-- line can show it, because every individual skip looks locally correct.
--
-- The earlier design kept that flag in the worker's memory and justified losing
-- it on restart as correct behaviour. That justification only holds while there
-- is exactly ONE worker, and there is not: this service runs as several
-- replicas, each with its own worker set (spec.md "Bài toán 1"). Two failures
-- follow, and they are the dangerous direction:
--
--   1. An HTTP request landing on replica B reads replica B's flag, while the
--      sweep that actually failed was run by replica A. The page then reports
--      HEALTHY while the mechanism is broken — the one answer FR-059 exists to
--      prevent.
--   2. Under APP_MODE=http no sweep worker runs at all, so `degraded` would be
--      permanently false: a false all-clear that is structurally guaranteed.
--
-- So the state lives in the database, where every replica reads the same row and
-- where the read-only operator API (FR-065) can reach it without a worker.
--
-- WHY ONE ROW PER CAMPAIGN, KEYED ON campaign_code. A sweep run is not per
-- campaign — one run covers every configured campaign — but the health an
-- operator asks about IS per campaign. Keying on the code is what makes
-- "which campaign is broken" answerable, and it matches `campaign_claims`, which
-- is keyed the same way, so a campaign's budget and its health sit side by side.
--
-- WHY THE COLUMNS ARE EXACTLY THESE FOUR. `last_sweep_at` answers "when did a
-- sweep last complete"; `degraded` is the boolean an operator scans a list for;
-- `degraded_reason` is the diagnosis, so nobody has to correlate a timestamp
-- with a log search. A sweep that completes CLEARS the reason, because the flag
-- describes the most recent run and "it stopped breaking" is as much of an event
-- as "it broke".
--
-- NOT A COUNTER. Nothing here accumulates. Every column is either a timestamp or
-- a state, so there is no stored number that could drift from `campaign_claims`
-- and need reconciling (data-model.md §5.1, §4.3).
-- ============================================================================

CREATE TABLE IF NOT EXISTS subscription.campaign_sweep_runs (
    campaign_code   VARCHAR(64) PRIMARY KEY,
    last_sweep_at   TIMESTAMPTZ,
    degraded        BOOLEAN     NOT NULL DEFAULT FALSE,
    degraded_reason TEXT,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE subscription.campaign_sweep_runs IS
    'One row per campaign: the health of the FR-079 catch-up sweep, read by the operator look-up (FR-065) from every replica.';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Reverse of Up. The table carries no data that cannot be rebuilt — the next
-- sweep rewrites every row — so dropping it loses only the history of how often
-- the sweep ran, which nothing reads.

DROP TABLE IF EXISTS subscription.campaign_sweep_runs;
-- +goose StatementEnd