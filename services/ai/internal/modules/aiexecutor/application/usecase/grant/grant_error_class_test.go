package grant

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// ---------------------------------------------------------------------------
// 027 US3 (T017): the classification itself, without a database.
//
// These cases use hand-built *pgconn.PgError values, which is correct HERE and
// forbidden for T016. The distinction is the point: this test pins the LOGIC of
// isCheckViolation (which code is accepted, and that Unwrap is walked), while
// T016 pins the INTEGRATION FACT that GORM keeps a *pgconn.PgError reachable
// through errors.As at all (research.md R-001). A hand-built error would be
// green even if the integration were broken, so it can never replace T016.
// ---------------------------------------------------------------------------

func TestIsCheckViolation(t *testing.T) {
	t.Run("nil is not a violation", func(t *testing.T) {
		if isCheckViolation(nil) {
			t.Error("isCheckViolation(nil) phải trả false, không được ném lỗi")
		}
	})

	t.Run("a check violation is found through several wrapping layers", func(t *testing.T) {
		base := &pgconn.PgError{Code: pgCheckViolationCode, Message: "new row violates check constraint"}
		err := fmt.Errorf("open transaction: %w", fmt.Errorf("create grant: %w", base))
		if !isCheckViolation(err) {
			t.Fatalf("errors.As phải bóc được *pgconn.PgError qua nhiều lớp bọc: %v", err)
		}
	})

	t.Run("a unique violation is not a check violation (FR-010)", func(t *testing.T) {
		// 23505 is the code that must NOT be reclassified. It is a recoverable
		// race, and a widened class-23 match would wrongly dead-letter it.
		err := fmt.Errorf("create grant: %w", &pgconn.PgError{Code: "23505"})
		if isCheckViolation(err) {
			t.Error("23505 (unique_violation) phải trả false; xếp nó vĩnh viễn sẽ phá FR-010")
		}
	})

	t.Run("a serialization failure is not a check violation (FR-010)", func(t *testing.T) {
		err := &pgconn.PgError{Code: "40001"}
		if isCheckViolation(err) {
			t.Error("40001 (serialization_failure) phải trả false; nó là sự cố tạm thời")
		}
	})

	t.Run("a non-Postgres error is not a check violation", func(t *testing.T) {
		if isCheckViolation(errors.New("connection reset by peer")) {
			t.Error("lỗi không phải Postgres phải trả false")
		}
	})
}
