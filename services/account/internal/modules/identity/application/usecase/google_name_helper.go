package usecase

import "strings"

// nilIfBlank returns a pointer to the trimmed value, or nil when it is blank.
// Used to avoid persisting empty strings for optional profile fields.
func nilIfBlank(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
