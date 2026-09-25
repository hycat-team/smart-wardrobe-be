package shared

import (
	"crypto/rand"
	"math/big"
)

const publicIDCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GeneratePublicID returns a random 32-character identifier used in post URLs.
func GeneratePublicID() (string, error) {
	const length = 32
	buf := make([]byte, length)
	max := big.NewInt(int64(len(publicIDCharset)))
	for i := range buf {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		buf[i] = publicIDCharset[n.Int64()]
	}
	return string(buf), nil
}
