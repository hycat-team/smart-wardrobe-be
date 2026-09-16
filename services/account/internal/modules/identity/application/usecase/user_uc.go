package usecase

import (
	subscription_contract "smart-wardrobe-be/services/account/contract/subscription"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/interface/security"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/shared/application/media"
	"smart-wardrobe-be/shared/config"
)

type UserUseCase struct {
	userRepo              repositories.IUserRepository
	passwordHasher        security.IPasswordHasher
	refreshTokenRepo      repositories.IRefreshTokenRepository
	tokenBlacklistService security.ITokenBlacklistService
	subscriptionContract  subscription_contract.IUserSubscriptionContract
	mediaService          media.IMediaService
	cfg                   *config.Config
}

func NewUserUseCase(
	userRepo repositories.IUserRepository,
	passwordHasher security.IPasswordHasher,
	refreshTokenRepo repositories.IRefreshTokenRepository,
	tokenBlacklistService security.ITokenBlacklistService,
	subscriptionContract subscription_contract.IUserSubscriptionContract,
	mediaService media.IMediaService,
	cfg *config.Config,
) uc_interfaces.IUserUseCase {
	return &UserUseCase{
		userRepo:              userRepo,
		passwordHasher:        passwordHasher,
		refreshTokenRepo:      refreshTokenRepo,
		tokenBlacklistService: tokenBlacklistService,
		subscriptionContract:  subscriptionContract,
		mediaService:          mediaService,
		cfg:                   cfg,
	}
}
