package validation

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"
)

func vnDay(year int, month time.Month, day, hour int) time.Time {
	return time.Date(year, month, day, hour, 0, 0, 0, timeutils.VnLoc())
}

func assertZeroTime(t *testing.T, name string, got time.Time) {
	t.Helper()
	if !got.IsZero() {
		t.Fatalf("%s: expected zero time, got %s", name, got)
	}
}

func assertTimeEqual(t *testing.T, name string, got, want time.Time) {
	t.Helper()
	if !got.Equal(want) {
		t.Fatalf("%s: expected %s, got %s", name, want, got)
	}
}

func assertBadRequest(t *testing.T, err error) *apperror.Error {
	t.Helper()
	if err == nil {
		t.Fatal("expected a validation error, got nil")
	}
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.Error, got %T: %v", err, err)
	}
	if appErr.Status != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", appErr.Status)
	}
	return appErr
}

func TestParseVNDateRangeQueryBothEmpty(t *testing.T) {
	from, to, err := ParseVNDateRangeQuery("", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertZeroTime(t, "from", from)
	assertZeroTime(t, "to", to)
}

func TestParseVNDateRangeQueryOnlyFromDate(t *testing.T) {
	from, to, err := ParseVNDateRangeQuery("2026-03-01", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertTimeEqual(t, "from", from, vnDay(2026, time.March, 1, 0))
	assertZeroTime(t, "to", to)
}

func TestParseVNDateRangeQueryOnlyToDate(t *testing.T) {
	from, to, err := ParseVNDateRangeQuery("", "2026-03-01")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertZeroTime(t, "from", from)
	assertTimeEqual(t, "to", to, vnDay(2026, time.March, 2, 0))
}

func TestParseVNDateRangeQuerySameDayCoversWholeVNDay(t *testing.T) {
	from, to, err := ParseVNDateRangeQuery("2026-03-01", "2026-03-01")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertTimeEqual(t, "from", from, vnDay(2026, time.March, 1, 0))
	assertTimeEqual(t, "to", to, vnDay(2026, time.March, 2, 0))
}

func TestParseVNDateRangeQueryMonthEndRollsToNextMonth(t *testing.T) {
	from, to, err := ParseVNDateRangeQuery("2026-02-28", "2026-02-28")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertTimeEqual(t, "from", from, vnDay(2026, time.February, 28, 0))
	assertTimeEqual(t, "to", to, vnDay(2026, time.March, 1, 0))
}

func TestParseVNDateRangeQueryVNBoundaries(t *testing.T) {
	from, to, err := ParseVNDateRangeQuery("2026-03-01", "2026-03-31")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// from is 00:00 (+07:00) of the first day; to is 00:00 (+07:00) of the day
	// after the last one.
	assertTimeEqual(t, "from", from, vnDay(2026, time.March, 1, 0))
	assertTimeEqual(t, "to", to, vnDay(2026, time.April, 1, 0))

	// The boundaries must carry the VN offset, not UTC.
	if _, offset := from.Zone(); offset != 7*60*60 {
		t.Fatalf("expected from offset +07:00, got %d seconds", offset)
	}
	if _, offset := to.Zone(); offset != 7*60*60 {
		t.Fatalf("expected to offset +07:00, got %d seconds", offset)
	}
}

func TestParseVNDateRangeQueryInvalidFormat(t *testing.T) {
	tests := []struct {
		name     string
		fromDate string
		toDate   string
	}{
		{name: "invalid fromDate", fromDate: "03-01-2026"},
		{name: "invalid toDate", toDate: "2026/03/01"},
		{name: "nonsense fromDate", fromDate: "not-a-date"},
		{name: "impossible day", fromDate: "2026-02-30"},
		{name: "both invalid", fromDate: "01-03-2026", toDate: "2026-03"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to, err := ParseVNDateRangeQuery(tt.fromDate, tt.toDate)
			assertBadRequest(t, err)
			assertZeroTime(t, "from", from)
			assertZeroTime(t, "to", to)
		})
	}
}

func TestParseVNDateRangeQueryFromAfterTo(t *testing.T) {
	from, to, err := ParseVNDateRangeQuery("2026-03-31", "2026-03-01")
	assertBadRequest(t, err)
	assertZeroTime(t, "from", from)
	assertZeroTime(t, "to", to)
}

func TestParseVNDateRangeQueryAdjacentDayInversionIsRejected(t *testing.T) {
	// With toDate = D the upper bound is 00:00 of D+1, which equals
	// StartOfDayInVN(fromDate) when fromDate = D+1 — the comparison must use the
	// calendar days, not the derived bounds.
	from, to, err := ParseVNDateRangeQuery("2026-03-02", "2026-03-01")
	assertBadRequest(t, err)
	assertZeroTime(t, "from", from)
	assertZeroTime(t, "to", to)
}
