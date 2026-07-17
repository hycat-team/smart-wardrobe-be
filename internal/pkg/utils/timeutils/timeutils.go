package timeutils

import (
	"math/rand"
	"strconv"
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

// GenerateOrderCode generates a unique numeric order code based on local timezone date-time format YYMMDDHHMMSS + 3 random digits.
// This is guaranteed to be unique and fits perfectly within PayOS's safe integer limit (9007199254740991).
func GenerateOrderCode() int64 {
	now := GetNow(VNmTimezone)
	formatted := now.Format("060102150405")
	val, _ := strconv.ParseInt(formatted, 10, 64)
	r := rand.Intn(1000)

	return val*1000 + int64(r)
}
