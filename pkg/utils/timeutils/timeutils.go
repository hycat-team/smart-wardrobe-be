package timeutils

import (
	"time"
)

// GetNow returns the current time in the specified timezone
func GetNow(timezone string) time.Time {
	if timezone == "" {
		timezone = "Asia/Ho_Chi_Minh"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Now().In(time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60))
	}
	return time.Now().In(loc)
}
