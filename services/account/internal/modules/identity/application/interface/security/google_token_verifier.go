package security

import (
	"context"
	"errors"
)

// Sentinel errors returned by IGoogleTokenVerifier adapters. The application
// layer maps them to apperror constructors.
var (
	ErrGoogleIdentityInvalid = errors.New("google identity invalid")
	ErrGoogleConfigMissing   = errors.New("google oauth config missing")
)

// GoogleIdentity is the verified, normalized identity extracted from a Google
// ID token.
type GoogleIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
	Picture       string
}

// IGoogleTokenVerifier verifies a Google ID token and returns its identity.
type IGoogleTokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*GoogleIdentity, error)
}
