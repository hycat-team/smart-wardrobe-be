package usecase

import (
	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/modules/identity/application/interface/security"
	uc_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/identity/domain/repositories"
	subscription_contract "smart-wardrobe-be/internal/modules/subscription/presentation/contract"
	"smart-wardrobe-be/internal/shared/application/media"
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
