package numberutils

import "strconv"

// ToFloat64 converts any numeric type from pgx map scan to float64.
// pgx driver maps PostgreSQL int8/float8/numeric to int64/float64/[]uint8 depending on context.
func ToFloat64(v any) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int64:
		return float64(val)
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case uint64:
		return float64(val)
	case uint:
		return float64(val)
	case []uint8:
		f, _ := strconv.ParseFloat(string(val), 64)
		return f
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}
