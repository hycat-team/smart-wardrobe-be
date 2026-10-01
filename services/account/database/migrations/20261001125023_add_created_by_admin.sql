-- +goose Up
-- +goose StatementBegin
-- ============================================================================
-- Feature 025 T015 (FR-050 / FR-091) — origin of an account row.
--
-- WHY THIS EXISTS. `identity.users` records nothing about WHO created the row, so
-- an administrator-created account is structurally indistinguishable from a
-- self-serve one. The campaign exclusion rule (FR-091) can then only be evaluated
-- on the event path, where the publishing code happens to know which flow it is in
-- — and the catch-up sweep (FR-079), which reads the database, could not evaluate
-- it at all. It would have granted the promotional plan to the administrator's own
-- test accounts and eaten real customers' quota to do it.
--
-- This is a permanent property of the account, not a campaign field: it is equally
-- correct for reporting, support and abuse analysis. Precedent: feature 024
-- carried `RoleSlug` on the identity contract for exactly the same reason
-- (services/account/dto/identity/contract_dto.go:20-24).
--
-- NOT DERIVED FROM `email_verified`. That is a different fact: an administrator may
-- create an account that is already verified, and every self-serve account is
-- verified. Deriving one from the other fails silently, which is the failure mode
-- FR-090 exists to forbid.
--
-- DEFAULT FALSE, NOT NULL. The two self-serve doors (email sign-up, Google first
-- login) never set the flag, so the database default is what records "not an
-- administrator account" for them — and a column that can be NULL would make
-- "unknown" a third answer the campaign rule would have to interpret.
--
-- Written at exactly ONE site in the whole source tree
-- (identity/application/usecase/admin_uc.go, FR-091). A boundary test proves it,
-- because the compiler cannot.
-- ============================================================================

ALTER TABLE identity.users
    ADD COLUMN created_by_admin BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN identity.users.created_by_admin IS
    'True only when the account was created through the administrator endpoint. Set at exactly one write site; never written from configuration.';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drop the column rather than resetting it to FALSE: there is no third state to
-- preserve, and a rollback that left the column behind would make "migrated down"
-- and "migrated up" leave different schemas behind.

ALTER TABLE identity.users DROP COLUMN IF EXISTS created_by_admin;
-- +goose StatementEnd
