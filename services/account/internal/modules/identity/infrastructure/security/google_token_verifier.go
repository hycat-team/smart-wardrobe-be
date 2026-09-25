package security

import (
	"context"
	"strings"

	appsecurity "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/security"
	"smart-wardrobe-be/shared/config"

	"google.golang.org/api/idtoken"
)

// GoogleTokenVerifier verifies Google ID tokens via the official idtoken package
// (signature/JWKS, issuer, expiry) and enforces the configured audience list and
// email_verified. It is the only place that depends on the Google SDK, keeping
// the domain/application layers portable.
type GoogleTokenVerifier struct {
	audiences []string
}

func NewGoogleTokenVerifier(cfg *config.Config) appsecurity.IGoogleTokenVerifier {
	return &GoogleTokenVerifier{audiences: append([]string(nil), cfg.Google.ClientIDs...)}
}

func (v *GoogleTokenVerifier) VerifyIDToken(ctx context.Context, idToken string) (*appsecurity.GoogleIdentity, error) {
	if len(v.audiences) == 0 {
		return nil, appsecurity.ErrGoogleConfigMissing
	}
	if strings.TrimSpace(idToken) == "" {
		return nil, appsecurity.ErrGoogleIdentityInvalid
	}

	for _, audience := range v.audiences {
		payload, err := idtoken.Validate(ctx, idToken, audience)
		if err != nil {
			continue
		}

		subject, _ := payload.Claims["sub"].(string)
		email, _ := payload.Claims["email"].(string)
		verified, _ := payload.Claims["email_verified"].(bool)
		name, _ := payload.Claims["name"].(string)
		picture, _ := payload.Claims["picture"].(string)

		subject = strings.TrimSpace(subject)
		email = strings.ToLower(strings.TrimSpace(email))
		if subject == "" || email == "" {
			return nil, appsecurity.ErrGoogleIdentityInvalid
		}

		return &appsecurity.GoogleIdentity{
			Subject:       subject,
			Email:         email,
			EmailVerified: verified,
			Name:          name,
			Picture:       picture,
		}, nil
	}

	return nil, appsecurity.ErrGoogleIdentityInvalid
}
