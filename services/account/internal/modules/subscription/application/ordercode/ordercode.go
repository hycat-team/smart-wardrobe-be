package ordercode

import (
	"crypto/rand"
	"math/big"
	"time"
)

const (
	// MaxSafeOrderCode mirrors JavaScript Number.MAX_SAFE_INTEGER so order codes
	// stay exactly representable when parsed by web clients.
	MaxSafeOrderCode int64 = 9007199254740991

	randSuffixModulo int64 = 1000
)

// Generate returns a time-ordered PayOS order code: unixMillis * 1000 + rand(0..999).
// The millisecond prefix keeps codes sortable and far above the legacy BIGSERIAL
// range (1, 2, 3, ...) that PayOS rejects with business error 231 after any DB
// reset, while the random suffix absorbs bursts within the same millisecond.
// The result always stays below MaxSafeOrderCode.
func Generate() int64 {
	return GenerateAt(time.Now().UTC(), randomSuffix())
}

// GenerateAt is the pure core of Generate, exposed for deterministic tests.
func GenerateAt(now time.Time, suffix int64) int64 {
	if suffix < 0 {
		suffix = -suffix
	}
	return now.UTC().UnixMilli()*randSuffixModulo + suffix%randSuffixModulo
}

func randomSuffix() int64 {
	n, err := rand.Int(rand.Reader, big.NewInt(randSuffixModulo))
	if err != nil {
		return time.Now().UTC().UnixNano() % randSuffixModulo
	}
	return n.Int64()
}
