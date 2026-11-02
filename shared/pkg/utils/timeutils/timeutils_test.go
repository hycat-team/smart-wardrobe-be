package timeutils_test

import (
	"testing"
	"time"

	"smart-wardrobe-be/shared/pkg/utils/timeutils"
)

func TestVNDayRangeUsesVietnamCalendar(t *testing.T) {
	// 08:00 +07 on 2026-10-11 is 01:00 UTC the same day.
	in := time.Date(2026, 10, 11, 8, 0, 0, 0, timeutils.VnLoc())
	start, end := timeutils.VNDayRange(in)

	wantStart := time.Date(2026, 10, 11, 0, 0, 0, 0, timeutils.VnLoc())
	if !start.Equal(wantStart) {
		t.Fatalf("day start must be VN midnight, got %v", start)
	}
	if end.Sub(start) != 24*time.Hour {
		t.Fatalf("day range must span exactly 24h, got %v", end.Sub(start))
	}
}

func TestVNDayRangeLateUTCBelongsToNextVNDay(t *testing.T) {
	// 23:30 UTC on 2026-10-10 is 06:30 +07 on 2026-10-11. A UTC-midnight
	// Truncate(24h) — the old implementation — would have put this on the
	// 10th and silently mixed seven hours of the previous VN day into "today".
	in := time.Date(2026, 10, 10, 23, 30, 0, 0, time.UTC)
	start, _ := timeutils.VNDayRange(in)

	wantStart := time.Date(2026, 10, 11, 0, 0, 0, 0, timeutils.VnLoc())
	if !start.Equal(wantStart) {
		t.Fatalf("expected VN day 2026-10-11, got %v", start)
	}
}

func TestTodayStartVNIsTodayMidnight(t *testing.T) {
	now := timeutils.NowVN()
	start := timeutils.TodayStartVN()
	if start.After(now) || now.Sub(start) >= 24*time.Hour {
		t.Fatalf("today start %v does not contain now %v", start, now)
	}
}

// TestFormatDateVNKeepsTheCustomersCalendar is the defect behind the expiry
// dates in emails: the stored instant is the same one the database session
// hands back, and formatting it directly inherits that location. A plan ending
// at 2026-11-01 17:30Z ends on 02/11 in Vietnam.
func TestFormatDateVNKeepsTheCustomersCalendar(t *testing.T) {
	cases := []struct {
		name string
		in   time.Time
		want string
	}{
		{"vn midnight", time.Date(2026, 11, 2, 0, 0, 0, 0, timeutils.VnLoc()), "02/11/2026"},
		{"late UTC belongs to next VN day", time.Date(2026, 11, 1, 17, 30, 0, 0, time.UTC), "02/11/2026"},
		{"early UTC is same VN day", time.Date(2026, 11, 2, 3, 0, 0, 0, time.UTC), "02/11/2026"},
		{"utc evening is next VN day", time.Date(2026, 11, 1, 23, 0, 0, 0, time.UTC), "02/11/2026"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := timeutils.FormatDateVN(tc.in); got != tc.want {
				t.Fatalf("FormatDateVN(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestCalendarDateAtNoonUTC pins the fix for the DATE-column write on
// Supabase: a UTC session casts '2026-10-20 00:00+07' (= 2026-10-19 17:00Z) to
// DATE 2026-10-19, one day earlier. The noon-UTC value must stay on 10-20 in
// both a UTC session and a +07 session.
func TestCalendarDateAtNoonUTC(t *testing.T) {
	want := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		in   time.Time
	}{
		{"vn midnight", time.Date(2026, 10, 20, 0, 0, 0, 0, timeutils.VnLoc())},
		{"late UTC previous day", time.Date(2026, 10, 19, 18, 30, 0, 0, time.UTC)},
		{"vn midday", time.Date(2026, 10, 20, 12, 0, 0, 0, timeutils.VnLoc())},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := timeutils.CalendarDateAtNoonUTC(tc.in)
			if !got.Equal(want) {
				t.Fatalf("CalendarDateAtNoonUTC(%v) = %v, want %v", tc.in, got, want)
			}
			// A DATE cast in a UTC session and in a +07 session must both be 10-20.
			if got.Format("2006-01-02") != "2026-10-20" {
				t.Fatalf("date label = %s, want 2026-10-20", got.Format("2006-01-02"))
			}
		})
	}
}
