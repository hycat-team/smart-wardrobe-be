package usecase

import (
	subscription_contract "smart-wardrobe-be/services/account/contract/subscription"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/interface/security"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	shared_event "smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/application/media"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/logger"
)

type UserUseCase struct {
	userRepo              repositories.IUserRepository
	passwordHasher        security.IPasswordHasher
	refreshTokenRepo      repositories.IRefreshTokenRepository
	tokenBlacklistService security.ITokenBlacklistService
	subscriptionContract  subscription_contract.IUserSubscriptionContract
	mediaService          media.IMediaService
	cfg                   *config.Config
	eventPublisher        shared_event.IEventPublisher
	logger                logger.Interface
}

func NewUserUseCase(
	userRepo repositories.IUserRepository,
	passwordHasher security.IPasswordHasher,
	refreshTokenRepo repositories.IRefreshTokenRepository,
	tokenBlacklistService security.ITokenBlacklistService,
	subscriptionContract subscription_contract.IUserSubscriptionContract,
	mediaService media.IMediaService,
	cfg *config.Config,
	eventPublisher shared_event.IEventPublisher,
	log logger.Interface,
) uc_interfaces.IUserUseCase {
	return &UserUseCase{
		userRepo:              userRepo,
		passwordHasher:        passwordHasher,
		refreshTokenRepo:      refreshTokenRepo,
		tokenBlacklistService: tokenBlacklistService,
		subscriptionContract:  subscriptionContract,
		mediaService:          mediaService,
		cfg:                   cfg,
		eventPublisher:        eventPublisher,
		logger:                log,
	}
}
