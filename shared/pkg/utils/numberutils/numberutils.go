package numberutils

import (
	"fmt"
	"math"
)

// FormatBytes renders a byte count for user-facing messages: KiB below 1 MiB,
// MiB at or above it, keeping a single decimal for fractional MiB values.
func FormatBytes(b int64) string {
	const (
		kib = 1024
		mib = 1024 * 1024
	)
	switch {
	case b >= mib:
		mb := float64(b) / mib
		if mb == math.Trunc(mb) {
			return fmt.Sprintf("%d MB", int64(mb))
		}
		return fmt.Sprintf("%.1f MB", mb)
	case b >= kib:
		return fmt.Sprintf("%.0f KB", float64(b)/kib)
	default:
		return fmt.Sprintf("%d byte", b)
	}
}
