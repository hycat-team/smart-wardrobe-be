package streamutils

import "fmt"

// SanitizeSSEData escapes a string safely for SSE data format
func SanitizeSSEData(content string) string {
	return fmt.Sprintf("%q", content)
}
