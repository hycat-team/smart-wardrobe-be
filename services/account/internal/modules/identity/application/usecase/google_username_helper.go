package usecase

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
)

var (
	usernameDisallowed = regexp.MustCompile(`[^a-z0-9._]`)
	usernameRepeats    = regexp.MustCompile(`[._]{2,}`)
)

const (
	usernameMinLen = 4
	usernameMaxLen = 20
)

// deriveUsernameBase converts a Google email into a candidate username that
// satisfies the project `username` validator: ^[a-zA-Z0-9._]{4,20}$, no
// consecutive [._], and no leading/trailing [._].
func deriveUsernameBase(email string) string {
	local := email
	if at := strings.IndexByte(email, '@'); at >= 0 {
		local = email[:at]
	}

	local = strings.ToLower(local)
	local = usernameDisallowed.ReplaceAllString(local, "")
	local = usernameRepeats.ReplaceAllString(local, "_")
	local = strings.Trim(local, "._")

	if len(local) > usernameMaxLen {
		local = strings.TrimRight(local[:usernameMaxLen], "._")
	}
	if len(local) < usernameMinLen {
		local += "user"
	}
	if len(local) > usernameMaxLen {
		local = strings.TrimRight(local[:usernameMaxLen], "._")
	}
	return local
}

// withUsernameSuffix appends a suffix while keeping the total within the limit.
func withUsernameSuffix(base string, suffix string) string {
	if base == "" {
		base = "user"
	}
	max := usernameMaxLen - len(suffix)
	if len(base) > max {
		base = base[:max]
	}
	base = strings.TrimRight(base, "._")
	if len(base) < usernameMinLen-len(suffix) {
		base += "user"
	}
	return base + suffix
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "0000"
	}
	return hex.EncodeToString(buf)
}
