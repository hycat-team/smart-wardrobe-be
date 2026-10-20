package timeutils

import "time"

const VNmTimezone = "Asia/Ho_Chi_Minh"

// vnLoc is the fixed +07:00 zone used by every daily aggregation. A fixed zone
// is deliberate: time.LoadLocation depends on the tzdata being present in the
// image, and a stats pipeline must not silently fall back to UTC because the
// container lacks it.
var vnLoc = time.FixedZone(VNmTimezone, 7*60*60)

// VnLoc returns the fixed +07:00 zone.
func VnLoc() *time.Location { return vnLoc }

// GetNow returns the current time in the specified timezone
func GetNow(timezone string) time.Time {
	if timezone == "" {
		timezone = VNmTimezone
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Now().In(vnLoc)
	}
	return time.Now().In(loc)
}

// NowVN returns the current time in Vietnam (+07:00).
func NowVN() time.Time {
	return time.Now().In(vnLoc)
}

// StartOfDayInVN returns 00:00 (+07:00) of the Vietnam calendar day that
// contains t.
func StartOfDayInVN(t time.Time) time.Time {
	local := t.In(vnLoc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, vnLoc)
}

// EndOfDayInVN returns 00:00 (+07:00) of the day after the Vietnam calendar day
// that contains t — the exclusive upper bound of that day.
func EndOfDayInVN(t time.Time) time.Time {
	return StartOfDayInVN(t).AddDate(0, 0, 1)
}

// VNDayRange returns the [start, end) bounds of the Vietnam calendar day that
// contains t.
func VNDayRange(t time.Time) (time.Time, time.Time) {
	start := StartOfDayInVN(t)
	return start, start.AddDate(0, 0, 1)
}

// TodayStartVN returns 00:00 (+07:00) of the current Vietnam day.
func TodayStartVN() time.Time {
	return StartOfDayInVN(NowVN())
}

// CalendarDateAtNoonUTC maps t to 12:00 UTC of the Vietnam calendar day that
// contains t.
//
// It exists for DATE columns written from a time.Time. Postgres casts a
// timestamptz to DATE with the SESSION timezone, and a Supabase/Supavisor
// session is UTC even when the DSN asks for Asia/Ho_Chi_Minh: a VN-midnight
// instant ('2026-10-20 00:00+07' = '2026-10-19 17:00Z') is then stored as the
// PREVIOUS day (reproduced on AWS: the cast yields 2026-10-19). Noon UTC maps
// to the same calendar date in every timezone within +/-12h, so the DATE value
// no longer depends on the session timezone.
func CalendarDateAtNoonUTC(t time.Time) time.Time {
	local := t.In(vnLoc)
	return time.Date(local.Year(), local.Month(), local.Day(), 12, 0, 0, 0, time.UTC)
}
