package grant

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// pgCheckViolationCode is PostgreSQL's SQLSTATE for `check_violation`, and it is
// the ONLY integrity-constraint code this classification accepts.
//
// DO NOT widen this to the whole SQLSTATE class 23 - for example by rewriting the
// comparison below as `strings.HasPrefix(code, "23")`. That "simplification" is
// exactly the change this comment exists to stop, and it fails silently:
//
//   - 23505 (`unique_violation`) is a recoverable race, not a verdict. The
//     duplicate check (`GetGrantBySourceEventID`) runs under the per-account
//     advisory lock, so a 23505 on insert means a concurrent delivery of the same
//     `source_event_id` won the race. A retry once that row is visible succeeds.
//   - 23503 (`foreign_key_violation`) and 23502 (`not_null_violation`) are
//     likewise conditions that a corrected producer or a later retry can clear,
//     and 027 deliberately does not reclassify them.
//
// Turning the whole class permanent would convert a single race into a dead
// letter and make FR-010 wrong, with no signal that anything broke. Only a CHECK
// violation is a deterministic consequence of the values the transaction
// computed: the transaction has already rolled back, the data did not change, and
// redelivering the message reproduces the identical failure. That is why exactly
// 23514 is refused permanently while every other SQLSTATE keeps the pre-027
// "transient" behaviour (contracts/error-classification.md §2.3).
const pgCheckViolationCode = "23514"

// isCheckViolation reports whether err carries a Postgres CHECK-constraint
// violation (SQLSTATE 23514).
//
// This function is the application layer's ENTIRE knowledge of Postgres: it
// answers with a bool, and no caller needs to know what a SQLSTATE is. The
// `pgx` import stops here on purpose (constitution §IV).
//
// `errors.As` rather than a type assertion is deliberate, and it is why US3
// needed an integration test on a real database: the error reaches this package
// wrapped by `gorm.io/driver/postgres`, and if GORM ever wrapped it without `%w`,
// this would return false, the violation would silently fall back to
// "transient", and 027 would be back to the exact bug it removes - with no
// signal that the fix stopped working (research.md R-001, plan.md Risk 1).
//
// A nil error is false rather than a panic, because callers reach this from an
// error that may be nil.
func isCheckViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.SQLState() == pgCheckViolationCode
}
