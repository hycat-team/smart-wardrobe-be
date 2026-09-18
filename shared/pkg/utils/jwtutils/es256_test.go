package jwtutils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"smart-wardrobe-be/shared/application/constants/jwttype"

	"github.com/google/uuid"
)

func es256TestKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestGenerateAndVerifyES256(t *testing.T) {
	key := es256TestKey(t)
	userID := uuid.New()

	signed, err := GenerateTokenES256(userID, "user@example.com", "user", jwttype.AccessToken, key, "iss", "aud", time.Hour)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	claims, err := ValidateTokenFlexible(signed, jwttype.AccessToken, VerifyOptions{
		Algorithm: "ES256",
		PublicKey: &key.PublicKey,
	})
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if claims.Subject != userID.String() {
		t.Fatalf("subject mismatch: %s", claims.Subject)
	}
}

func TestES256RejectedWhenSystemIsHS256(t *testing.T) {
	key := es256TestKey(t)
	signed, err := GenerateTokenES256(uuid.New(), "user@example.com", "user", jwttype.AccessToken, key, "iss", "aud", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	// System configured HS256 must not accept ES256 tokens.
	_, err = ValidateTokenFlexible(signed, jwttype.AccessToken, VerifyOptions{
		Algorithm: "HS256",
		Secret:    []byte("legacy-secret"),
	})
	if err == nil {
		t.Fatal("expected ES256 token to be rejected under HS256 config")
	}
}

func TestHS256LegacyFallback(t *testing.T) {
	secret := []byte("legacy-secret-key-for-transition")
	userID := uuid.New()
	signed, err := GenerateToken(userID, "user@example.com", "user", jwttype.AccessToken, string(secret), "iss", "aud", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	key := es256TestKey(t)

	// ES256 system with legacy secret present still accepts HS256 (transition).
	if _, err := ValidateTokenFlexible(signed, jwttype.AccessToken, VerifyOptions{
		Algorithm: "ES256",
		PublicKey: &key.PublicKey,
		Secret:    secret,
	}); err != nil {
		t.Fatalf("expected HS256 legacy fallback to pass: %v", err)
	}

	// ES256 system without legacy secret rejects HS256 (legacy off).
	if _, err := ValidateTokenFlexible(signed, jwttype.AccessToken, VerifyOptions{
		Algorithm: "ES256",
		PublicKey: &key.PublicKey,
	}); err == nil {
		t.Fatal("expected HS256 token to be rejected when legacy is off")
	}
}

func TestES256RejectsExpiredAndWrongType(t *testing.T) {
	key := es256TestKey(t)

	expired, err := GenerateTokenES256(uuid.New(), "u@e.c", "user", jwttype.AccessToken, key, "iss", "aud", -time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateTokenFlexible(expired, jwttype.AccessToken, VerifyOptions{Algorithm: "ES256", PublicKey: &key.PublicKey}); err == nil {
		t.Fatal("expected expired ES256 token to be rejected")
	}

	refresh, err := GenerateTokenES256(uuid.New(), "u@e.c", "user", jwttype.RefreshToken, key, "iss", "aud", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateTokenFlexible(refresh, jwttype.AccessToken, VerifyOptions{Algorithm: "ES256", PublicKey: &key.PublicKey}); err == nil {
		t.Fatal("expected wrong-type ES256 token to be rejected")
	}
}

func TestLoadECDSAKeysFromPEM(t *testing.T) {
	key := es256TestKey(t)

	privDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER})
	privPath := filepath.Join(t.TempDir(), "ec_priv.pem")
	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		t.Fatal(err)
	}

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	pubPath := filepath.Join(t.TempDir(), "ec_pub.pem")
	if err := os.WriteFile(pubPath, pubPEM, 0600); err != nil {
		t.Fatal(err)
	}

	loadedPriv, err := LoadECDSAPrivateKey(privPath)
	if err != nil {
		t.Fatalf("load private key: %v", err)
	}
	loadedPub, err := LoadECDSAPublicKey(pubPath)
	if err != nil {
		t.Fatalf("load public key: %v", err)
	}

	signed, err := GenerateTokenES256(uuid.New(), "u@e.c", "user", jwttype.AccessToken, loadedPriv, "iss", "aud", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateTokenFlexible(signed, jwttype.AccessToken, VerifyOptions{Algorithm: "ES256", PublicKey: loadedPub}); err != nil {
		t.Fatalf("round-trip with loaded keys failed: %v", err)
	}

	if _, err := LoadECDSAPrivateKey(filepath.Join(t.TempDir(), "missing.pem")); err == nil {
		t.Fatal("expected error for missing private key file")
	}
}
