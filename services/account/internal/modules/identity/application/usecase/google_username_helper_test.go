package usecase

import (
	"regexp"
	"testing"
)

var validUsernamePattern = regexp.MustCompile(`^[a-z0-9._]{4,20}$`)

func TestDeriveUsernameBase(t *testing.T) {
	cases := []struct {
		email string
		want  string
	}{
		{"duck@gmail.com", "duck"},
		{"Duck.Nguyen@Gmail.com", "duck.nguyen"},
		{"duck.nguyen+tag@x.com", "duck.nguyentag"},
		{"ab@x.com", "abuser"},
		{"...@x.com", "user"},
		{"a.very.long.local.part.here@x.com", "a.very.long.local.pa"},
	}
	for _, tc := range cases {
		got := deriveUsernameBase(tc.email)
		if got != tc.want {
			t.Errorf("deriveUsernameBase(%q) = %q, want %q", tc.email, got, tc.want)
		}
		if !validUsernamePattern.MatchString(got) {
			t.Errorf("deriveUsernameBase(%q) = %q does not match username rules", tc.email, got)
		}
	}
}

func TestWithUsernameSuffix(t *testing.T) {
	got := withUsernameSuffix("a.very.long.local.part.here", "_a1b2")
	if len(got) > 20 {
		t.Fatalf("suffixed username too long: %q", got)
	}
	if !validUsernamePattern.MatchString(got) {
		t.Fatalf("suffixed username invalid: %q", got)
	}
}
