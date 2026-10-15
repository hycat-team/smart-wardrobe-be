package usecase

import (
	subscription_contract "smart-wardrobe-be/services/account/contract/subscription"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/interface/security"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	shared_event "smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/application/media"
	"smart-wardrobe-be/shared/config"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"
)

type UserUseCase struct {
	userRepo repositories.IUserRepository
	// activityRepo backs the admin activity list (029 R3). The repository is
	// already provided in the wire graph — the admin dashboard reader consumes
	// it through UserRepository — so this is one extra edge, not a new provider.
	activityRepo          repositories.IUserActivityRepository
	passwordHasher        security.IPasswordHasher
	refreshTokenRepo      repositories.IRefreshTokenRepository
	tokenBlacklistService security.ITokenBlacklistService
	subscriptionContract  subscription_contract.IUserSubscriptionContract
	// 024 T196: the account and its entitlement must commit together, so the
	// administrator creation path needs a unit of work and the narrow
	// provisioning contract. Neither is a new repository - it is the existing
	// subscription provisioning body, reached through the module contract.
	provisioningContract subscription_contract.IUserSubscriptionProvisioningContract
	uow                  shared_repos.IUnitOfWork
	mediaService         media.IMediaService
	cfg                  *config.Config
	eventPublisher       shared_event.IEventPublisher
	logger               logger.Interface
}

func NewUserUseCase(
	userRepo repositories.IUserRepository,
	activityRepo repositories.IUserActivityRepository,
	passwordHasher security.IPasswordHasher,
	refreshTokenRepo repositories.IRefreshTokenRepository,
	tokenBlacklistService security.ITokenBlacklistService,
	subscriptionContract subscription_contract.IUserSubscriptionContract,
	provisioningContract subscription_contract.IUserSubscriptionProvisioningContract,
	uow shared_repos.IUnitOfWork,
	mediaService media.IMediaService,
	cfg *config.Config,
	eventPublisher shared_event.IEventPublisher,
	log logger.Interface,
) uc_interfaces.IUserUseCase {
	return &UserUseCase{
		userRepo:              userRepo,
		activityRepo:          activityRepo,
		passwordHasher:        passwordHasher,
		refreshTokenRepo:      refreshTokenRepo,
		tokenBlacklistService: tokenBlacklistService,
		subscriptionContract:  subscriptionContract,
		provisioningContract:  provisioningContract,
		uow:                   uow,
		mediaService:          mediaService,
		cfg:                   cfg,
		eventPublisher:        eventPublisher,
		logger:                log,
	}
}
