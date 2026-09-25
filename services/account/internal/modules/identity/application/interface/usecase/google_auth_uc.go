package usecase

import (
	"context"

	"smart-wardrobe-be/services/account/dto/identity"
)

// GoogleRedirectResult is the outcome of the web OAuth callback. Exactly one of
// Tokens (success) or ErrorCode (redirect-with-error) is set.
type GoogleRedirectResult struct {
	Tokens      *dto.TokenRes
	RedirectURL string
	ErrorCode   string
}

// IGoogleAuthUseCase handles Google login for both mobile (ID token) and web
// (Authorization Code + redirect).
type IGoogleAuthUseCase interface {
	LoginWithIDToken(ctx context.Context, input dto.GoogleLoginReq) (*dto.TokenRes, error)
	// StartRedirect validates the target and returns the Google authorization URL.
	StartRedirect(ctx context.Context, redirectURL string) (string, error)
	// HandleCallback consumes state, exchanges the code and resolves the account.
	// providerError carries Google's `error` query param (e.g. access_denied).
	HandleCallback(ctx context.Context, code string, state string, providerError string) (*GoogleRedirectResult, error)
}
