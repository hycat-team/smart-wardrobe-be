-- +goose Up
-- +goose StatementBegin
-- ============================================================================
-- Feature 024 — single-owner AI policy grants (GĐ0 schema)
--
-- Four schema changes in ONE file on purpose: they ship as a single release
-- and all four live in the Account migration tree (the AI service has none of
-- its own — see research.md R6). Splitting them would only add files.
--
-- (1) Index ai_policy_grants(source_event_id).
--     Every inbound plan-changed event runs a duplicate check on that column
--     (FR-033), and the existing index is (user_id, effective_from,
--     effective_to), which cannot serve it. Verified on PostgreSQL 17.10.
--
-- (2) ai_usage_period_ledgers.grant_id becomes nullable + a dedicated partial
--     unique index for rows that have no grant.
--     Fallback tier 1 (FR-007) must still go through the reservation-first
--     FOR UPDATE ledger of constitution §VI — only accounts WITHOUT a grant
--     need a ledger row, and that row has no grant_id to point at. The existing
--     UNIQUE (grant_id, period_index) cannot dedupe those because SQL treats
--     NULL as distinct from NULL, so every call would open a fresh ledger and
--     the anti-overspend control would evaporate. The partial index below
--     restores dedup for exactly those rows and stays disjoint from the real
--     ledgers, which remain keyed by grant_id.
--     Verified on PostgreSQL 17.10: first insert 1 row, second insert 0 rows.
--
-- (3) ai_cost_policy_operations.max_paid_attempts_per_day becomes nullable and
--     is set to NULL for premium-default.
--     In strict mode that column is never read (FR-005e) — strict policies are
--     bounded by the cycle hard cap instead. NULL is the honest value; leaving
--     30/10 there would be dead configuration that reads like a live limit.
--     A NULL in observe_only remains a misconfiguration and is refused at
--     runtime, because the free policy has no hard cap and the per-day counter
--     is then its only bound.
--
-- (4) New table aiexecutor.ai_cost_policy_audit_logs.
--     Audit trail for administrator edits of the cost policy, owned by the AI
--     service. shared.admin_audit_logs belongs to Account's admin module and
--     must not be written from AI (constitution §I). The same transaction also
--     publishes an admin.audit.log outbox record so the change still reaches
--     the central audit screen.
--
-- The two *_actor_user_id / changed_at indexes support "what did this admin
-- change" and "what changed in this window" lookups.
-- ============================================================================

CREATE INDEX IF NOT EXISTS ix_ai_policy_grants_source_event_id
    ON aiexecutor.ai_policy_grants (source_event_id);

ALTER TABLE aiexecutor.ai_usage_period_ledgers
    ALTER COLUMN grant_id DROP NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_ledgers_user_period_fallback
    ON aiexecutor.ai_usage_period_ledgers (user_id, period_index)
    WHERE grant_id IS NULL;

ALTER TABLE aiexecutor.ai_cost_policy_operations
    ALTER COLUMN max_paid_attempts_per_day DROP NOT NULL;

UPDATE aiexecutor.ai_cost_policy_operations
   SET max_paid_attempts_per_day = NULL
 WHERE policy_id = (SELECT id FROM aiexecutor.ai_cost_policies WHERE code = 'premium-default');

CREATE TABLE IF NOT EXISTS aiexecutor.ai_cost_policy_audit_logs (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id uuid        NOT NULL,
    changed_at    timestamptz NOT NULL DEFAULT now(),
    request_ip    varchar(64),
    policy_code   varchar(100) NOT NULL,
    operation     varchar(30)  NOT NULL,
    field_name    varchar(60)  NOT NULL,
    -- Nullable on purpose: "was absent" and "was set to the empty string" are
    -- different facts, and an audit trail that cannot tell a deletion from an
    -- empty write cannot answer a later question.
    old_value     varchar(255),
    new_value     varchar(255),
    action        varchar(60)  NOT NULL,
    created_at    timestamptz  NOT NULL DEFAULT now(),
    updated_at    timestamptz  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_ai_cost_policy_audit_logs_actor_user_id
    ON aiexecutor.ai_cost_policy_audit_logs (actor_user_id);

CREATE INDEX IF NOT EXISTS ix_ai_cost_policy_audit_logs_changed_at
    ON aiexecutor.ai_cost_policy_audit_logs (changed_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- The Down half must undo the four changes in the REVERSE order, and two of
-- them need data restored before the NOT NULL constraint can come back —
-- otherwise the ALTER fails and the whole rollback is stuck.

-- (4) audit table first, while nothing else depends on it.
DROP TABLE IF EXISTS aiexecutor.ai_cost_policy_audit_logs;

-- (3) Put the premium values back BEFORE re-adding NOT NULL. These are the
-- values observed on 2026-09-29; restoring an exact prior state is only
-- possible because the column was not nullable before this migration.
-- The lookup has to be correlated on the operation as well as the policy:
-- a derived table keyed on policy_id alone returns four candidate rows for one
-- policy, and PostgreSQL then picks an arbitrary one — every operation would
-- come back as the chat value.
UPDATE aiexecutor.ai_cost_policy_operations o
   SET max_paid_attempts_per_day = CASE o.operation
        WHEN 'chat'     THEN 10
        WHEN 'outfit'   THEN 30
        WHEN 'rewriter' THEN 5
        WHEN 'summary'  THEN 5
        ELSE o.max_paid_attempts_per_day
   END
 WHERE EXISTS (
        SELECT 1
          FROM aiexecutor.ai_cost_policies p
         WHERE p.id = o.policy_id
           AND p.code = 'premium-default'
           AND o.max_paid_attempts_per_day IS NULL
 );

ALTER TABLE aiexecutor.ai_cost_policy_operations
    ALTER COLUMN max_paid_attempts_per_day SET NOT NULL;

-- (2) Delete the fallback ledger rows BEFORE re-adding NOT NULL: they have no
-- grant_id to restore, so they cannot satisfy the constraint. Deleting them is
-- safe because a NULL-grant ledger only exists to bound fallback traffic, and
-- the next request recreates one.
DELETE FROM aiexecutor.ai_usage_period_ledgers WHERE grant_id IS NULL;

ALTER TABLE aiexecutor.ai_usage_period_ledgers
    ALTER COLUMN grant_id SET NOT NULL;

DROP INDEX IF EXISTS aiexecutor.ux_ledgers_user_period_fallback;

-- (1) Index last.
DROP INDEX IF EXISTS aiexecutor.ix_ai_policy_grants_source_event_id;
-- +goose StatementEnd
