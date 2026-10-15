package validation

import (
	"time"

	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"
)

const vnDateLayout = "2006-01-02"

// ParseVNDateRangeQuery parses fromDate/toDate (YYYY-MM-DD Vietnam calendar
// days, both ends inclusive) into the half-open range [from, to) that every
// period query filters on:
//
//	from = StartOfDayInVN(fromDate)
//	to   = EndOfDayInVN(toDate)   // 00:00 (+07:00) of the next VN day
//
// A missing side (empty string) yields the zero time.Time for that side so the
// repository can skip the corresponding condition (open-ended range).
//
// It returns a 400 apperror only when a present side is malformed or when both
// sides are present and fromDate is after toDate.
func ParseVNDateRangeQuery(fromDate, toDate string) (from, to time.Time, err error) {
	var parsedFrom, parsedTo time.Time
	var hasFrom, hasTo bool

	if fromDate != "" {
		parsedFrom, err = time.Parse(vnDateLayout, fromDate)
		if err != nil {
			return time.Time{}, time.Time{}, apperror.NewBadRequest("Định dạng ngày bắt đầu (fromDate) không hợp lệ. Vui lòng sử dụng YYYY-MM-DD.")
		}
		hasFrom = true
		from = timeutils.StartOfDayInVN(parsedFrom)
	}

	if toDate != "" {
		parsedTo, err = time.Parse(vnDateLayout, toDate)
		if err != nil {
			return time.Time{}, time.Time{}, apperror.NewBadRequest("Định dạng ngày kết thúc (toDate) không hợp lệ. Vui lòng sử dụng YYYY-MM-DD.")
		}
		hasTo = true
		to = timeutils.EndOfDayInVN(parsedTo)
	}

	// Compare the calendar days themselves: with date D and D+1 the derived
	// bounds are equal (both 00:00 of D+1), so from.After(to) would miss that
	// inversion.
	if hasFrom && hasTo && parsedFrom.After(parsedTo) {
		return time.Time{}, time.Time{}, apperror.NewBadRequest("Ngày bắt đầu (fromDate) không được sau ngày kết thúc (toDate).")
	}

	return from, to, nil
}
