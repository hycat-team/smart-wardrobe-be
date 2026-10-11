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
