package jwtutils

import (
	"errors"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/application/constants/jwttype"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type CustomClaims struct {
	Email    string          `json:"email"`
	RoleSlug string          `json:"role_slug"`
	Type     jwttype.JwtType `json:"type"`
	jwt.RegisteredClaims
}

func GenerateToken(
	userID uuid.UUID,
	email string,
	roleSlug string,
	tokenType jwttype.JwtType,
	secret string,
	issuer string,
	audience string,
	expiresIn time.Duration,
) (string, error) {
	claims := &CustomClaims{
		Email:    email,
		RoleSlug: roleSlug,
		Type:     tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateToken(secret []byte, tokenString string, expectedTokenType jwttype.JwtType) (*CustomClaims, error) {
	claims := &CustomClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apperror.ErrUnexpectedSigningToken()
		}
		return secret, nil
	})
	if err != nil {
		var ve *jwt.ValidationError
		if errors.As(err, &ve) {
			if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, apperror.ErrInvalidAccessToken()
			}
		}
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
