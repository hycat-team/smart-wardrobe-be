package jwtutils

import (
	"crypto/ecdsa"
	"strings"

	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/application/constants/jwttype"

	"github.com/golang-jwt/jwt/v4"
)

// VerifyOptions selects the token verification scheme (T047, FR-022/FR-023).
// ES256 is primary when Algorithm is "ES256"; a non-empty Secret additionally
// enables the transitional HS256 legacy fallback for HS256-signed tokens.
type VerifyOptions struct {
	Algorithm string
	PublicKey *ecdsa.PublicKey
	Secret    []byte
}

// ValidateTokenFlexible verifies a token under the configured scheme. It
// peeks at the token header to dispatch ES256 vs HS256, so mixed fleets keep
// working during the asymmetric migration.
func ValidateTokenFlexible(tokenString string, expectedTokenType jwttype.JwtType, opts VerifyOptions) (*CustomClaims, error) {
	parser := jwt.NewParser()
	unverified, _, err := parser.ParseUnverified(tokenString, &CustomClaims{})
	if err != nil {
		return nil, apperror.ErrInvalidAccessToken()
	}
	alg := ""
	if unverified.Method != nil {
		alg = unverified.Method.Alg()
	}

	wantES256 := strings.EqualFold(strings.TrimSpace(opts.Algorithm), "ES256")

	switch alg {
	case "ES256":
		if !wantES256 {
			return nil, apperror.ErrUnexpectedSigningToken()
		}
		if opts.PublicKey == nil {
			return nil, apperror.NewInternalError("Thiếu khóa công khai để xác thực phiên làm việc.")
		}
		return validateES256Token(tokenString, expectedTokenType, opts.PublicKey)
	case "HS256":
		if wantES256 && len(opts.Secret) == 0 {
			return nil, apperror.ErrUnexpectedSigningToken()
		}
		return ValidateToken(opts.Secret, tokenString, expectedTokenType)
	default:
		return nil, apperror.ErrUnexpectedSigningToken()
	}
}

func validateES256Token(tokenString string, expectedTokenType jwttype.JwtType, publicKey *ecdsa.PublicKey) (*CustomClaims, error) {
	claims := &CustomClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, apperror.ErrUnexpectedSigningToken()
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, apperror.ErrInvalidAccessToken()
	}

	if !token.Valid {
		return nil, apperror.ErrInvalidToken()
	}

	if claims.Type != expectedTokenType {
		return nil, apperror.ErrInvalidToken()
	}

	return claims, nil
}
