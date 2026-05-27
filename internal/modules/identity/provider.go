package identity

import (
	"smart-wardrobe-be/internal/modules/identity/application/contract"
	"smart-wardrobe-be/internal/modules/identity/application/usecase"
	"smart-wardrobe-be/internal/modules/identity/infrastructure/caching"
	"smart-wardrobe-be/internal/modules/identity/infrastructure/communication"
	"smart-wardrobe-be/internal/modules/identity/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/identity/infrastructure/security"
	"smart-wardrobe-be/internal/modules/identity/presentation/handler"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewUserRepository,
	persistence.NewRefreshTokenRepository,
	caching.NewRedisOtpService,
	communication.NewGmailSmtpService,
	security.NewBcryptPasswordHasher,
	security.NewRedisTokenBlacklistService,
	contract.NewIdentityModuleContractImpl,
	usecase.NewUserUseCase,
	usecase.NewAuthUseCase,
	handler.NewAuthHandler,
	handler.NewMeHandler,
)
