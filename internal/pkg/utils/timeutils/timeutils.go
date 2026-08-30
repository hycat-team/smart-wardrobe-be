package timeutils

import (
	"time"
)

const VNmTimezone = "Asia/Ho_Chi_Minh"

// GetNow returns the current time in the specified timezone
func GetNow(timezone string) time.Time {
	if timezone == "" {
		timezone = VNmTimezone
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Now().In(time.FixedZone(VNmTimezone, 7*60*60))
	}
	return time.Now().In(loc)
}
