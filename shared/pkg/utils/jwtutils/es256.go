package jwtutils

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"smart-wardrobe-be/shared/application/constants/jwttype"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// LoadECDSAPrivateKey reads an EC private key (SEC1 or PKCS#8 PEM) for ES256
// signing. Only the identity service holds this key (FR-022).
func LoadECDSAPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read EC private key: %w", err)
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("decode EC private key: invalid PEM block")
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse EC private key: %w", err)
	}
	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("PKCS#8 key is not an ECDSA private key")
	}
	return ecKey, nil
}

// LoadECDSAPublicKey reads an EC public key (PKIX PEM) for ES256 verification.
// Every service except identity holds only this key (FR-022).
func LoadECDSAPublicKey(path string) (*ecdsa.PublicKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read EC public key: %w", err)
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("decode EC public key: invalid PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse EC public key: %w", err)
	}
	ecPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("PKIX key is not an ECDSA public key")
	}
	return ecPub, nil
}

// GenerateTokenES256 signs a token with ES256 (identity only). Claims shape
// matches GenerateToken so verifiers treat both algorithms uniformly.
func GenerateTokenES256(
	userID uuid.UUID,
	email string,
	roleSlug string,
	tokenType jwttype.JwtType,
	privateKey *ecdsa.PrivateKey,
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

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	return token.SignedString(privateKey)
}
