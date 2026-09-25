package usecase

import (
	"context"

	"smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
)

// SessionDevice carries optional device metadata for a newly issued session.
type SessionDevice struct {
	Name      *string
	IP        *string
	UserAgent *string
}

type ISessionUseCase interface {
	Login(ctx context.Context, input dto.LoginReq) (*dto.TokenRes, error)
	RefreshToken(ctx context.Context, input dto.RefreshTokenReq) (*dto.TokenRes, error)
	Logout(ctx context.Context, input dto.LogoutReq) (bool, error)
	// IssueSession is exposed so Google login can reuse the exact same session
	// issuance path as password login.
	IssueSession(ctx context.Context, user *entities.User, device SessionDevice) (*dto.TokenRes, error)
}
